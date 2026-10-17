package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/repository"
	"server/internal/auth/service/oauth"
	db "server/internal/db/generated"
)

func TestSessionService_Login(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	validPassword := "correctPassword123"
	hashedPasswordBytes, _ := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.MinCost)
	hashedPasswordStr := string(hashedPasswordBytes)

	t.Run("successful login with verified account returns token pair", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		verifiedAt := time.Now()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:              testUserID,
				Email:           email,
				PasswordHash:    pgtype.Text{String: hashedPasswordStr, Valid: true},
				EmailVerifiedAt: &verifiedAt,
			}, nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.Login(ctx, "jane@example.com", validPassword)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" || tokenPair.RefreshToken == "" {
			t.Fatalf("expected valid token pair, got %v", tokenPair)
		}
	})

	t.Run("fails when password login is disabled", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.EnablePasswordLogin = false

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "jane@example.com", validPassword)
		if !errors.Is(err, autherr.ErrPasswordLoginDisabled) {
			t.Fatalf("expected ErrPasswordLoginDisabled, got %v", err)
		}
	})

	t.Run("missing user performs dummy password compare and returns ErrInvalidCredentials", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "nonexistent@example.com", validPassword)
		if !errors.Is(err, autherr.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("OAuth user without password returns ErrOAuthAccount", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pgtype.Text{Valid: false},
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "oauth@example.com", validPassword)
		if !errors.Is(err, autherr.ErrOAuthAccount) {
			t.Fatalf("expected ErrOAuthAccount, got %v", err)
		}
	})

	t.Run("incorrect password returns ErrInvalidCredentials", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pgtype.Text{String: hashedPasswordStr, Valid: true},
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "jane@example.com", "wrongPassword999")
		if !errors.Is(err, autherr.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("suspended user returns ErrAccountSuspended", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pgtype.Text{String: hashedPasswordStr, Valid: true},
				IsSuspended:  true,
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "suspended@example.com", validPassword)
		if !errors.Is(err, autherr.ErrAccountSuspended) {
			t.Fatalf("expected ErrAccountSuspended, got %v", err)
		}
	})

	t.Run("deactivated user returns ErrAccountDeactivated", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:            uuid.New(),
				Email:         email,
				PasswordHash:  pgtype.Text{String: hashedPasswordStr, Valid: true},
				IsDeactivated: true,
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "deactivated@example.com", validPassword)
		if !errors.Is(err, autherr.ErrAccountDeactivated) {
			t.Fatalf("expected ErrAccountDeactivated, got %v", err)
		}
	})

	t.Run("unverified user returns ErrEmailNotVerified when verification is required", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Verification.Required = true

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:              uuid.New(),
				Email:           email,
				PasswordHash:    pgtype.Text{String: hashedPasswordStr, Valid: true},
				EmailVerifiedAt: nil,
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Login(ctx, "unverified@example.com", validPassword)
		if !errors.Is(err, autherr.ErrEmailNotVerified) {
			t.Fatalf("expected ErrEmailNotVerified, got %v", err)
		}
	})
}

func TestSessionService_Refresh(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	t.Run("successful refresh with token rotation revokes old token and issues new pair", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Session.RotateRefreshToken = true

		testUserID := uuid.New()
		testTokenID := uuid.New()
		rawToken := "valid-refresh-token-xyz"
		oldTokenRevoked := false

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{
				ID:        testTokenID,
				UserID:    testUserID,
				RevokedAt: nil,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		}
		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{ID: testUserID}, nil
		}
		mockRepo.RevokeRefreshTokenFn = func(ctx context.Context, id uuid.UUID) error {
			if id == testTokenID {
				oldTokenRevoked = true
			}
			return nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.Refresh(ctx, rawToken)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" || tokenPair.RefreshToken == "" {
			t.Fatalf("expected new token pair, got %v", tokenPair)
		}
		if tokenPair.RefreshToken == rawToken {
			t.Fatal("expected rotated refresh token, got identical token")
		}
		if !oldTokenRevoked {
			t.Fatal("expected old refresh token to be revoked")
		}
	})

	t.Run("successful refresh without rotation preserves existing refresh token", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Session.RotateRefreshToken = false

		testUserID := uuid.New()
		rawToken := "valid-refresh-token-no-rotate"

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{
				ID:        uuid.New(),
				UserID:    testUserID,
				RevokedAt: nil,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		}
		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{ID: testUserID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.Refresh(ctx, rawToken)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair.RefreshToken != rawToken {
			t.Fatalf("expected preserved refresh token, got %s", tokenPair.RefreshToken)
		}
	})

	t.Run("theft detection: replay of revoked token revokes all user sessions", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		testUserID := uuid.New()
		revokedTime := time.Now().Add(-1 * time.Hour)
		allSessionsRevoked := false

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{
				ID:        uuid.New(),
				UserID:    testUserID,
				RevokedAt: &revokedTime,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}, nil
		}
		mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
			if userID == testUserID {
				allSessionsRevoked = true
			}
			return nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Refresh(ctx, "stolen-replayed-token")

		if !errors.Is(err, autherr.ErrRefreshTokenReused) {
			t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
		}
		if !allSessionsRevoked {
			t.Fatal("expected RevokeAllUserRefreshTokens to be called on replay attack")
		}
	})

	t.Run("expired refresh token returns ErrInvalidRefreshToken", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				RevokedAt: nil,
				ExpiresAt: time.Now().Add(-1 * time.Minute),
			}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.Refresh(ctx, "expired-token")
		if !errors.Is(err, autherr.ErrRefreshTokenInvalid) {
			t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
		}
	})
}

func TestSessionService_Logout(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	t.Run("successful logout revokes token", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testTokenID := uuid.New()
		tokenRevoked := false

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{
				ID:        testTokenID,
				RevokedAt: nil,
			}, nil
		}
		mockRepo.RevokeRefreshTokenFn = func(ctx context.Context, id uuid.UUID) error {
			if id == testTokenID {
				tokenRevoked = true
			}
			return nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		err := sessionService.Logout(ctx, "active-refresh-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !tokenRevoked {
			t.Fatal("expected token to be revoked")
		}
	})

	t.Run("idempotent when token is not found or already revoked", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetRefreshTokenFn = func(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
			return db.RefreshToken{}, repository.ErrNotFound
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		err := sessionService.Logout(ctx, "missing-token")
		if err != nil {
			t.Fatalf("expected nil for idempotent logout, got %v", err)
		}
	})
}

func TestSessionService_OAuthLogin(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	mockProvider := &mockOAuthProvider{
		VerifyTokenFn: func(ctx context.Context, idToken string) (*oauth.Identity, error) {
			if idToken == "valid-google-token" {
				return &oauth.Identity{
					ProviderID: "google-sub-12345",
					Email:      "oauthuser@example.com",
					FirstName:  "OAuth",
					LastName:   "User",
				}, nil
			}
			return nil, errors.New("bad id token")
		},
	}

	t.Run("success with existing connection logs in linked user", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.OAuth.Enabled = true
		cfg.OAuth.Providers = map[string]oauth.Provider{
			"google": mockProvider,
		}

		testUserID := uuid.New()
		mockRepo.GetOAuthConnectionFn = func(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error) {
			return db.OauthConnection{
				UserID:         testUserID,
				Provider:       provider,
				ProviderUserID: providerUserID,
			}, nil
		}
		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{ID: testUserID, Email: "oauthuser@example.com"}, nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.OAuthLogin(ctx, "google", "valid-google-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" {
			t.Fatalf("expected token pair, got %v", tokenPair)
		}
	})

	t.Run("success with unlinked user links account and logs in", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.OAuth.Enabled = true
		cfg.OAuth.Providers = map[string]oauth.Provider{
			"google": mockProvider,
		}

		testUserID := uuid.New()
		mockRepo.GetOAuthConnectionFn = func(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error) {
			return db.OauthConnection{}, repository.ErrNotFound
		}
		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		connectionCreated := false
		mockRepo.CreateOAuthConnectionFn = func(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error) {
			connectionCreated = true
			return db.OauthConnection{UserID: userID, Provider: provider}, nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.OAuthLogin(ctx, "google", "valid-google-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil {
			t.Fatal("expected token pair")
		}
		if !connectionCreated {
			t.Fatal("expected OAuth connection to be linked")
		}
	})

	t.Run("success with JIT provisioning for new user", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.OAuth.Enabled = true
		cfg.Registration.EnableOAuthRegistration = true
		cfg.Registration.Mode = config.RegistrationModeOpen
		cfg.OAuth.Providers = map[string]oauth.Provider{
			"google": mockProvider,
		}

		testUserID := uuid.New()
		userCreated := false
		profileCreated := false
		connectionCreated := false

		mockRepo.GetOAuthConnectionFn = func(ctx context.Context, provider, providerUserID string) (db.OauthConnection, error) {
			return db.OauthConnection{}, repository.ErrNotFound
		}
		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}
		mockRepo.CreateUserFn = func(ctx context.Context, email string, passwordHash *string) (db.User, error) {
			userCreated = true
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			return nil
		}
		mockRepo.CreateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			profileCreated = true
			return db.Profile{UserID: userID}, nil
		}
		mockRepo.CreateOAuthConnectionFn = func(ctx context.Context, userID uuid.UUID, provider, providerUserID string) (db.OauthConnection, error) {
			connectionCreated = true
			return db.OauthConnection{UserID: userID, Provider: provider}, nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		tokenPair, err := sessionService.OAuthLogin(ctx, "google", "valid-google-token")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil {
			t.Fatal("expected token pair")
		}
		if !userCreated || !profileCreated || !connectionCreated {
			t.Fatalf("expected userCreated=%v, profileCreated=%v, connectionCreated=%v",
				userCreated, profileCreated, connectionCreated)
		}
	})

	t.Run("rejected when provider is not supported or disabled", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.OAuth.Enabled = false

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.OAuthLogin(ctx, "unsupported_provider", "token")
		if !errors.Is(err, autherr.ErrOAuthProviderNotSupported) {
			t.Fatalf("expected ErrOAuthProviderNotSupported, got %v", err)
		}
	})

	t.Run("rejected when id token is invalid", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.OAuth.Enabled = true
		cfg.OAuth.Providers = map[string]oauth.Provider{
			"google": mockProvider,
		}

		sessionService := NewSessionService(mockRepo, cfg, jwtIssuer)
		_, err := sessionService.OAuthLogin(ctx, "google", "invalid-token-signature")
		if !errors.Is(err, autherr.ErrOAuthTokenInvalid) {
			t.Fatalf("expected ErrOAuthTokenInvalid, got %v", err)
		}
	})
}
