package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"server/internal/auth/token"
	db "server/internal/platform/db/generated"
	"server/internal/workspace/model"
	"server/internal/workspace/repository"
	"server/internal/workspace/wksperr"
)

// AcceptInvitation consumes a single-use invitation token to add an authenticated user into the workspace.
//
// Business Rules & Guarantees (v1 Specification §9):
//  1. Token Validity: Token must exist, match hash, and not be expired or already accepted.
//  2. Identity Matching: Authenticated user's email must strictly match the invitation recipient email.
//  3. Single Atomic Transaction:
//     - Marks invitation as accepted to prevent token replay attacks.
//     - Adds user to workspace_members with the assigned tier and capabilities.
//     - Emits an audit log entry ("member.joined").
func (service *WorkspaceService) AcceptInvitation(
	ctx context.Context,
	rawToken string,
	authenticatedUserEmail string,
	authenticatedUserID uuid.UUID,
) error {
	tokenHash := token.HashToken(rawToken)

	// Step 1: Look up invitation by token hash.
	invitation, err := service.repo.GetWorkspaceInvitationByToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return wksperr.ErrInvitationNotFound
		}
		return err
	}

	// Step 2: Strict identity check (§9). Token forwarding cannot transfer access.
	if invitation.Email != authenticatedUserEmail {
		return wksperr.ErrInvitationEmailMismatch
	}

	// Step 3: Parse capabilities stored on the invitation.
	caps, err := parseCapabilities(invitation.Capabilities)
	if err != nil {
		return err
	}

	// Step 4: Execute atomic acceptance & membership insertion.
	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		// Mark invitation accepted.
		if txErr := txRepo.MarkWorkspaceInvitationAccepted(ctx, invitation.ID); txErr != nil {
			return txErr
		}

		// Insert workspace member record.
		_, txErr := txRepo.AddWorkspaceMember(
			ctx,
			invitation.WorkspaceID,
			authenticatedUserID,
			invitation.Tier,
			caps,
			&invitation.InvitedBy,
		)
		if txErr != nil {
			return txErr
		}

		// Record audit entry.
		auditMeta := map[string]interface{}{"invitation_id": invitation.ID, "tier": invitation.Tier}
		_, txErr = txRepo.CreateAuditLog(
			ctx,
			invitation.WorkspaceID,
			&authenticatedUserID,
			"member.joined",
			"user",
			&authenticatedUserID,
			auditMeta,
		)
		return txErr
	})
}

// UpdateMemberRole modifies a non-owner member's tier and capability set.
//
// Business Rules & Guarantees (§4 & §9):
//  1. Authority: Caller must have "workspace.manage_members" capability on an active workspace.
//  2. Tier Safety: Cannot promote to "owner" here. Ownership handover MUST occur via Two-Key control.
//  3. Transactional Audit: Updating member and writing audit log occur atomically.
func (service *WorkspaceService) UpdateMemberRole(
	ctx context.Context,
	workspaceID, targetUserID, callerID uuid.UUID,
	newTier db.WorkspaceTier,
	newCapabilities []string,
) (*db.WorkspaceMember, error) {
	// Guard: Caller needs manage_members permission.
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceManageMembers, true)
	if err != nil {
		return nil, err
	}

	// Rule: Cannot assign "owner" tier through standard member role editing.
	if newTier == db.WorkspaceTierOwner {
		return nil, wksperr.ErrForbidden
	}

	var updatedMember db.WorkspaceMember
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error
		updatedMember, txErr = txRepo.UpdateMemberTierAndCapabilities(ctx, workspaceID, targetUserID, newTier, newCapabilities)
		if txErr != nil {
			return txErr
		}

		auditMeta := map[string]interface{}{"new_tier": newTier, "capabilities": newCapabilities}
		_, txErr = txRepo.CreateAuditLog(
			ctx,
			workspaceID,
			&callerID,
			"member.role_updated",
			"user",
			&targetUserID,
			auditMeta,
		)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &updatedMember, nil
}

// RemoveMember removes a user from the workspace or handles voluntary departure ("leave workspace").
//
// Business Rules & Guarantees (§9):
//  1. Authority: Allowed if caller is removing themselves (leaving) OR caller has "workspace.manage_members".
//  2. Non-Orphaning Guarantee (§5 Rule 13): The sole active owner CANNOT leave without first transferring ownership.
//  3. Historical Integrity: Membership status is set to "removed" (never hard deleted), preserving audit trail links.
func (service *WorkspaceService) RemoveMember(
	ctx context.Context,
	workspaceID, targetUserID, callerID uuid.UUID,
) error {
	// Guard: If removing someone else, caller must have manage_members permission.
	isSelf := (callerID == targetUserID)
	if !isSelf {
		_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceManageMembers, true)
		if err != nil {
			return err
		}
	}

	// Verify target member exists and check owner departure rule.
	targetMember, err := service.repo.GetActiveMember(ctx, workspaceID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return wksperr.ErrWorkspaceNotFound
		}
		return err
	}

	// Rule: Owner cannot leave without transferring ownership (§5).
	if targetMember.Tier == db.WorkspaceTierOwner {
		return wksperr.ErrCannotLeaveAsOwner
	}

	// Execute removal and audit atomically.
	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		if txErr := txRepo.RemoveMember(ctx, workspaceID, targetUserID); txErr != nil {
			return txErr
		}

		action := "member.removed"
		if isSelf {
			action = "member.left"
		}

		auditMeta := map[string]interface{}{"removed_user_id": targetUserID, "is_voluntary": isSelf}
		_, txErr := txRepo.CreateAuditLog(
			ctx,
			workspaceID,
			&callerID,
			action,
			"user",
			&targetUserID,
			auditMeta,
		)
		return txErr
	})
}

// RevokeInvitation cancels a pending workspace invitation.
func (service *WorkspaceService) RevokeInvitation(
	ctx context.Context,
	workspaceID, invitationID, callerID uuid.UUID,
) error {
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceManageMembers, true)
	if err != nil {
		return err
	}

	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		if txErr := txRepo.RevokeWorkspaceInvitation(ctx, invitationID); txErr != nil {
			return txErr
		}

		_, txErr := txRepo.CreateAuditLog(
			ctx,
			workspaceID,
			&callerID,
			"invitation.revoked",
			"invitation",
			&invitationID,
			nil,
		)
		return txErr
	})
}

// ListPendingInvitations lists all active, unaccepted invitations for a workspace.
func (service *WorkspaceService) ListPendingInvitations(
	ctx context.Context,
	workspaceID, callerID uuid.UUID,
) ([]db.WorkspaceInvitation, error) {
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceManageMembers, false)
	if err != nil {
		return nil, err
	}

	return service.repo.ListPendingInvitations(ctx, workspaceID)
}
