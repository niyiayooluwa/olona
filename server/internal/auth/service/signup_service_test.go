package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

func TestSignupService_Signup(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	t.Run("success when email verification is required", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost
		cfg.Verification.Required = true

		testUserID := uuid.New()
		userCreated := false
		profileCreated := false
		otpCreated := false
		mailSent := false

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return false, nil
		}
		mockRepo.CreateUserFn = func(ctx context.Context, email string, passwordHash *string) (db.User, error) {
			userCreated = true
			return db.User{
				ID:    testUserID,
				Email: email,
			}, nil
		}
		mockRepo.CreateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			profileCreated = true
			return db.Profile{UserID: userID}, nil
		}
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			otpCreated = true
			if purpose != "signup_verify" {
				t.Fatalf("expected purpose 'signup_verify', got %s", purpose)
			}
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}
		mockMailerInstance.SendOTPFn = func(ctx context.Context, email, code string) error {
			mailSent = true
			return nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, mockMailerInstance)
		username := "janedoe"
		tokenPair, err := signupService.Signup(ctx, "jane@example.com", "securePassword123", model.ProfileInput{
			Username: &username,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair != nil {
			t.Fatalf("expected tokenPair to be nil when verification is required, got %v", tokenPair)
		}
		if !userCreated || !profileCreated || !otpCreated || !mailSent {
			t.Fatalf("expected all lifecycle steps to execute, got user=%v profile=%v otp=%v mail=%v",
				userCreated, profileCreated, otpCreated, mailSent)
		}
	})

	t.Run("success when email verification is optional", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost
		cfg.Verification.Required = false

		testUserID := uuid.New()
		verifiedCalled := false

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return false, nil
		}
		mockRepo.CreateUserFn = func(ctx context.Context, email string, passwordHash *string) (db.User, error) {
			return db.User{
				ID:    testUserID,
				Email: email,
			}, nil
		}
		mockRepo.CreateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			return db.Profile{UserID: userID}, nil
		}
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			verifiedCalled = true
			return nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, mockMailerInstance)
		tokenPair, err := signupService.Signup(ctx, "jane@example.com", "securePassword123", model.ProfileInput{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" || tokenPair.RefreshToken == "" {
			t.Fatalf("expected active token pair, got %v", tokenPair)
		}
		if !verifiedCalled {
			t.Fatal("expected MarkEmailVerified to be called when verification is optional")
		}
	})

	t.Run("rejected when registration mode is closed or invite only", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Registration.Mode = config.RegistrationModeInviteOnly

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.Signup(ctx, "jane@example.com", "securePassword123", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrRegistrationDisabled) {
			t.Fatalf("expected ErrRegistrationDisabled, got %v", err)
		}
	})

	t.Run("rejected when password is too short", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.MinLength = 10

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.Signup(ctx, "jane@example.com", "short", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrPasswordTooShort) {
			t.Fatalf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("rejected when username is already taken", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return true, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		username := "claimed_username"
		_, err := signupService.Signup(ctx, "jane@example.com", "securePassword123", model.ProfileInput{
			Username: &username,
		})
		if !errors.Is(err, autherr.ErrUsernameTaken) {
			t.Fatalf("expected ErrUsernameTaken, got %v", err)
		}
	})

	t.Run("rejected when email is already registered", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost

		mockRepo.CreateUserFn = func(ctx context.Context, email string, passwordHash *string) (db.User, error) {
			return db.User{}, repository.ErrAlreadyExists
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.Signup(ctx, "jane@example.com", "securePassword123", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrEmailAlreadyRegistered) {
			t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
		}
	})
}

func TestSignupService_VerifyEmail(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	t.Run("successful verification consumes OTP and returns token pair", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		testOTPID := uuid.New()
		rawCode := "123456"
		codeHash := utils.HashToken(rawCode)

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:        testOTPID,
				UserID:    testUserID,
				CodeHash:  codeHash,
				Purpose:   "signup_verify",
				Attempts:  0,
				ExpiresAt: time.Now().Add(10 * time.Minute),
			}, nil
		}

		otpMarkedUsed := false
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
			otpMarkedUsed = true
			return nil
		}

		emailMarkedVerified := false
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			emailMarkedVerified = true
			return nil
		}

		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		tokenPair, err := signupService.VerifyEmail(ctx, "jane@example.com", rawCode)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" {
			t.Fatalf("expected valid token pair, got %v", tokenPair)
		}
		if !otpMarkedUsed || !emailMarkedVerified {
			t.Fatalf("expected OTP marked used (%v) and email marked verified (%v)", otpMarkedUsed, emailMarkedVerified)
		}
	})

	t.Run("user not found returns ErrOTPNotFound", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.VerifyEmail(ctx, "missing@example.com", "123456")
		if !errors.Is(err, autherr.ErrOTPNotFound) {
			t.Fatalf("expected ErrOTPNotFound, got %v", err)
		}
	})

	t.Run("OTP record not found returns ErrOTPNotFound", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{}, repository.ErrNotFound
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.VerifyEmail(ctx, "jane@example.com", "123456")
		if !errors.Is(err, autherr.ErrOTPNotFound) {
			t.Fatalf("expected ErrOTPNotFound, got %v", err)
		}
	})

	t.Run("max attempts exceeded returns ErrOTPMaxAttempts", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Verification.MaxOTPAttempts = 5
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:       uuid.New(),
				UserID:   testUserID,
				CodeHash: "some_hash",
				Attempts: 5,
			}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.VerifyEmail(ctx, "jane@example.com", "123456")
		if !errors.Is(err, autherr.ErrOTPMaxAttempts) {
			t.Fatalf("expected ErrOTPMaxAttempts, got %v", err)
		}
	})

	t.Run("incorrect code increments attempts and returns ErrOTPIncorrect", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Verification.MaxOTPAttempts = 5
		testUserID := uuid.New()
		testOTPID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:       testOTPID,
				UserID:   testUserID,
				CodeHash: utils.HashToken("654321"),
				Attempts: 1,
			}, nil
		}

		incrementCalled := false
		mockRepo.IncrementOTPAttemptsFn = func(ctx context.Context, id uuid.UUID) (int32, error) {
			incrementCalled = true
			return 2, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.VerifyEmail(ctx, "jane@example.com", "000000")
		if !errors.Is(err, autherr.ErrOTPIncorrect) {
			t.Fatalf("expected ErrOTPIncorrect, got %v", err)
		}
		if !incrementCalled {
			t.Fatal("expected IncrementOTPAttempts to be called")
		}
	})
}

func TestSignupService_ResendOTP(t *testing.T) {
	ctx := context.Background()

	t.Run("success generates new OTP and sends email", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}
		cfg := setupTestConfig()
		cfg.Verification.ResendCooldown = 60 * time.Second

		testUserID := uuid.New()
		testOTPID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email, EmailVerifiedAt: nil}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			// Created 2 minutes ago (outside cooldown)
			return db.OtpCode{
				ID:        testOTPID,
				UserID:    testUserID,
				CreatedAt: time.Now().Add(-2 * time.Minute),
			}, nil
		}

		oldMarkedUsed := false
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
			oldMarkedUsed = true
			return nil
		}

		newOTPCreated := false
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			newOTPCreated = true
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}

		mailSent := false
		mockMailerInstance.SendOTPFn = func(ctx context.Context, email, code string) error {
			mailSent = true
			return nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, mockMailerInstance)
		err := signupService.ResendOTP(ctx, "jane@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !oldMarkedUsed || !newOTPCreated || !mailSent {
			t.Fatalf("expected oldMarkedUsed=%v, newOTPCreated=%v, mailSent=%v", oldMarkedUsed, newOTPCreated, mailSent)
		}
	})

	t.Run("anti-enumeration returns nil if user does not exist", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "ghost@example.com")
		if err != nil {
			t.Fatalf("expected nil for anti-enumeration, got %v", err)
		}
	})

	t.Run("returns nil if user is already verified", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		verifiedAt := time.Now()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: uuid.New(), Email: email, EmailVerifiedAt: &verifiedAt}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "verified@example.com")
		if err != nil {
			t.Fatalf("expected nil for verified account, got %v", err)
		}
	})

	t.Run("returns ErrOTPCooldown when existing code within cooldown window", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Verification.ResendCooldown = 60 * time.Second
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email, EmailVerifiedAt: nil}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			// Created 10 seconds ago (inside 60s cooldown)
			return db.OtpCode{
				ID:        uuid.New(),
				UserID:    testUserID,
				CreatedAt: time.Now().Add(-10 * time.Second),
			}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "jane@example.com")
		if !errors.Is(err, autherr.ErrOTPCooldown) {
			t.Fatalf("expected ErrOTPCooldown, got %v", err)
		}
	})

	// Regression: these cases used to return nil, reporting success for a
	// database outage and telling the caller an email had been sent.
	t.Run("propagates a database error when looking up the user", func(t *testing.T) {
		dbErr := errors.New("connection refused")
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, dbErr
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "jane@example.com")
		if !errors.Is(err, dbErr) {
			t.Fatalf("expected the underlying database error to propagate, got %v", err)
		}
	})

	t.Run("propagates a database error when loading the existing OTP", func(t *testing.T) {
		dbErr := errors.New("connection reset")
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email, EmailVerifiedAt: nil}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{}, dbErr
		}

		// A new code must NOT be issued when we could not read the current one.
		createdNew := false
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			createdNew = true
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "jane@example.com")
		if !errors.Is(err, dbErr) {
			t.Fatalf("expected the underlying database error to propagate, got %v", err)
		}
		if createdNew {
			t.Fatal("expected no new OTP to be created when the existing one could not be read")
		}
	})

	t.Run("treats a missing existing OTP as nothing to invalidate", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email, EmailVerifiedAt: nil}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{}, repository.ErrNotFound
		}
		createdNew := false
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			createdNew = true
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		if err := signupService.ResendOTP(ctx, "jane@example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !createdNew {
			t.Fatal("expected a fresh OTP to be issued when no active code exists")
		}
	})

	t.Run("aborts when the previous OTP cannot be invalidated", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		markErr := errors.New("deadlock detected")

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: testUserID, Email: email, EmailVerifiedAt: nil}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			// Outside the cooldown window, so we attempt to invalidate it.
			return db.OtpCode{ID: uuid.New(), UserID: userID, CreatedAt: time.Now().Add(-2 * time.Minute)}, nil
		}
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
			return markErr
		}
		createdNew := false
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			createdNew = true
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, nil, nil)
		err := signupService.ResendOTP(ctx, "jane@example.com")
		if !errors.Is(err, markErr) {
			t.Fatalf("expected the MarkOTPUsed error to propagate, got %v", err)
		}
		// Issuing a second live code while the first still verifies would let
		// either one be used, so this must not proceed.
		if createdNew {
			t.Fatal("expected no new OTP when the previous one could not be invalidated")
		}
	})
}

func TestSignupService_AcceptInvite(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	t.Run("successful invite redemption creates verified user and returns token pair", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost

		testInviteID := uuid.New()
		testUserID := uuid.New()
		rawToken := "secure-invite-token-abc"

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:         testInviteID,
				Email:      "invitee@example.com",
				ExpiresAt:  time.Now().Add(24 * time.Hour),
				AcceptedAt: nil,
			}, nil
		}
		mockRepo.CreateUserFn = func(ctx context.Context, email string, passwordHash *string) (db.User, error) {
			return db.User{ID: testUserID, Email: email}, nil
		}
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			return nil
		}
		mockRepo.CreateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			return db.Profile{UserID: userID}, nil
		}
		inviteAccepted := false
		mockRepo.MarkInvitationAcceptedFn = func(ctx context.Context, id uuid.UUID) error {
			inviteAccepted = true
			return nil
		}
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: userID}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		tokenPair, err := signupService.AcceptInvite(ctx, rawToken, "securePassword123", model.ProfileInput{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokenPair == nil || tokenPair.AccessToken == "" {
			t.Fatalf("expected token pair, got %v", tokenPair)
		}
		if !inviteAccepted {
			t.Fatal("expected invitation to be marked accepted")
		}
	})

	t.Run("invite not found returns ErrInvitationNotFound", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{}, repository.ErrNotFound
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.AcceptInvite(ctx, "invalid-token", "securePassword123", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrInvitationNotFound) {
			t.Fatalf("expected ErrInvitationNotFound, got %v", err)
		}
	})

	t.Run("already accepted invite returns ErrInvitationAlreadyAccepted", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		acceptedTime := time.Now().Add(-1 * time.Hour)

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:         uuid.New(),
				AcceptedAt: &acceptedTime,
				ExpiresAt:  time.Now().Add(24 * time.Hour),
			}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.AcceptInvite(ctx, "used-token", "securePassword123", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrInvitationAlreadyAccepted) {
			t.Fatalf("expected ErrInvitationAlreadyAccepted, got %v", err)
		}
	})

	t.Run("expired invite returns ErrInvitationExpired", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:         uuid.New(),
				AcceptedAt: nil,
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
			}, nil
		}

		signupService := NewSignupService(mockRepo, cfg, jwtIssuer, nil)
		_, err := signupService.AcceptInvite(ctx, "expired-token", "securePassword123", model.ProfileInput{})
		if !errors.Is(err, autherr.ErrInvitationExpired) {
			t.Fatalf("expected ErrInvitationExpired, got %v", err)
		}
	})
}
