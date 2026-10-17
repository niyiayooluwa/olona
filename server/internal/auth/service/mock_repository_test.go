package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"server/internal/auth/config"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service/oauth"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

// mockAuthRepository is a test double that satisfies repository.AuthRepository.
type mockAuthRepository struct {
	// User
	CreateUserFn         func(ctx context.Context, email string, passwordHash *string) (db.User, error)
	GetUserByIDFn        func(ctx context.Context, id uuid.UUID) (db.User, error)
	GetUserByEmailFn     func(ctx context.Context, email string) (db.User, error)
	UpdateUserPasswordFn func(ctx context.Context, id uuid.UUID, passwordHash *string) error
	MarkEmailVerifiedFn  func(ctx context.Context, id uuid.UUID) error
	SuspendUserFn        func(ctx context.Context, id uuid.UUID) error
	UnsuspendUserFn      func(ctx context.Context, id uuid.UUID) error
	DeactivateUserFn     func(ctx context.Context, id uuid.UUID) error
	DeleteUserFn         func(ctx context.Context, id uuid.UUID) error

	// Profile
	CreateProfileFn       func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error)
	GetProfileByUserIDFn  func(ctx context.Context, userID uuid.UUID) (db.Profile, error)
	UpdateProfileFn       func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error)
	CheckUsernameExistsFn func(ctx context.Context, username string) (bool, error)
	DeleteProfileFn       func(ctx context.Context, userID uuid.UUID) error

	// OAuth
	GetOAuthConnectionFn       func(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error)
	GetUserByOAuthProviderFn   func(ctx context.Context, provider, providerUserID string) (db.User, error)
	CreateOAuthConnectionFn    func(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error)
	DeleteOAuthConnectionFn    func(ctx context.Context, userID uuid.UUID, provider string) error
	GetOAuthProvidersForUserFn func(ctx context.Context, userID uuid.UUID) ([]string, error)

	// Refresh Tokens
	CreateRefreshTokenFn         func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error)
	GetRefreshTokenFn            func(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshTokenFn         func(ctx context.Context, id uuid.UUID) error
	RevokeAllUserRefreshTokensFn func(ctx context.Context, userID uuid.UUID) error
	DeleteRefreshTokenFn         func(ctx context.Context, id uuid.UUID) error

	// OTP
	CreateOTPFn            func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error)
	GetValidOTPFn          func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error)
	GetValidOTPForUpdateFn func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error)
	IncrementOTPAttemptsFn func(ctx context.Context, id uuid.UUID) (int32, error)
	MarkOTPUsedFn          func(ctx context.Context, id uuid.UUID) error
	DeleteOTPFn            func(ctx context.Context, id uuid.UUID) error

	// Invitations
	CreateInvitationFn       func(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error)
	GetInvitationByIDFn      func(ctx context.Context, id uuid.UUID) (db.Invitation, error)
	GetInvitationByTokenFn   func(ctx context.Context, tokenHash string) (db.Invitation, error)
	MarkInvitationAcceptedFn func(ctx context.Context, id uuid.UUID) error
	DeleteInvitationFn       func(ctx context.Context, id uuid.UUID) error

	// Transactions
	// WithTxHookFn overrides the default WithTx behaviour so a test can
	// simulate a rollback, a commit failure, or a repository with no
	// transaction source configured.
	WithTxHookFn func(ctx context.Context, fn func(repository.AuthRepository) error) error
}

// User implementations
func (mock *mockAuthRepository) CreateUser(ctx context.Context, email string, passwordHash *string) (db.User, error) {
	if mock.CreateUserFn != nil {
		return mock.CreateUserFn(ctx, email, passwordHash)
	}
	return db.User{}, errors.New("CreateUser not implemented")
}

func (mock *mockAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if mock.GetUserByIDFn != nil {
		return mock.GetUserByIDFn(ctx, id)
	}
	return db.User{}, errors.New("GetUserByID not implemented")
}

func (mock *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if mock.GetUserByEmailFn != nil {
		return mock.GetUserByEmailFn(ctx, email)
	}
	return db.User{}, errors.New("GetUserByEmail not implemented")
}

func (mock *mockAuthRepository) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash *string) error {
	if mock.UpdateUserPasswordFn != nil {
		return mock.UpdateUserPasswordFn(ctx, id, passwordHash)
	}
	return errors.New("UpdateUserPassword not implemented")
}

func (mock *mockAuthRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	if mock.MarkEmailVerifiedFn != nil {
		return mock.MarkEmailVerifiedFn(ctx, id)
	}
	return errors.New("MarkEmailVerified not implemented")
}

func (mock *mockAuthRepository) SuspendUser(ctx context.Context, id uuid.UUID) error {
	if mock.SuspendUserFn != nil {
		return mock.SuspendUserFn(ctx, id)
	}
	return errors.New("SuspendUser not implemented")
}

func (mock *mockAuthRepository) UnsuspendUser(ctx context.Context, id uuid.UUID) error {
	if mock.UnsuspendUserFn != nil {
		return mock.UnsuspendUserFn(ctx, id)
	}
	return errors.New("UnsuspendUser not implemented")
}

func (mock *mockAuthRepository) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	if mock.DeactivateUserFn != nil {
		return mock.DeactivateUserFn(ctx, id)
	}
	return errors.New("DeactivateUser not implemented")
}

func (mock *mockAuthRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if mock.DeleteUserFn != nil {
		return mock.DeleteUserFn(ctx, id)
	}
	return errors.New("DeleteUser not implemented")
}

// Profile implementations
func (mock *mockAuthRepository) CreateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
	if mock.CreateProfileFn != nil {
		return mock.CreateProfileFn(ctx, userID, firstName, lastName, username, displayName, profileURL, avatarURL)
	}
	return db.Profile{}, errors.New("CreateProfile not implemented")
}

func (mock *mockAuthRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
	if mock.GetProfileByUserIDFn != nil {
		return mock.GetProfileByUserIDFn(ctx, userID)
	}
	return db.Profile{}, errors.New("GetProfileByUserID not implemented")
}

func (mock *mockAuthRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
	if mock.UpdateProfileFn != nil {
		return mock.UpdateProfileFn(ctx, userID, firstName, lastName, username, displayName, profileURL, avatarURL)
	}
	return db.Profile{}, errors.New("UpdateProfile not implemented")
}

func (mock *mockAuthRepository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	if mock.CheckUsernameExistsFn != nil {
		return mock.CheckUsernameExistsFn(ctx, username)
	}
	return false, errors.New("CheckUsernameExists not implemented")
}

func (mock *mockAuthRepository) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	if mock.DeleteProfileFn != nil {
		return mock.DeleteProfileFn(ctx, userID)
	}
	return errors.New("DeleteProfile not implemented")
}

// OAuth implementations
func (mock *mockAuthRepository) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error) {
	if mock.GetOAuthConnectionFn != nil {
		return mock.GetOAuthConnectionFn(ctx, provider, providerUserID)
	}
	return db.OauthConnection{}, errors.New("GetOAuthConnection not implemented")
}

func (mock *mockAuthRepository) GetUserByOAuthProvider(ctx context.Context, provider, providerUserID string) (db.User, error) {
	if mock.GetUserByOAuthProviderFn != nil {
		return mock.GetUserByOAuthProviderFn(ctx, provider, providerUserID)
	}
	return db.User{}, errors.New("GetUserByOAuthProvider not implemented")
}

func (mock *mockAuthRepository) CreateOAuthConnection(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error) {
	if mock.CreateOAuthConnectionFn != nil {
		return mock.CreateOAuthConnectionFn(ctx, userID, provider, providerUserID)
	}
	return db.OauthConnection{}, errors.New("CreateOAuthConnection not implemented")
}

func (mock *mockAuthRepository) DeleteOAuthConnection(ctx context.Context, userID uuid.UUID, provider string) error {
	if mock.DeleteOAuthConnectionFn != nil {
		return mock.DeleteOAuthConnectionFn(ctx, userID, provider)
	}
	return errors.New("DeleteOAuthConnection not implemented")
}

func (mock *mockAuthRepository) GetOAuthProvidersForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	if mock.GetOAuthProvidersForUserFn != nil {
		return mock.GetOAuthProvidersForUserFn(ctx, userID)
	}
	return nil, errors.New("GetOAuthProvidersForUser not implemented")
}

// Refresh Token implementations
func (mock *mockAuthRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	if mock.CreateRefreshTokenFn != nil {
		return mock.CreateRefreshTokenFn(ctx, userID, tokenHash, expiresAt)
	}
	return db.RefreshToken{}, errors.New("CreateRefreshToken not implemented")
}

func (mock *mockAuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	if mock.GetRefreshTokenFn != nil {
		return mock.GetRefreshTokenFn(ctx, tokenHash)
	}
	return db.RefreshToken{}, errors.New("GetRefreshToken not implemented")
}

func (mock *mockAuthRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if mock.RevokeRefreshTokenFn != nil {
		return mock.RevokeRefreshTokenFn(ctx, id)
	}
	return errors.New("RevokeRefreshToken not implemented")
}

func (mock *mockAuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	if mock.RevokeAllUserRefreshTokensFn != nil {
		return mock.RevokeAllUserRefreshTokensFn(ctx, userID)
	}
	return errors.New("RevokeAllUserRefreshTokens not implemented")
}

func (mock *mockAuthRepository) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	if mock.DeleteRefreshTokenFn != nil {
		return mock.DeleteRefreshTokenFn(ctx, id)
	}
	return errors.New("DeleteRefreshToken not implemented")
}

// OTP implementations
func (mock *mockAuthRepository) CreateOTP(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
	if mock.CreateOTPFn != nil {
		return mock.CreateOTPFn(ctx, userID, codeHash, purpose, expiresAt)
	}
	return db.OtpCode{}, errors.New("CreateOTP not implemented")
}

func (mock *mockAuthRepository) GetValidOTP(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
	if mock.GetValidOTPFn != nil {
		return mock.GetValidOTPFn(ctx, userID, purpose)
	}
	return db.OtpCode{}, errors.New("GetValidOTP not implemented")
}

func (mock *mockAuthRepository) GetValidOTPForUpdate(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
	if mock.GetValidOTPForUpdateFn != nil {
		return mock.GetValidOTPForUpdateFn(ctx, userID, purpose)
	}
	return db.OtpCode{}, errors.New("GetValidOTPForUpdate not implemented")
}

func (mock *mockAuthRepository) IncrementOTPAttempts(ctx context.Context, id uuid.UUID) (int32, error) {
	if mock.IncrementOTPAttemptsFn != nil {
		return mock.IncrementOTPAttemptsFn(ctx, id)
	}
	return 0, errors.New("IncrementOTPAttempts not implemented")
}

func (mock *mockAuthRepository) MarkOTPUsed(ctx context.Context, id uuid.UUID) error {
	if mock.MarkOTPUsedFn != nil {
		return mock.MarkOTPUsedFn(ctx, id)
	}
	return errors.New("MarkOTPUsed not implemented")
}

func (mock *mockAuthRepository) DeleteOTP(ctx context.Context, id uuid.UUID) error {
	if mock.DeleteOTPFn != nil {
		return mock.DeleteOTPFn(ctx, id)
	}
	return errors.New("DeleteOTP not implemented")
}

// Invitation implementations
func (mock *mockAuthRepository) CreateInvitation(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error) {
	if mock.CreateInvitationFn != nil {
		return mock.CreateInvitationFn(ctx, email, tokenHash, invitedBy, expiresAt)
	}
	return db.Invitation{}, errors.New("CreateInvitation not implemented")
}

func (mock *mockAuthRepository) GetInvitationByID(ctx context.Context, id uuid.UUID) (db.Invitation, error) {
	if mock.GetInvitationByIDFn != nil {
		return mock.GetInvitationByIDFn(ctx, id)
	}
	return db.Invitation{}, errors.New("GetInvitationByID not implemented")
}

func (mock *mockAuthRepository) GetInvitationByToken(ctx context.Context, tokenHash string) (db.Invitation, error) {
	if mock.GetInvitationByTokenFn != nil {
		return mock.GetInvitationByTokenFn(ctx, tokenHash)
	}
	return db.Invitation{}, errors.New("GetInvitationByToken not implemented")
}

func (mock *mockAuthRepository) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	if mock.MarkInvitationAcceptedFn != nil {
		return mock.MarkInvitationAcceptedFn(ctx, id)
	}
	return errors.New("MarkInvitationAccepted not implemented")
}

func (mock *mockAuthRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	if mock.DeleteInvitationFn != nil {
		return mock.DeleteInvitationFn(ctx, id)
	}
	return errors.New("DeleteInvitation not implemented")
}

// WithTx runs fn and returns its error, as a committed transaction would.
//
// Defaulting to the mock itself keeps existing tests working without having to
// stub the transaction boundary. Tests that care about atomicity should set
// WithTxFn so the outcome is explicit.
func (mock *mockAuthRepository) WithTx(ctx context.Context, fn func(repository.AuthRepository) error) error {
	if mock.WithTxHookFn != nil {
		return mock.WithTxHookFn(ctx, fn)
	}
	return fn(mock)
}

// Compile-time check
var _ repository.AuthRepository = (*mockAuthRepository)(nil)

// mockMailer is a test double that satisfies model.Mailer.
type mockMailer struct {
	SendOTPFn           func(ctx context.Context, email, code string) error
	SendInviteFn        func(ctx context.Context, email, token string) error
	SendPasswordResetFn func(ctx context.Context, email, code string) error
}

func (mock *mockMailer) SendOTP(ctx context.Context, email, code string) error {
	if mock.SendOTPFn != nil {
		return mock.SendOTPFn(ctx, email, code)
	}
	return nil
}

func (mock *mockMailer) SendInvite(ctx context.Context, email, token string) error {
	if mock.SendInviteFn != nil {
		return mock.SendInviteFn(ctx, email, token)
	}
	return nil
}

func (mock *mockMailer) SendPasswordReset(ctx context.Context, email, code string) error {
	if mock.SendPasswordResetFn != nil {
		return mock.SendPasswordResetFn(ctx, email, code)
	}
	return nil
}

var _ model.Mailer = (*mockMailer)(nil)

// mockOAuthProvider is a test double for oauth.Provider.
type mockOAuthProvider struct {
	VerifyTokenFn func(ctx context.Context, idToken string) (*oauth.Identity, error)
}

func (mock *mockOAuthProvider) VerifyToken(ctx context.Context, idToken string) (*oauth.Identity, error) {
	if mock.VerifyTokenFn != nil {
		return mock.VerifyTokenFn(ctx, idToken)
	}
	return nil, errors.New("VerifyToken not implemented")
}

var _ oauth.Provider = (*mockOAuthProvider)(nil)

// helper to setup a test JWTIssuer
func setupTestJWTIssuer() *utils.JWTIssuer {
	return utils.NewJWTIssuer("test-super-secret-key-32-chars-long!")
}

// helper to setup default AuthConfig
func setupTestConfig() config.AuthConfig {
	return config.DefaultConfig()
}
