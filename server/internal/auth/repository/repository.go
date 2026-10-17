// Package repository provides the persistence layer for the auth domain.
// It wraps the sqlc-generated *db.Queries and translates db-level errors
// (such as pgx.ErrNoRows, 42501, 23505) into domain-safe sentinel errors.
// Higher layers (service/handler) should never import the db package directly.
package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "server/internal/db/generated"
)

// AuthRepository defines the persistence contract for all authentication operations.
//
// WithTx runs a unit of work atomically. Operations that must not be split by a
// concurrent request — OTP verification (which relies on SELECT ... FOR UPDATE
// to serialise guesses), signup, invite redemption and account deletion — have
// to go through it, because in autocommit mode a row lock is released the
// instant its statement finishes and buys nothing.
type AuthRepository interface {
	// ===================================
	// USER
	// ===================================
	CreateUser(ctx context.Context, email string, passwordHash *string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash *string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID) error
	SuspendUser(ctx context.Context, id uuid.UUID) error
	UnsuspendUser(ctx context.Context, id uuid.UUID) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// ===================================
	// PROFILE
	// ===================================
	CreateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error)
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (db.Profile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error)
	CheckUsernameExists(ctx context.Context, username string) (bool, error)
	DeleteProfile(ctx context.Context, userID uuid.UUID) error

	// ===================================
	// OAUTH
	// ===================================
	GetOAuthConnection(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error)
	GetUserByOAuthProvider(ctx context.Context, provider, providerUserID string) (db.User, error)
	CreateOAuthConnection(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error)
	DeleteOAuthConnection(ctx context.Context, userID uuid.UUID, provider string) error
	GetOAuthProvidersForUser(ctx context.Context, userID uuid.UUID) ([]string, error)

	// ===================================
	// REFRESH TOKENS
	// ===================================
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error

	// ===================================
	// OTP
	// ===================================
	CreateOTP(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error)
	GetValidOTP(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error)
	GetValidOTPForUpdate(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error)
	IncrementOTPAttempts(ctx context.Context, id uuid.UUID) (int32, error)
	MarkOTPUsed(ctx context.Context, id uuid.UUID) error
	DeleteOTP(ctx context.Context, id uuid.UUID) error

	// ===================================
	// INVITATIONS
	// ===================================
	CreateInvitation(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error)
	GetInvitationByID(ctx context.Context, id uuid.UUID) (db.Invitation, error)
	GetInvitationByToken(ctx context.Context, tokenHash string) (db.Invitation, error)
	MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error

	// ===================================
	// TRANSACTIONS
	// ===================================

	// WithTx runs fn inside a single database transaction, committing when fn
	// returns nil and rolling back when it returns an error or panics.
	//
	// The repo handed to fn is backed by the transaction, so every call fn
	// makes runs inside it. fn must use that repo, not the receiver.
	//
	// Why this exists: SELECT ... FOR UPDATE only holds its row lock until the
	// end of the enclosing transaction. In autocommit mode each statement is
	// its own transaction, so the lock is released the moment the SELECT
	// returns and a concurrent request reads the row before the first one has
	// written to it. Anything that reads a row to make a decision and then
	// writes based on it — counting OTP attempts, redeeming a code, deleting a
	// user and their profile — has to run here or the decision is made against
	// stale state.
	WithTx(ctx context.Context, fn func(AuthRepository) error) error
}

// ErrTransactionsUnsupported is returned by WithTx when the repository was
// built without a transaction source. Production wiring must supply one via
// WithTxSource; tests that never exercise WithTx can omit it.
var ErrTransactionsUnsupported = errors.New(
	"repository: transactions are not configured; pass WithTxSource when constructing the repository",
)

// TxBeginner opens a database transaction. *pgxpool.Pool satisfies it.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// RepositoryOption customises a repository at construction time.
type RepositoryOption func(*authRepositoryImpl)

// WithTxSource enables WithTx on the repository by supplying the pool or
// connection that transactions are started from.
func WithTxSource(beginner TxBeginner) RepositoryOption {
	return func(repo *authRepositoryImpl) {
		repo.beginner = beginner
	}
}

type authRepositoryImpl struct {
	queries  db.Querier
	beginner TxBeginner
}

// NewAuthRepository creates a new AuthRepository instance backed by db.Querier.
//
// Without the WithTxSource option the repository still works, but WithTx
// returns ErrTransactionsUnsupported — pass it in anything that serves traffic.
func NewAuthRepository(queries db.Querier, opts ...RepositoryOption) AuthRepository {
	repo := &authRepositoryImpl{queries: queries}
	for _, opt := range opts {
		opt(repo)
	}
	return repo
}

// WithTx runs fn in a transaction, committing on success and rolling back on
// error or panic.
func (repo *authRepositoryImpl) WithTx(ctx context.Context, fn func(AuthRepository) error) error {
	if repo.beginner == nil {
		return ErrTransactionsUnsupported
	}

	tx, err := repo.beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}

	// Rollback after a successful commit is a no-op that reports
	// ErrTxClosed, which pgx makes safe to ignore. Rolling back on panic is
	// not optional: an aborted handler must not leave locks held.
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				log.Printf("auth repository: rollback failed: %v", rbErr)
			}
		}
	}()

	if err := fn(&authRepositoryImpl{queries: db.New(tx)}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit transaction: %w", err)
	}
	committed = true
	return nil
}

// ===================================
// USER
// ===================================

func (repo *authRepositoryImpl) CreateUser(ctx context.Context, email string, passwordHash *string) (db.User, error) {
	user, err := repo.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: toText(passwordHash),
	})
	return user, wrap(err)
}

func (repo *authRepositoryImpl) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, err := repo.queries.GetUserByID(ctx, id)
	return user, wrap(err)
}

func (repo *authRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	user, err := repo.queries.GetUserByEmail(ctx, email)
	return user, wrap(err)
}

func (repo *authRepositoryImpl) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash *string) error {
	err := repo.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: toText(passwordHash),
	})
	return wrap(err)
}

func (repo *authRepositoryImpl) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.MarkEmailVerified(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) SuspendUser(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.SuspendUser(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) UnsuspendUser(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.UnsuspendUser(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.DeactivateUser(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.DeleteUser(ctx, id)
	return wrap(err)
}

// ===================================
// PROFILE
// ===================================

func (repo *authRepositoryImpl) CreateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
	profile, err := repo.queries.CreateProfile(ctx, db.CreateProfileParams{
		UserID:      userID,
		FirstName:   toText(firstName),
		LastName:    toText(lastName),
		Username:    toText(username),
		DisplayName: toText(displayName),
		ProfileUrl:  toText(profileURL),
		AvatarUrl:   toText(avatarURL),
	})
	return profile, wrap(err)
}

func (repo *authRepositoryImpl) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
	profile, err := repo.queries.GetProfileByUserID(ctx, userID)
	return profile, wrap(err)
}

func (repo *authRepositoryImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
	profile, err := repo.queries.UpdateProfile(ctx, db.UpdateProfileParams{
		UserID:      userID,
		FirstName:   toText(firstName),
		LastName:    toText(lastName),
		Username:    toText(username),
		DisplayName: toText(displayName),
		ProfileUrl:  toText(profileURL),
		AvatarUrl:   toText(avatarURL),
	})
	return profile, wrap(err)
}

func (repo *authRepositoryImpl) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	exists, err := repo.queries.CheckUsernameExists(ctx, pgtype.Text{String: username, Valid: true})
	return exists, wrap(err)
}

func (repo *authRepositoryImpl) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	err := repo.queries.DeleteProfile(ctx, userID)
	return wrap(err)
}

// ===================================
// OAUTH
// ===================================

func (repo *authRepositoryImpl) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error) {
	connection, err := repo.queries.GetOAuthConnection(ctx, db.GetOAuthConnectionParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	return connection, wrap(err)
}

func (repo *authRepositoryImpl) GetUserByOAuthProvider(ctx context.Context, provider, providerUserID string) (db.User, error) {
	user, err := repo.queries.GetUserByOAuthProvider(ctx, db.GetUserByOAuthProviderParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	return user, wrap(err)
}

func (repo *authRepositoryImpl) CreateOAuthConnection(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error) {
	connection, err := repo.queries.CreateOAuthConnection(ctx, db.CreateOAuthConnectionParams{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	return connection, wrap(err)
}

func (repo *authRepositoryImpl) DeleteOAuthConnection(ctx context.Context, userID uuid.UUID, provider string) error {
	err := repo.queries.DeleteOAuthConnection(ctx, db.DeleteOAuthConnectionParams{
		UserID:   userID,
		Provider: provider,
	})
	return wrap(err)
}

func (repo *authRepositoryImpl) GetOAuthProvidersForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	providers, err := repo.queries.GetOAuthProvidersForUser(ctx, userID)
	return providers, wrap(err)
}

// ===================================
// REFRESH TOKENS
// ===================================

func (repo *authRepositoryImpl) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	token, err := repo.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	return token, wrap(err)
}

func (repo *authRepositoryImpl) GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	token, err := repo.queries.GetRefreshToken(ctx, tokenHash)
	return token, wrap(err)
}

func (repo *authRepositoryImpl) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.RevokeRefreshToken(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	err := repo.queries.RevokeAllUserRefreshTokens(ctx, userID)
	return wrap(err)
}

func (repo *authRepositoryImpl) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.DeleteRefreshToken(ctx, id)
	return wrap(err)
}

// ===================================
// OTP
// ===================================

func (repo *authRepositoryImpl) CreateOTP(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
	otpCode, err := repo.queries.CreateOTP(ctx, db.CreateOTPParams{
		UserID:    userID,
		CodeHash:  codeHash,
		Purpose:   purpose,
		ExpiresAt: expiresAt,
	})
	return otpCode, wrap(err)
}

func (repo *authRepositoryImpl) GetValidOTP(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
	otpCode, err := repo.queries.GetValidOTP(ctx, db.GetValidOTPParams{
		UserID:  userID,
		Purpose: purpose,
	})
	return otpCode, wrap(err)
}

func (repo *authRepositoryImpl) GetValidOTPForUpdate(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
	otpCode, err := repo.queries.GetValidOTPForUpdate(ctx, db.GetValidOTPForUpdateParams{
		UserID:  userID,
		Purpose: purpose,
	})
	return otpCode, wrap(err)
}

func (repo *authRepositoryImpl) IncrementOTPAttempts(ctx context.Context, id uuid.UUID) (int32, error) {
	attempts, err := repo.queries.IncrementOTPAttempts(ctx, id)
	return attempts, wrap(err)
}

func (repo *authRepositoryImpl) MarkOTPUsed(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.MarkOTPUsed(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) DeleteOTP(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.DeleteOTP(ctx, id)
	return wrap(err)
}

// ===================================
// INVITATIONS
// ===================================

func (repo *authRepositoryImpl) CreateInvitation(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error) {
	invitation, err := repo.queries.CreateInvitation(ctx, db.CreateInvitationParams{
		Email:     email,
		TokenHash: tokenHash,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
	})
	return invitation, wrap(err)
}

func (repo *authRepositoryImpl) GetInvitationByID(ctx context.Context, id uuid.UUID) (db.Invitation, error) {
	invitation, err := repo.queries.GetInvitationByID(ctx, id)
	return invitation, wrap(err)
}

func (repo *authRepositoryImpl) GetInvitationByToken(ctx context.Context, tokenHash string) (db.Invitation, error) {
	invitation, err := repo.queries.GetInvitationByToken(ctx, tokenHash)
	return invitation, wrap(err)
}

func (repo *authRepositoryImpl) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.MarkInvitationAccepted(ctx, id)
	return wrap(err)
}

func (repo *authRepositoryImpl) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	err := repo.queries.DeleteInvitation(ctx, id)
	return wrap(err)
}

// ===================================
// HELPERS & ERROR HANDLING
// ===================================

// Sentinel errors exposed by the repository package.
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("record already exists")
	ErrPermissionDenied = errors.New("insufficient database privilege")
	ErrForeignKey       = errors.New("referenced record does not exist")
)

// wrap normalizes database-level errors (pgx, pgconn) into domain-safe sentinel errors.
func wrap(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501": // insufficient_privilege
			return ErrPermissionDenied
		case "23505": // unique_violation
			return ErrAlreadyExists
		case "23503": // foreign_key_violation
			return ErrForeignKey
		}
	}

	return err
}

func toText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}
