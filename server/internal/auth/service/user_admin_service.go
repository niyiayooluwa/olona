package service

import (
	"context"

	"github.com/google/uuid"

	"server/internal/auth/repository"
)

// UserAdminService handles administrative user lifecycle operations including suspension, deactivation, and account deletion.
type UserAdminService struct {
	repo repository.AuthRepository
}

// NewUserAdminService constructs a new UserAdminService.
func NewUserAdminService(repo repository.AuthRepository) *UserAdminService {
	return &UserAdminService{
		repo: repo,
	}
}

// SuspendUser marks a user account as suspended and immediately revokes all their active sessions.
//
// Security guarantees:
// - Sets is_suspended = true in the database.
// - Immediately revokes all active refresh tokens so the user is instantly logged out of all devices.
// - Subsequent attempts to log in or refresh tokens will fail with ErrAccountSuspended.
func (service *UserAdminService) SuspendUser(ctx context.Context, userID uuid.UUID) error {
	// 1. Set suspended flag in database
	if err := service.repo.SuspendUser(ctx, userID); err != nil {
		return err
	}

	// 2. Security: Revoke all active refresh tokens immediately
	return service.repo.RevokeAllUserRefreshTokens(ctx, userID)
}

// UnsuspendUser removes the suspension flag from a user account, restoring their ability to log in.
func (service *UserAdminService) UnsuspendUser(ctx context.Context, userID uuid.UUID) error {
	return service.repo.UnsuspendUser(ctx, userID)
}

// DeactivateUser marks an account as deactivated and immediately terminates all active sessions.
//
// Security guarantees:
// - Sets is_deactivated = true in the database.
// - Revokes all active refresh tokens so existing sessions cannot refresh.
// - Subsequent login and token refresh requests will fail with ErrAccountDeactivated.
func (service *UserAdminService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	// 1. Set deactivated flag in database
	if err := service.repo.DeactivateUser(ctx, userID); err != nil {
		return err
	}

	// 2. Security: Revoke all active refresh tokens immediately
	return service.repo.RevokeAllUserRefreshTokens(ctx, userID)
}

// DeleteUser permanently deletes a user account, their associated profile, and revokes all refresh tokens.
//
// Security guarantees:
// - First revokes all active refresh tokens.
// - Deletes profile record.
// - Deletes user credentials from the database.
func (service *UserAdminService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return service.repo.WithTx(ctx, func(txRepo repository.AuthRepository) error {
		// 1. Revoke active refresh tokens
		if err := txRepo.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
			return err
		}

		// 2. Delete user profile record
		if err := txRepo.DeleteProfile(ctx, userID); err != nil {
			return err
		}

		// 3. Delete user account record
		return txRepo.DeleteUser(ctx, userID)
	})
}
