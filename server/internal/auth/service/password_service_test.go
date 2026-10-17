package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	autherr "server/internal/auth/errors"
	"server/internal/auth/repository"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

func TestPasswordService_ForgotPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("success sends reset email and creates OTP", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}
		cfg := setupTestConfig()
		cfg.Verification.ResendCooldown = 60 * time.Second

		testUserID := uuid.New()
		otpCreated := false
		emailSent := false

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           testUserID,
				Email:        email,
				PasswordHash: pgtype.Text{String: "hash", Valid: true},
			}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{}, repository.ErrNotFound
		}
		mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
			otpCreated = true
			if purpose != "password_reset" {
				t.Fatalf("expected purpose 'password_reset', got %s", purpose)
			}
			return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
		}
		mockMailerInstance.SendPasswordResetFn = func(ctx context.Context, email, code string) error {
			emailSent = true
			return nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, mockMailerInstance)
		err := passwordService.ForgotPassword(ctx, "jane@example.com")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !otpCreated || !emailSent {
			t.Fatalf("expected otpCreated=%v, emailSent=%v", otpCreated, emailSent)
		}
	})

	t.Run("anti-enumeration returns nil if user does not exist", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ForgotPassword(ctx, "ghost@example.com")
		if err != nil {
			t.Fatalf("expected nil for anti-enumeration, got %v", err)
		}
	})

	t.Run("anti-enumeration returns nil if user is OAuth-only", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pgtype.Text{Valid: false},
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ForgotPassword(ctx, "oauth@example.com")
		if err != nil {
			t.Fatalf("expected nil for OAuth account, got %v", err)
		}
	})

	t.Run("anti-enumeration returns nil if user is suspended or deactivated", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: pgtype.Text{String: "hash", Valid: true},
				IsSuspended:  true,
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ForgotPassword(ctx, "suspended@example.com")
		if err != nil {
			t.Fatalf("expected nil for suspended account, got %v", err)
		}
	})

	t.Run("returns ErrOTPCooldown if requested within cooldown window", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Verification.ResendCooldown = 60 * time.Second
		testUserID := uuid.New()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:           testUserID,
				Email:        email,
				PasswordHash: pgtype.Text{String: "hash", Valid: true},
			}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:        uuid.New(),
				UserID:    testUserID,
				CreatedAt: time.Now().Add(-15 * time.Second),
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ForgotPassword(ctx, "jane@example.com")
		if !errors.Is(err, autherr.ErrOTPCooldown) {
			t.Fatalf("expected ErrOTPCooldown, got %v", err)
		}
	})
}

func TestPasswordService_ResetPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("successful reset updates password, verifies email, and revokes all refresh tokens", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost

		testUserID := uuid.New()
		testOTPID := uuid.New()
		rawCode := "888999"
		codeHash := utils.HashToken(rawCode)

		passwordUpdated := false
		otpMarkedUsed := false
		emailVerified := false
		tokensRevoked := false

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{
				ID:              testUserID,
				Email:           email,
				EmailVerifiedAt: nil,
			}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, userID uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:       testOTPID,
				UserID:   testUserID,
				CodeHash: codeHash,
				Attempts: 0,
			}, nil
		}
		mockRepo.UpdateUserPasswordFn = func(ctx context.Context, id uuid.UUID, passwordHash *string) error {
			if id == testUserID {
				passwordUpdated = true
			}
			return nil
		}
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
			if id == testOTPID {
				otpMarkedUsed = true
			}
			return nil
		}
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			if id == testUserID {
				emailVerified = true
			}
			return nil
		}
		mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
			if userID == testUserID {
				tokensRevoked = true
			}
			return nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ResetPassword(ctx, "jane@example.com", rawCode, "brandNewPassword123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passwordUpdated || !otpMarkedUsed || !emailVerified || !tokensRevoked {
			t.Fatalf("expected passwordUpdated=%v, otpMarkedUsed=%v, emailVerified=%v, tokensRevoked=%v",
				passwordUpdated, otpMarkedUsed, emailVerified, tokensRevoked)
		}
	})

	t.Run("rejected when password is too short", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.MinLength = 10

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ResetPassword(ctx, "jane@example.com", "123456", "short")
		if !errors.Is(err, autherr.ErrPasswordTooShort) {
			t.Fatalf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("rejected when user is not found", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ResetPassword(ctx, "missing@example.com", "123456", "validPassword123")
		if !errors.Is(err, autherr.ErrOTPNotFound) {
			t.Fatalf("expected ErrOTPNotFound, got %v", err)
		}
	})

	t.Run("rejected when OTP attempts exceed maximum", func(t *testing.T) {
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
				CodeHash: "hash",
				Attempts: 5,
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ResetPassword(ctx, "jane@example.com", "123456", "validPassword123")
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
				CodeHash: utils.HashToken("111222"),
				Attempts: 1,
			}, nil
		}

		incrementCalled := false
		mockRepo.IncrementOTPAttemptsFn = func(ctx context.Context, id uuid.UUID) (int32, error) {
			incrementCalled = true
			return 2, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ResetPassword(ctx, "jane@example.com", "999999", "validPassword123")
		if !errors.Is(err, autherr.ErrOTPIncorrect) {
			t.Fatalf("expected ErrOTPIncorrect, got %v", err)
		}
		if !incrementCalled {
			t.Fatal("expected IncrementOTPAttempts to be called")
		}
	})
}

func TestPasswordService_ChangePassword(t *testing.T) {
	ctx := context.Background()

	oldPassword := "currentPassword123"
	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.MinCost)
	hashedStr := string(hashedBytes)

	t.Run("successful change updates password and revokes all refresh tokens", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		cfg.Password.BcryptCost = bcrypt.MinCost

		testUserID := uuid.New()
		passwordUpdated := false
		tokensRevoked := false

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:           testUserID,
				PasswordHash: pgtype.Text{String: hashedStr, Valid: true},
			}, nil
		}
		mockRepo.UpdateUserPasswordFn = func(ctx context.Context, id uuid.UUID, passwordHash *string) error {
			passwordUpdated = true
			return nil
		}
		mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
			tokensRevoked = true
			return nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ChangePassword(ctx, testUserID, oldPassword, "newDifferentPassword123")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !passwordUpdated || !tokensRevoked {
			t.Fatalf("expected passwordUpdated=%v, tokensRevoked=%v", passwordUpdated, tokensRevoked)
		}
	})

	t.Run("rejected when user is OAuth-only", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:           testUserID,
				PasswordHash: pgtype.Text{Valid: false},
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ChangePassword(ctx, testUserID, oldPassword, "newPassword123")
		if !errors.Is(err, autherr.ErrOAuthAccount) {
			t.Fatalf("expected ErrOAuthAccount, got %v", err)
		}
	})

	t.Run("rejected when old password does not match", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:           testUserID,
				PasswordHash: pgtype.Text{String: hashedStr, Valid: true},
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ChangePassword(ctx, testUserID, "incorrectOldPassword", "newPassword123")
		if !errors.Is(err, autherr.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejected when new password is identical to old password", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:           testUserID,
				PasswordHash: pgtype.Text{String: hashedStr, Valid: true},
			}, nil
		}

		passwordService := NewPasswordService(mockRepo, cfg, nil)
		err := passwordService.ChangePassword(ctx, testUserID, oldPassword, oldPassword)
		if !errors.Is(err, autherr.ErrPasswordSame) {
			t.Fatalf("expected ErrPasswordSame, got %v", err)
		}
	})
}
