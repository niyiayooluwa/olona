package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service/helpers"
	"server/internal/auth/utils"
)

// PasswordService handles password reset requests, reset confirmations, and password changes.
type PasswordService struct {
	repo   repository.AuthRepository
	config config.AuthConfig
	mailer model.Mailer
}

// NewPasswordService constructs a new PasswordService.
func NewPasswordService(
	repo repository.AuthRepository,
	cfg config.AuthConfig,
	mailer model.Mailer,
) *PasswordService {
	return &PasswordService{
		repo:   repo,
		config: cfg,
		mailer: mailer,
	}
}

// ForgotPassword initiates a self-service password reset flow for the user identified by email.
//
// Security & Anti-Enumeration:
// To prevent malicious actors from enumerating registered user emails, this function returns
// nil without an error if:
// 1. The email address is not registered in the system.
// 2. The account was created via OAuth and has no password set (user.PasswordHash.Valid is false).
// 3. The account is suspended or deactivated.
//
// When a legitimate account is found:
// - Verifies the resend cooldown window against any active reset OTP to prevent email spam.
// - Generates a secure 6-digit numeric OTP and stores its SHA-256 hash with purpose "password_reset".
// - Dispatches the plain code to the user's inbox via Mailer.SendPasswordReset.
//
// Returns ErrOTPCooldown if a code was requested too recently.
func (service *PasswordService) ForgotPassword(ctx context.Context, email string) error {
	// 1. Normalize email address (lowercase and trim whitespace)
	normalizedEmail := helpers.NormalizeEmail(email)

	// 2. Look up user by email; return nil silently on ErrNotFound to mitigate account enumeration
	user, err := service.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	// 3. OAuth-only accounts do not possess a password credential; return nil silently
	if !user.PasswordHash.Valid {
		return nil
	}

	// 4. Do not dispatch reset emails to suspended or deactivated accounts
	if user.IsSuspended || user.IsDeactivated {
		return nil
	}

	// 5. Enforce cooldown to prevent spamming transactional email endpoints.
	// 5-7. Read the previous code, decide, and issue the replacement — atomically.
	//
	// The INSERT belongs inside the transaction for the same reason as in
	// ResendOTP: marking the old code used and inserting its replacement are two
	// statements, and splitting them across the transaction boundary lets two
	// concurrent resends each insert a live code. Whoever holds the row lock
	// must hold it through the insert.
	//
	// The code is generated before the transaction opens since it is pure
	// computation, which keeps the lock window as short as possible.
	newCode, err := utils.GenerateOTP()
	if err != nil {
		return err
	}
	newHash := utils.HashToken(newCode)
	expiresAt := time.Now().Add(service.config.Verification.OTPTTL)

	// Captured rather than returned from fn so a cooldown rejection still lets
	// the not-found branch fall through to issuing a fresh code.
	var resendErr error

	err = service.repo.WithTx(ctx, func(txRepo repository.AuthRepository) error {
		existingOTP, err := txRepo.GetValidOTPForUpdate(ctx, user.ID, "password_reset")
		if err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				// A transient failure here must not fall through to issuing a
				// code: that would put a second live reset OTP in circulation.
				return err
			}
			// No active code on record; fall through and issue one below.
		} else {
			if time.Since(existingOTP.CreatedAt) < service.config.Verification.ResendCooldown {
				resendErr = autherr.ErrOTPCooldown
				return nil
			}
			// Invalidate the previous unconsumed OTP. On failure the transaction
			// aborts, so the old code stays valid and no replacement is issued.
			if err := txRepo.MarkOTPUsed(ctx, existingOTP.ID); err != nil {
				return err
			}
		}

		_, err = txRepo.CreateOTP(ctx, user.ID, newHash, "password_reset", expiresAt)
		return err
	})
	if err != nil {
		return err
	}
	if resendErr != nil {
		return resendErr
	}

	// 8. Deliver the reset code to the user's inbox
	if service.mailer != nil {
		return service.mailer.SendPasswordReset(ctx, normalizedEmail, newCode)
	}

	return nil
}

// ResetPassword consumes a password reset OTP and assigns a new password to the user.
//
// Security guarantees:
//   - Enforces password length policy (min length up to 72 bytes).
//   - Acquires a pessimistic row-level lock (SELECT FOR UPDATE) on the OTP record to prevent
//     concurrent race conditions or duplicate submissions.
//   - Enforces maximum attempt thresholds (VerificationConfig.MaxOTPAttempts) to defend against brute force.
//   - Successfully resetting password automatically verifies the email address (possession of inbox proven).
//   - Revokes ALL active refresh tokens across all devices to immediately expel unauthorized sessions.
//
// Returns:
// - ErrPasswordTooShort / ErrPasswordTooLong if newPassword violates length constraints.
// - ErrOTPNotFound if the code does not exist or has expired.
// - ErrOTPMaxAttempts if failed verification attempts exceed the configured limit.
// - ErrOTPIncorrect if the code does not match (increments failed attempt counter).
// - ErrAccountSuspended / ErrAccountDeactivated if account is locked.
func (service *PasswordService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	// 1. Validate password policy constraints
	if err := helpers.ValidatePassword(newPassword, service.config.Password.MinLength); err != nil {
		return err
	}

	normalizedEmail := helpers.NormalizeEmail(email)

	// 2. Fetch target user
	user, err := service.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return autherr.ErrOTPNotFound
		}
		return err
	}

	// 3. Verify user status
	if user.IsSuspended {
		return autherr.ErrAccountSuspended
	}
	if user.IsDeactivated {
		return autherr.ErrAccountDeactivated
	}

	// 4. Hash the new password before opening the transaction. bcrypt at the
	// configured cost takes ~260ms, and the transaction below holds a row lock
	// that concurrent reset attempts block on. Hashing inside would make every
	// competing request wait out our key-stretching.
	hashedPassword, err := utils.HashPassword(newPassword, service.config.Password.BcryptCost)
	if err != nil {
		return err
	}

	// 5-9. Lock the OTP row, decide, and apply the reset — atomically.
	//
	// The lock in GetValidOTPForUpdate is only held for the life of the
	// enclosing transaction. Without one, concurrent requests all read the same
	// attempt count before any of them increments it, so the brute-force limit
	// binds sequential guesses only. On a password reset that is an account
	// takeover vector, not just a rate-limit bypass.
	//
	// As in VerifyEmail, the outcome is captured in a variable instead of
	// returned from fn: a wrong code must still commit its attempt increment,
	// and returning an error from fn would roll it back.
	var resetErr error

	err = service.repo.WithTx(ctx, func(txRepo repository.AuthRepository) error {
		otpRecord, err := txRepo.GetValidOTPForUpdate(ctx, user.ID, "password_reset")
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				resetErr = autherr.ErrOTPNotFound
				return nil
			}
			return err
		}

		if otpRecord.Attempts >= int32(service.config.Verification.MaxOTPAttempts) {
			resetErr = autherr.ErrOTPMaxAttempts
			return nil
		}

		if utils.HashToken(code) != otpRecord.CodeHash {
			if _, err := txRepo.IncrementOTPAttempts(ctx, otpRecord.ID); err != nil {
				return err
			}
			resetErr = autherr.ErrOTPIncorrect
			return nil
		}

		// The code is valid, so the reset proceeds. Persisting the new password
		// without consuming the code would leave a live reset token sitting in
		// the table — these steps belong in one transaction for that reason
		// alone, independent of the locking above.
		if err := txRepo.UpdateUserPassword(ctx, user.ID, &hashedPassword); err != nil {
			return err
		}
		if err := txRepo.MarkOTPUsed(ctx, otpRecord.ID); err != nil {
			return err
		}

		// Resetting via email OTP demonstrates inbox ownership; auto-verify.
		if user.EmailVerifiedAt == nil {
			if err := txRepo.MarkEmailVerified(ctx, user.ID); err != nil {
				return err
			}
		}

		// Invalidate every existing session across all devices.
		if err := txRepo.RevokeAllUserRefreshTokens(ctx, user.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return resetErr
}

// ChangePassword updates an authenticated user's password using their current password.
//
// Security guarantees:
// - Verifies the caller's old password using constant-time bcrypt comparison.
// - Rejects requests where the new password is identical to the old password.
// - Rejects OAuth-only accounts that have never configured a password.
// - Upon success, revokes ALL active refresh tokens so existing sessions must re-authenticate.
//
// Returns:
// - ErrPasswordTooShort / ErrPasswordTooLong if newPassword violates length constraints.
// - ErrOAuthAccount if user signed up with OAuth and has no password set.
// - ErrInvalidCredentials if oldPassword does not match the current password hash.
// - ErrPasswordSame if newPassword is identical to oldPassword.
func (service *PasswordService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// 1. Validate new password length constraints
	if err := helpers.ValidatePassword(newPassword, service.config.Password.MinLength); err != nil {
		return err
	}

	// 2. Fetch user record
	user, err := service.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// 3. OAuth-only accounts must set a password via ForgotPassword / ResetPassword flow
	if !user.PasswordHash.Valid {
		return autherr.ErrOAuthAccount
	}

	// 4. Verify existing password with constant-time bcrypt comparison
	if !utils.CheckPassword(user.PasswordHash.String, oldPassword) {
		return autherr.ErrInvalidCredentials
	}

	// 5. Disallow recycling identical password
	if oldPassword == newPassword {
		return autherr.ErrPasswordSame
	}

	// 6. Hash new password with configured bcrypt cost
	hashedPassword, err := utils.HashPassword(newPassword, service.config.Password.BcryptCost)
	if err != nil {
		return err
	}

	// 7. Update password hash in database
	if err := service.repo.UpdateUserPassword(ctx, userID, &hashedPassword); err != nil {
		return err
	}

	// 8. Security: Revoke all existing sessions so previous refresh tokens cannot be used
	if err := service.repo.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return err
	}

	return nil
}
