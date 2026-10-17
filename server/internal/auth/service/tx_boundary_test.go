package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	autherr "server/internal/auth/errors"
	"server/internal/auth/repository"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

// These tests pin down the transaction boundary the OTP paths depend on.
//
// The bug they guard against is not hypothetical: with autocommit,
// SELECT ... FOR UPDATE releases its row lock the instant the statement
// finishes, so N concurrent wrong guesses all read attempts=0 and every one of
// them passes the MaxOTPAttempts gate. Measured against a live Postgres, 13-20
// of 20 concurrent guesses were evaluated against a limit of 5. Wrapping the
// read-decide-write sequence in one transaction holds the limit at exactly 5.

func TestVerifyEmailRunsTheOTPFlowInOneTransaction(t *testing.T) {
	ctx := context.Background()
	jwtIssuer := setupTestJWTIssuer()

	newService := func(t *testing.T) (*SignupService, *mockAuthRepository, uuid.UUID, string) {
		t.Helper()
		mockRepo := &mockAuthRepository{}
		userID := uuid.New()
		rawCode := "123456"
		codeHash := utils.HashToken(rawCode)

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: userID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:       uuid.New(),
				UserID:   id,
				CodeHash: codeHash,
				Purpose:  "signup_verify",
				Attempts: 0,
			}, nil
		}
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error { return nil }
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error { return nil }
		mockRepo.CreateRefreshTokenFn = func(ctx context.Context, id uuid.UUID, hash string, exp time.Time) (db.RefreshToken, error) {
			return db.RefreshToken{ID: uuid.New(), UserID: id}, nil
		}

		return NewSignupService(mockRepo, setupTestConfig(), jwtIssuer, nil), mockRepo, userID, rawCode
	}

	t.Run("success path", func(t *testing.T) {
		svc, mockRepo, _, rawCode := newService(t)

		txCalls := 0
		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			txCalls++
			return fn(mockRepo)
		}

		if _, err := svc.VerifyEmail(ctx, "jane@example.com", rawCode); err != nil {
			t.Fatalf("VerifyEmail() = %v, want nil", err)
		}
		if txCalls != 1 {
			t.Errorf("WithTx called %d times, want 1 — the OTP read-decide-write must be a single transaction", txCalls)
		}
	})

	t.Run("wrong code commits its attempt increment", func(t *testing.T) {
		svc, mockRepo, _, _ := newService(t)

		incremented := false
		mockRepo.IncrementOTPAttemptsFn = func(ctx context.Context, id uuid.UUID) (int32, error) {
			incremented = true
			return 1, nil
		}

		// Model a rollback: WithTx discards everything fn did when the hook
		// returns an error, exactly as a real rollback does.
		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			if err := fn(mockRepo); err != nil {
				incremented = false // rolled back
				return err
			}
			return nil
		}

		_, err := svc.VerifyEmail(ctx, "jane@example.com", "000000")

		if !errors.Is(err, autherr.ErrOTPIncorrect) {
			t.Fatalf("VerifyEmail() = %v, want ErrOTPIncorrect", err)
		}
		// The wrong-code path must return its outcome through a variable, not
		// via fn's error. If it returned the error from fn, the increment
		// would roll back and the counter would never advance — leaving the
		// brute-force limit permanently open.
		if !incremented {
			t.Error("attempt increment was rolled back; a wrong guess must still commit its increment")
		}
	})

	t.Run("OTP consumed only if the whole flow succeeds", func(t *testing.T) {
		svc, mockRepo, _, rawCode := newService(t)

		markedUsed := false
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
			markedUsed = true
			return nil
		}
		verifyErr := errors.New("mark email verified failed")
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error {
			return verifyErr
		}

		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			if err := fn(mockRepo); err != nil {
				markedUsed = false // rolled back
				return err
			}
			return nil
		}

		_, err := svc.VerifyEmail(ctx, "jane@example.com", rawCode)

		if !errors.Is(err, verifyErr) {
			t.Fatalf("VerifyEmail() = %v, want %v", err, verifyErr)
		}
		if markedUsed {
			t.Error("OTP was consumed even though the flow failed; a consumed code must not survive a failed verification")
		}
	})
}

func TestResetPasswordRunsTheOTPFlowInOneTransaction(t *testing.T) {
	ctx := context.Background()

	newService := func(t *testing.T) (*PasswordService, *mockAuthRepository, string) {
		t.Helper()
		mockRepo := &mockAuthRepository{}
		userID := uuid.New()
		rawCode := "123456"

		mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
			return db.User{ID: userID, Email: email}, nil
		}
		mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
			return db.OtpCode{
				ID:       uuid.New(),
				UserID:   id,
				CodeHash: utils.HashToken(rawCode),
				Purpose:  "password_reset",
				Attempts: 0,
			}, nil
		}
		mockRepo.UpdateUserPasswordFn = func(ctx context.Context, id uuid.UUID, passwordHash *string) error { return nil }
		mockRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error { return nil }
		mockRepo.MarkEmailVerifiedFn = func(ctx context.Context, id uuid.UUID) error { return nil }
		mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, id uuid.UUID) error { return nil }

		return NewPasswordService(mockRepo, setupTestConfig(), nil), mockRepo, rawCode
	}

	t.Run("success path runs in one transaction", func(t *testing.T) {
		svc, mockRepo, rawCode := newService(t)

		txCalls := 0
		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			txCalls++
			return fn(mockRepo)
		}

		if err := svc.ResetPassword(ctx, "jane@example.com", rawCode, "New-Password-1!"); err != nil {
			t.Fatalf("ResetPassword() = %v, want nil", err)
		}
		if txCalls != 1 {
			t.Errorf("WithTx called %d times, want 1", txCalls)
		}
	})

	t.Run("session revocation failure aborts the reset", func(t *testing.T) {
		// The old code discarded this error entirely, so a reset could report
		// success while the attacker's existing sessions stayed valid.
		svc, mockRepo, rawCode := newService(t)

		revokeErr := errors.New("revoke failed")
		mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, id uuid.UUID) error {
			return revokeErr
		}

		passwordChanged := false
		mockRepo.UpdateUserPasswordFn = func(ctx context.Context, id uuid.UUID, passwordHash *string) error {
			passwordChanged = true
			return nil
		}
		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			if err := fn(mockRepo); err != nil {
				passwordChanged = false // rolled back
				return err
			}
			return nil
		}

		err := svc.ResetPassword(ctx, "jane@example.com", rawCode, "New-Password-1!")

		if !errors.Is(err, revokeErr) {
			t.Fatalf("ResetPassword() = %v, want %v", err, revokeErr)
		}
		if passwordChanged {
			t.Error("password was persisted despite the failure; leaving sessions live is not a recoverable state")
		}
	})

	t.Run("wrong code commits its attempt increment", func(t *testing.T) {
		svc, mockRepo, _ := newService(t)

		incremented := false
		mockRepo.IncrementOTPAttemptsFn = func(ctx context.Context, id uuid.UUID) (int32, error) {
			incremented = true
			return 1, nil
		}
		mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
			if err := fn(mockRepo); err != nil {
				incremented = false // rolled back
				return err
			}
			return nil
		}

		err := svc.ResetPassword(ctx, "jane@example.com", "000000", "New-Password-1!")

		if !errors.Is(err, autherr.ErrOTPIncorrect) {
			t.Fatalf("ResetPassword() = %v, want ErrOTPIncorrect", err)
		}
		if !incremented {
			t.Error("attempt increment was rolled back")
		}
	})
}
