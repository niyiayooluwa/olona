package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"server/internal/auth/autherr"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service/helpers"
	"server/internal/auth/token"
)

// InvitationService manages issuing, inspecting, and revoking organization and team invitations.
type InvitationService struct {
	repo   repository.AuthRepository
	mailer model.Mailer
}

// NewInvitationService constructs a new InvitationService.
func NewInvitationService(
	repo repository.AuthRepository,
	mailer model.Mailer,
) *InvitationService {
	return &InvitationService{
		repo:   repo,
		mailer: mailer,
	}
}

// Invite generates a single-use invitation token for a new user and dispatches an invite email.
//
// Behavior:
// - Generates a 32-byte CSPRNG random token and saves its SHA-256 hash in the database.
// - Tokens are valid for a default lifetime of 7 days.
// - Dispatches the raw token in an invite link via Mailer.SendInvite.
//
// Parameters:
// - invitedBy: The UUID of the admin or team member issuing the invitation.
// - email: The recipient's email address.
func (service *InvitationService) Invite(ctx context.Context, invitedBy uuid.UUID, email string) error {
	// 1. Normalize invitee email address
	normalizedEmail := helpers.NormalizeEmail(email)

	// 2. Generate secure 32-byte CSPRNG token and deterministic SHA-256 hash
	rawToken, tokenHash, err := token.GenerateRandomToken()
	if err != nil {
		return err
	}

	// 3. Compute 7-day expiration timestamp
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// 4. Persist invitation in database
	_, err = service.repo.CreateInvitation(ctx, normalizedEmail, tokenHash, invitedBy, expiresAt)
	if err != nil {
		return err
	}

	// 5. Dispatch invite link via transactional mailer
	if service.mailer != nil {
		return service.mailer.SendInvite(ctx, normalizedEmail, rawToken)
	}

	return nil
}

// GetInvitation retrieves invitation details for a given raw invite token without consuming it.
// Used by web frontends to display "Join Olona as user@example.com" before the user chooses a password.
//
// Returns:
// - ErrInvitationNotFound if no invitation with the token's hash exists.
// - ErrInvitationAlreadyAccepted if the invite has already been redeemed.
// - ErrInvitationExpired if the invite lifetime has elapsed.
func (service *InvitationService) GetInvitation(ctx context.Context, rawToken string) (*model.InvitationDetails, error) {
	// 1. Compute SHA-256 hash of presented raw invite token
	tokenHash := token.HashToken(rawToken)

	// 2. Query repository for matching invitation
	invitation, err := service.repo.GetInvitationByToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, autherr.ErrInvitationNotFound
		}
		return nil, err
	}

	// 3. Reject already redeemed invitations
	if invitation.AcceptedAt != nil {
		return nil, autherr.ErrInvitationAlreadyAccepted
	}

	// 4. Reject expired invitations
	if time.Now().After(invitation.ExpiresAt) {
		return nil, autherr.ErrInvitationExpired
	}

	// 5. Return safe preview metadata
	return &model.InvitationDetails{
		ID:        invitation.ID,
		Email:     invitation.Email,
		InvitedBy: invitation.InvitedBy,
		ExpiresAt: invitation.ExpiresAt,
	}, nil
}

// RevokeInvitation deletes an outstanding invitation by ID, preventing it from being redeemed.
func (service *InvitationService) RevokeInvitation(ctx context.Context, invitationID uuid.UUID) error {
	return service.repo.DeleteInvitation(ctx, invitationID)
}
