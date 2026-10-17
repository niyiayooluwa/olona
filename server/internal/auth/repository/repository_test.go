package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "server/internal/db/generated"
)

// fakeQuerier is a test double that satisfies db.Querier.
type fakeQuerier struct {
	db.Querier

	createUserFn         func(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	getUserByIDFn        func(ctx context.Context, id uuid.UUID) (db.User, error)
	getUserByEmailFn     func(ctx context.Context, email string) (db.User, error)
	updateUserPasswordFn func(ctx context.Context, arg db.UpdateUserPasswordParams) error
	markEmailVerifiedFn  func(ctx context.Context, id uuid.UUID) error
	suspendUserFn        func(ctx context.Context, id uuid.UUID) error
	unsuspendUserFn      func(ctx context.Context, id uuid.UUID) error
	deactivateUserFn     func(ctx context.Context, id uuid.UUID) error
	deleteUserFn         func(ctx context.Context, id uuid.UUID) error

	createProfileFn       func(ctx context.Context, arg db.CreateProfileParams) (db.Profile, error)
	getProfileByUserIDFn  func(ctx context.Context, userID uuid.UUID) (db.Profile, error)
	updateProfileFn       func(ctx context.Context, arg db.UpdateProfileParams) (db.Profile, error)
	checkUsernameExistsFn func(ctx context.Context, username pgtype.Text) (bool, error)
	deleteProfileFn       func(ctx context.Context, userID uuid.UUID) error

	getOAuthConnectionFn       func(ctx context.Context, arg db.GetOAuthConnectionParams) (db.OauthConnection, error)
	getUserByOAuthProviderFn   func(ctx context.Context, arg db.GetUserByOAuthProviderParams) (db.User, error)
	createOAuthConnectionFn    func(ctx context.Context, arg db.CreateOAuthConnectionParams) (db.OauthConnection, error)
	deleteOAuthConnectionFn    func(ctx context.Context, arg db.DeleteOAuthConnectionParams) error
	getOAuthProvidersForUserFn func(ctx context.Context, userID uuid.UUID) ([]string, error)

	createRefreshTokenFn         func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error)
	getRefreshTokenFn            func(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	revokeRefreshTokenFn         func(ctx context.Context, id uuid.UUID) error
	revokeAllUserRefreshTokensFn func(ctx context.Context, userID uuid.UUID) error
	deleteRefreshTokenFn         func(ctx context.Context, id uuid.UUID) error

	createOTPFn            func(ctx context.Context, arg db.CreateOTPParams) (db.OtpCode, error)
	getValidOTPFn          func(ctx context.Context, arg db.GetValidOTPParams) (db.OtpCode, error)
	getValidOTPForUpdateFn func(ctx context.Context, arg db.GetValidOTPForUpdateParams) (db.OtpCode, error)
	incrementOTPAttemptsFn func(ctx context.Context, id uuid.UUID) (int32, error)
	markOTPUsedFn          func(ctx context.Context, id uuid.UUID) error
	deleteOTPFn            func(ctx context.Context, id uuid.UUID) error

	createInvitationFn       func(ctx context.Context, arg db.CreateInvitationParams) (db.Invitation, error)
	getInvitationByIDFn      func(ctx context.Context, id uuid.UUID) (db.Invitation, error)
	getInvitationByTokenFn   func(ctx context.Context, tokenHash string) (db.Invitation, error)
	markInvitationAcceptedFn func(ctx context.Context, id uuid.UUID) error
	deleteInvitationFn       func(ctx context.Context, id uuid.UUID) error
}

func (f *fakeQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if f.createUserFn != nil {
		return f.createUserFn(ctx, arg)
	}
	return db.User{}, nil
}

func (f *fakeQuerier) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if f.getUserByIDFn != nil {
		return f.getUserByIDFn(ctx, id)
	}
	return db.User{}, nil
}

func (f *fakeQuerier) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if f.getUserByEmailFn != nil {
		return f.getUserByEmailFn(ctx, email)
	}
	return db.User{}, nil
}

func (f *fakeQuerier) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	if f.updateUserPasswordFn != nil {
		return f.updateUserPasswordFn(ctx, arg)
	}
	return nil
}

func (f *fakeQuerier) MarkEmailVerified(ctx context.Context, id uuid.UUID) error {
	if f.markEmailVerifiedFn != nil {
		return f.markEmailVerifiedFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) SuspendUser(ctx context.Context, id uuid.UUID) error {
	if f.suspendUserFn != nil {
		return f.suspendUserFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) UnsuspendUser(ctx context.Context, id uuid.UUID) error {
	if f.unsuspendUserFn != nil {
		return f.unsuspendUserFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	if f.deactivateUserFn != nil {
		return f.deactivateUserFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if f.deleteUserFn != nil {
		return f.deleteUserFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) CreateProfile(ctx context.Context, arg db.CreateProfileParams) (db.Profile, error) {
	if f.createProfileFn != nil {
		return f.createProfileFn(ctx, arg)
	}
	return db.Profile{}, nil
}

func (f *fakeQuerier) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
	if f.getProfileByUserIDFn != nil {
		return f.getProfileByUserIDFn(ctx, userID)
	}
	return db.Profile{}, nil
}

func (f *fakeQuerier) UpdateProfile(ctx context.Context, arg db.UpdateProfileParams) (db.Profile, error) {
	if f.updateProfileFn != nil {
		return f.updateProfileFn(ctx, arg)
	}
	return db.Profile{}, nil
}

func (f *fakeQuerier) CheckUsernameExists(ctx context.Context, username pgtype.Text) (bool, error) {
	if f.checkUsernameExistsFn != nil {
		return f.checkUsernameExistsFn(ctx, username)
	}
	return false, nil
}

func (f *fakeQuerier) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	if f.deleteProfileFn != nil {
		return f.deleteProfileFn(ctx, userID)
	}
	return nil
}

func (f *fakeQuerier) GetOAuthConnection(ctx context.Context, arg db.GetOAuthConnectionParams) (db.OauthConnection, error) {
	if f.getOAuthConnectionFn != nil {
		return f.getOAuthConnectionFn(ctx, arg)
	}
	return db.OauthConnection{}, nil
}

func (f *fakeQuerier) GetUserByOAuthProvider(ctx context.Context, arg db.GetUserByOAuthProviderParams) (db.User, error) {
	if f.getUserByOAuthProviderFn != nil {
		return f.getUserByOAuthProviderFn(ctx, arg)
	}
	return db.User{}, nil
}

func (f *fakeQuerier) CreateOAuthConnection(ctx context.Context, arg db.CreateOAuthConnectionParams) (db.OauthConnection, error) {
	if f.createOAuthConnectionFn != nil {
		return f.createOAuthConnectionFn(ctx, arg)
	}
	return db.OauthConnection{}, nil
}

func (f *fakeQuerier) DeleteOAuthConnection(ctx context.Context, arg db.DeleteOAuthConnectionParams) error {
	if f.deleteOAuthConnectionFn != nil {
		return f.deleteOAuthConnectionFn(ctx, arg)
	}
	return nil
}

func (f *fakeQuerier) GetOAuthProvidersForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	if f.getOAuthProvidersForUserFn != nil {
		return f.getOAuthProvidersForUserFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeQuerier) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	if f.createRefreshTokenFn != nil {
		return f.createRefreshTokenFn(ctx, arg)
	}
	return db.RefreshToken{}, nil
}

func (f *fakeQuerier) GetRefreshToken(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	if f.getRefreshTokenFn != nil {
		return f.getRefreshTokenFn(ctx, tokenHash)
	}
	return db.RefreshToken{}, nil
}

func (f *fakeQuerier) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if f.revokeRefreshTokenFn != nil {
		return f.revokeRefreshTokenFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	if f.revokeAllUserRefreshTokensFn != nil {
		return f.revokeAllUserRefreshTokensFn(ctx, userID)
	}
	return nil
}

func (f *fakeQuerier) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	if f.deleteRefreshTokenFn != nil {
		return f.deleteRefreshTokenFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) CreateOTP(ctx context.Context, arg db.CreateOTPParams) (db.OtpCode, error) {
	if f.createOTPFn != nil {
		return f.createOTPFn(ctx, arg)
	}
	return db.OtpCode{}, nil
}

func (f *fakeQuerier) GetValidOTP(ctx context.Context, arg db.GetValidOTPParams) (db.OtpCode, error) {
	if f.getValidOTPFn != nil {
		return f.getValidOTPFn(ctx, arg)
	}
	return db.OtpCode{}, nil
}

func (f *fakeQuerier) GetValidOTPForUpdate(ctx context.Context, arg db.GetValidOTPForUpdateParams) (db.OtpCode, error) {
	if f.getValidOTPForUpdateFn != nil {
		return f.getValidOTPForUpdateFn(ctx, arg)
	}
	return db.OtpCode{}, nil
}

func (f *fakeQuerier) IncrementOTPAttempts(ctx context.Context, id uuid.UUID) (int32, error) {
	if f.incrementOTPAttemptsFn != nil {
		return f.incrementOTPAttemptsFn(ctx, id)
	}
	return 0, nil
}

func (f *fakeQuerier) MarkOTPUsed(ctx context.Context, id uuid.UUID) error {
	if f.markOTPUsedFn != nil {
		return f.markOTPUsedFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) DeleteOTP(ctx context.Context, id uuid.UUID) error {
	if f.deleteOTPFn != nil {
		return f.deleteOTPFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) CreateInvitation(ctx context.Context, arg db.CreateInvitationParams) (db.Invitation, error) {
	if f.createInvitationFn != nil {
		return f.createInvitationFn(ctx, arg)
	}
	return db.Invitation{}, nil
}

func (f *fakeQuerier) GetInvitationByID(ctx context.Context, id uuid.UUID) (db.Invitation, error) {
	if f.getInvitationByIDFn != nil {
		return f.getInvitationByIDFn(ctx, id)
	}
	return db.Invitation{}, nil
}

func (f *fakeQuerier) GetInvitationByToken(ctx context.Context, tokenHash string) (db.Invitation, error) {
	if f.getInvitationByTokenFn != nil {
		return f.getInvitationByTokenFn(ctx, tokenHash)
	}
	return db.Invitation{}, nil
}

func (f *fakeQuerier) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	if f.markInvitationAcceptedFn != nil {
		return f.markInvitationAcceptedFn(ctx, id)
	}
	return nil
}

func (f *fakeQuerier) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	if f.deleteInvitationFn != nil {
		return f.deleteInvitationFn(ctx, id)
	}
	return nil
}

// ===================================
// UNIT TESTS: WRAP & HELPERS
// ===================================

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected error
	}{
		{
			name:     "nil error returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "pgx.ErrNoRows wraps to ErrNotFound",
			input:    pgx.ErrNoRows,
			expected: ErrNotFound,
		},
		{
			name:     "SQLSTATE 42501 wraps to ErrPermissionDenied",
			input:    &pgconn.PgError{Code: "42501"},
			expected: ErrPermissionDenied,
		},
		{
			name:     "SQLSTATE 23505 wraps to ErrAlreadyExists",
			input:    &pgconn.PgError{Code: "23505"},
			expected: ErrAlreadyExists,
		},
		{
			name:     "SQLSTATE 23503 wraps to ErrForeignKey",
			input:    &pgconn.PgError{Code: "23503"},
			expected: ErrForeignKey,
		},
		{
			name:     "arbitrary error passes through",
			input:    errors.New("some network timeout"),
			expected: errors.New("some network timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrap(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got: %v", got)
				}
				return
			}

			if errors.Is(tt.expected, ErrNotFound) ||
				errors.Is(tt.expected, ErrPermissionDenied) ||
				errors.Is(tt.expected, ErrAlreadyExists) ||
				errors.Is(tt.expected, ErrForeignKey) {
				if !errors.Is(got, tt.expected) {
					t.Fatalf("expected %v, got %v", tt.expected, got)
				}
			} else if got.Error() != tt.expected.Error() {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestToText(t *testing.T) {
	t.Run("nil string pointer returns invalid text", func(t *testing.T) {
		got := toText(nil)
		if got.Valid {
			t.Fatalf("expected Valid to be false, got true")
		}
	})

	t.Run("valid string pointer returns valid text", func(t *testing.T) {
		str := "hello"
		got := toText(&str)
		if !got.Valid || got.String != "hello" {
			t.Fatalf("expected 'hello' with Valid=true, got: %+v", got)
		}
	})
}

// ===================================
// UNIT TESTS: USER (HAPPY & SAD)
// ===================================

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	testUserID := uuid.New()
	testEmail := "user@example.com"
	testPassword := "hashed_pass"

	t.Run("CreateUser - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createUserFn: func(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
				return db.User{ID: testUserID, Email: arg.Email}, nil
			},
		}
		repo := NewAuthRepository(fake)
		user, err := repo.CreateUser(ctx, testEmail, &testPassword)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != testUserID || user.Email != testEmail {
			t.Errorf("expected user ID %v and email %s, got %+v", testUserID, testEmail, user)
		}
	})

	t.Run("CreateUser - Sad Path (Duplicate Email 23505)", func(t *testing.T) {
		fake := &fakeQuerier{
			createUserFn: func(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
				return db.User{}, &pgconn.PgError{Code: "23505"}
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.CreateUser(ctx, testEmail, &testPassword)
		if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("expected ErrAlreadyExists, got: %v", err)
		}
	})

	t.Run("GetUserByID - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getUserByIDFn: func(ctx context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Email: testEmail}, nil
			},
		}
		repo := NewAuthRepository(fake)
		user, err := repo.GetUserByID(ctx, testUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != testUserID {
			t.Errorf("expected user ID %v, got %v", testUserID, user.ID)
		}
	})

	t.Run("GetUserByID - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getUserByIDFn: func(ctx context.Context, id uuid.UUID) (db.User, error) {
				return db.User{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetUserByID(ctx, testUserID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("GetUserByEmail - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getUserByEmailFn: func(ctx context.Context, email string) (db.User, error) {
				return db.User{ID: testUserID, Email: email}, nil
			},
		}
		repo := NewAuthRepository(fake)
		user, err := repo.GetUserByEmail(ctx, testEmail)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != testEmail {
			t.Errorf("expected email %s, got %s", testEmail, user.Email)
		}
	})

	t.Run("GetUserByEmail - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getUserByEmailFn: func(ctx context.Context, email string) (db.User, error) {
				return db.User{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetUserByEmail(ctx, testEmail)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("Status Operations - Happy Path", func(t *testing.T) {
		var suspended, unsuspended, deactivated, deleted bool
		fake := &fakeQuerier{
			suspendUserFn: func(ctx context.Context, id uuid.UUID) error {
				suspended = true
				return nil
			},
			unsuspendUserFn: func(ctx context.Context, id uuid.UUID) error {
				unsuspended = true
				return nil
			},
			deactivateUserFn: func(ctx context.Context, id uuid.UUID) error {
				deactivated = true
				return nil
			},
			deleteUserFn: func(ctx context.Context, id uuid.UUID) error {
				deleted = true
				return nil
			},
		}
		repo := NewAuthRepository(fake)
		_ = repo.SuspendUser(ctx, testUserID)
		_ = repo.UnsuspendUser(ctx, testUserID)
		_ = repo.DeactivateUser(ctx, testUserID)
		_ = repo.DeleteUser(ctx, testUserID)

		if !suspended || !unsuspended || !deactivated || !deleted {
			t.Fatalf("expected all user status actions to be executed successfully")
		}
	})

	t.Run("Status Operations - Sad Path (Permission Denied 42501)", func(t *testing.T) {
		fake := &fakeQuerier{
			deleteUserFn: func(ctx context.Context, id uuid.UUID) error {
				return &pgconn.PgError{Code: "42501"}
			},
		}
		repo := NewAuthRepository(fake)
		err := repo.DeleteUser(ctx, testUserID)
		if !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("expected ErrPermissionDenied, got: %v", err)
		}
	})
}

// ===================================
// UNIT TESTS: PROFILE (HAPPY & SAD)
// ===================================

func TestProfileRepository(t *testing.T) {
	ctx := context.Background()
	testUserID := uuid.New()
	testUsername := "johndoe"

	t.Run("CreateProfile - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createProfileFn: func(ctx context.Context, arg db.CreateProfileParams) (db.Profile, error) {
				return db.Profile{UserID: arg.UserID}, nil
			},
		}
		repo := NewAuthRepository(fake)
		profile, err := repo.CreateProfile(ctx, testUserID, nil, nil, &testUsername, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.UserID != testUserID {
			t.Errorf("expected user ID %v, got %v", testUserID, profile.UserID)
		}
	})

	t.Run("CreateProfile - Sad Path (Username Conflict 23505)", func(t *testing.T) {
		fake := &fakeQuerier{
			createProfileFn: func(ctx context.Context, arg db.CreateProfileParams) (db.Profile, error) {
				return db.Profile{}, &pgconn.PgError{Code: "23505"}
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.CreateProfile(ctx, testUserID, nil, nil, &testUsername, nil, nil, nil)
		if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("expected ErrAlreadyExists, got: %v", err)
		}
	})

	t.Run("GetProfileByUserID - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getProfileByUserIDFn: func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
				return db.Profile{UserID: userID}, nil
			},
		}
		repo := NewAuthRepository(fake)
		profile, err := repo.GetProfileByUserID(ctx, testUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile.UserID != testUserID {
			t.Errorf("expected user ID %v, got %v", testUserID, profile.UserID)
		}
	})

	t.Run("GetProfileByUserID - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getProfileByUserIDFn: func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
				return db.Profile{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetProfileByUserID(ctx, testUserID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("DeleteProfile - Happy Path & Sad Path", func(t *testing.T) {
		fakeSuccess := &fakeQuerier{
			deleteProfileFn: func(ctx context.Context, userID uuid.UUID) error {
				return nil
			},
		}
		repo := NewAuthRepository(fakeSuccess)
		if err := repo.DeleteProfile(ctx, testUserID); err != nil {
			t.Fatalf("expected success, got %v", err)
		}

		fakeFail := &fakeQuerier{
			deleteProfileFn: func(ctx context.Context, userID uuid.UUID) error {
				return errors.New("db disconnect")
			},
		}
		repoFail := NewAuthRepository(fakeFail)
		if err := repoFail.DeleteProfile(ctx, testUserID); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// ===================================
// UNIT TESTS: OAUTH (HAPPY & SAD)
// ===================================

func TestOAuthRepository(t *testing.T) {
	ctx := context.Background()
	testUserID := uuid.New()
	testProvider := "google"
	testProviderUserID := "google-123"

	t.Run("CreateOAuthConnection - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createOAuthConnectionFn: func(ctx context.Context, arg db.CreateOAuthConnectionParams) (db.OauthConnection, error) {
				return db.OauthConnection{UserID: arg.UserID, Provider: arg.Provider, ProviderUserID: arg.ProviderUserID}, nil
			},
		}
		repo := NewAuthRepository(fake)
		conn, err := repo.CreateOAuthConnection(ctx, testUserID, testProvider, testProviderUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if conn.Provider != testProvider {
			t.Errorf("expected provider %s, got %s", testProvider, conn.Provider)
		}
	})

	t.Run("CreateOAuthConnection - Sad Path (Foreign Key Violation 23503)", func(t *testing.T) {
		fake := &fakeQuerier{
			createOAuthConnectionFn: func(ctx context.Context, arg db.CreateOAuthConnectionParams) (db.OauthConnection, error) {
				return db.OauthConnection{}, &pgconn.PgError{Code: "23503"}
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.CreateOAuthConnection(ctx, testUserID, testProvider, testProviderUserID)
		if !errors.Is(err, ErrForeignKey) {
			t.Fatalf("expected ErrForeignKey, got: %v", err)
		}
	})

	t.Run("GetOAuthConnection - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getOAuthConnectionFn: func(ctx context.Context, arg db.GetOAuthConnectionParams) (db.OauthConnection, error) {
				return db.OauthConnection{Provider: arg.Provider, ProviderUserID: arg.ProviderUserID}, nil
			},
		}
		repo := NewAuthRepository(fake)
		conn, err := repo.GetOAuthConnection(ctx, testProvider, testProviderUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if conn.ProviderUserID != testProviderUserID {
			t.Errorf("expected provider user ID %s, got %s", testProviderUserID, conn.ProviderUserID)
		}
	})

	t.Run("GetOAuthConnection - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getOAuthConnectionFn: func(ctx context.Context, arg db.GetOAuthConnectionParams) (db.OauthConnection, error) {
				return db.OauthConnection{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetOAuthConnection(ctx, testProvider, testProviderUserID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})
}

// ===================================
// UNIT TESTS: REFRESH TOKENS (HAPPY & SAD)
// ===================================

func TestRefreshTokenRepository(t *testing.T) {
	ctx := context.Background()
	testUserID := uuid.New()
	testTokenID := uuid.New()
	testTokenHash := "hash_abc_123"

	t.Run("CreateRefreshToken - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createRefreshTokenFn: func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
				return db.RefreshToken{ID: testTokenID, UserID: arg.UserID, TokenHash: arg.TokenHash}, nil
			},
		}
		repo := NewAuthRepository(fake)
		token, err := repo.CreateRefreshToken(ctx, testUserID, testTokenHash, time.Now().Add(24*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.TokenHash != testTokenHash {
			t.Errorf("expected hash %s, got %s", testTokenHash, token.TokenHash)
		}
	})

	t.Run("CreateRefreshToken - Sad Path (Foreign Key Violation 23503)", func(t *testing.T) {
		fake := &fakeQuerier{
			createRefreshTokenFn: func(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
				return db.RefreshToken{}, &pgconn.PgError{Code: "23503"}
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.CreateRefreshToken(ctx, testUserID, testTokenHash, time.Now().Add(24*time.Hour))
		if !errors.Is(err, ErrForeignKey) {
			t.Fatalf("expected ErrForeignKey, got: %v", err)
		}
	})

	t.Run("GetRefreshToken - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getRefreshTokenFn: func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
				return db.RefreshToken{ID: testTokenID, TokenHash: tokenHash}, nil
			},
		}
		repo := NewAuthRepository(fake)
		token, err := repo.GetRefreshToken(ctx, testTokenHash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.ID != testTokenID {
			t.Errorf("expected token ID %v, got %v", testTokenID, token.ID)
		}
	})

	t.Run("GetRefreshToken - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getRefreshTokenFn: func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
				return db.RefreshToken{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetRefreshToken(ctx, testTokenHash)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("Revoke & Delete - Happy & Sad", func(t *testing.T) {
		fake := &fakeQuerier{
			revokeRefreshTokenFn: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
			deleteRefreshTokenFn: func(ctx context.Context, id uuid.UUID) error {
				return errors.New("timeout")
			},
		}
		repo := NewAuthRepository(fake)
		if err := repo.RevokeRefreshToken(ctx, testTokenID); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if err := repo.DeleteRefreshToken(ctx, testTokenID); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// ===================================
// UNIT TESTS: OTP (HAPPY & SAD)
// ===================================

func TestOTPRepository(t *testing.T) {
	ctx := context.Background()
	testUserID := uuid.New()
	testOTPID := uuid.New()
	testPurpose := "signup"

	t.Run("CreateOTP - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createOTPFn: func(ctx context.Context, arg db.CreateOTPParams) (db.OtpCode, error) {
				return db.OtpCode{ID: testOTPID, UserID: arg.UserID, Purpose: arg.Purpose}, nil
			},
		}
		repo := NewAuthRepository(fake)
		otp, err := repo.CreateOTP(ctx, testUserID, "codehash", testPurpose, time.Now().Add(10*time.Minute))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if otp.Purpose != testPurpose {
			t.Errorf("expected purpose %s, got %s", testPurpose, otp.Purpose)
		}
	})

	t.Run("GetValidOTP - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getValidOTPFn: func(ctx context.Context, arg db.GetValidOTPParams) (db.OtpCode, error) {
				return db.OtpCode{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetValidOTP(ctx, testUserID, testPurpose)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("GetValidOTPForUpdate - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getValidOTPForUpdateFn: func(ctx context.Context, arg db.GetValidOTPForUpdateParams) (db.OtpCode, error) {
				return db.OtpCode{ID: testOTPID, UserID: arg.UserID, Purpose: arg.Purpose}, nil
			},
		}
		repo := NewAuthRepository(fake)
		otp, err := repo.GetValidOTPForUpdate(ctx, testUserID, testPurpose)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if otp.ID != testOTPID {
			t.Errorf("expected OTP ID %v, got %v", testOTPID, otp.ID)
		}
	})

	t.Run("GetValidOTPForUpdate - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getValidOTPForUpdateFn: func(ctx context.Context, arg db.GetValidOTPForUpdateParams) (db.OtpCode, error) {
				return db.OtpCode{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetValidOTPForUpdate(ctx, testUserID, testPurpose)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("IncrementOTPAttempts - Happy & Sad", func(t *testing.T) {
		fake := &fakeQuerier{
			incrementOTPAttemptsFn: func(ctx context.Context, id uuid.UUID) (int32, error) {
				return 3, nil
			},
		}
		repo := NewAuthRepository(fake)
		attempts, err := repo.IncrementOTPAttempts(ctx, testOTPID)
		if err != nil || attempts != 3 {
			t.Fatalf("expected 3 attempts, got %d, err: %v", attempts, err)
		}

		fakeFail := &fakeQuerier{
			incrementOTPAttemptsFn: func(ctx context.Context, id uuid.UUID) (int32, error) {
				return 0, errors.New("db error")
			},
		}
		repoFail := NewAuthRepository(fakeFail)
		_, err = repoFail.IncrementOTPAttempts(ctx, testOTPID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

// ===================================
// UNIT TESTS: INVITATIONS (HAPPY & SAD)
// ===================================

func TestInvitationRepository(t *testing.T) {
	ctx := context.Background()
	testInviteID := uuid.New()
	testInvitedBy := uuid.New()
	testEmail := "invited@example.com"
	testTokenHash := "hash123"

	t.Run("CreateInvitation - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			createInvitationFn: func(ctx context.Context, arg db.CreateInvitationParams) (db.Invitation, error) {
				return db.Invitation{ID: testInviteID, Email: arg.Email}, nil
			},
		}
		repo := NewAuthRepository(fake)
		inv, err := repo.CreateInvitation(ctx, testEmail, testTokenHash, testInvitedBy, time.Now().Add(7*24*time.Hour))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inv.Email != testEmail {
			t.Errorf("expected email %s, got %s", testEmail, inv.Email)
		}
	})

	t.Run("CreateInvitation - Sad Path (Foreign Key Violation 23503)", func(t *testing.T) {
		fake := &fakeQuerier{
			createInvitationFn: func(ctx context.Context, arg db.CreateInvitationParams) (db.Invitation, error) {
				return db.Invitation{}, &pgconn.PgError{Code: "23503"}
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.CreateInvitation(ctx, testEmail, testTokenHash, testInvitedBy, time.Now().Add(7*24*time.Hour))
		if !errors.Is(err, ErrForeignKey) {
			t.Fatalf("expected ErrForeignKey, got: %v", err)
		}
	})

	t.Run("GetInvitationByID - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getInvitationByIDFn: func(ctx context.Context, id uuid.UUID) (db.Invitation, error) {
				return db.Invitation{ID: id, Email: testEmail}, nil
			},
		}
		repo := NewAuthRepository(fake)
		inv, err := repo.GetInvitationByID(ctx, testInviteID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inv.ID != testInviteID {
			t.Errorf("expected ID %v, got %v", testInviteID, inv.ID)
		}
	})

	t.Run("GetInvitationByID - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getInvitationByIDFn: func(ctx context.Context, id uuid.UUID) (db.Invitation, error) {
				return db.Invitation{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetInvitationByID(ctx, testInviteID)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("GetInvitationByToken - Happy Path", func(t *testing.T) {
		fake := &fakeQuerier{
			getInvitationByTokenFn: func(ctx context.Context, tokenHash string) (db.Invitation, error) {
				return db.Invitation{ID: testInviteID, TokenHash: tokenHash}, nil
			},
		}
		repo := NewAuthRepository(fake)
		inv, err := repo.GetInvitationByToken(ctx, testTokenHash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inv.TokenHash != testTokenHash {
			t.Errorf("expected token hash %s, got %s", testTokenHash, inv.TokenHash)
		}
	})

	t.Run("GetInvitationByToken - Sad Path (Not Found)", func(t *testing.T) {
		fake := &fakeQuerier{
			getInvitationByTokenFn: func(ctx context.Context, tokenHash string) (db.Invitation, error) {
				return db.Invitation{}, pgx.ErrNoRows
			},
		}
		repo := NewAuthRepository(fake)
		_, err := repo.GetInvitationByToken(ctx, testTokenHash)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("DeleteInvitation - Happy & Sad", func(t *testing.T) {
		fake := &fakeQuerier{
			deleteInvitationFn: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		repo := NewAuthRepository(fake)
		if err := repo.DeleteInvitation(ctx, testInviteID); err != nil {
			t.Fatalf("expected success, got %v", err)
		}

		fakeFail := &fakeQuerier{
			deleteInvitationFn: func(ctx context.Context, id uuid.UUID) error {
				return errors.New("db error")
			},
		}
		repoFail := NewAuthRepository(fakeFail)
		if err := repoFail.DeleteInvitation(ctx, testInviteID); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}
