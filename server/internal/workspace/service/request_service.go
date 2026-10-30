package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	db "server/internal/platform/db/generated"
	"server/internal/workspace/model"
	"server/internal/workspace/repository"
	"server/internal/workspace/wksperr"
)

// CreateRequest opens a formal Two-Key Control governance request for sensitive lifecycle actions.
//
// Supported Request Types (§8):
//   - "archive": Requires parent oversight approval. Blocked if active children exist.
//   - "delete": Schedules 30-day pending_deletion purge. Blocked if active children exist.
//   - "transfer_ownership": Hands over sole ownership to an existing active admin of this workspace.
//   - "move": Reparents the workspace subtree under a new parent workspace within the same tree.
//
// Business Rules & Guarantees:
//   1. Authority: Requester must possess the matching "workspace.request_*" capability.
//   2. Expiration: Requests default to a 14-day validity window (§15 Open Decision 6).
//   3. Prerequisite validation (e.g. active child blocking, admin candidate verification) happens upfront.
func (service *WorkspaceService) CreateRequest(
	ctx context.Context,
	workspaceID, callerID uuid.UUID,
	reqType db.WorkspaceRequestType,
	reason *string,
	targetUserID, newParentID *uuid.UUID,
) (*db.WorkspaceRequest, error) {
	// Guard 1: Verify caller has the matching request_* capability.
	var requiredCap string
	switch reqType {
	case db.WorkspaceRequestTypeArchive:
		requiredCap = model.CapWorkspaceRequestArchive
	case db.WorkspaceRequestTypeDelete:
		requiredCap = model.CapWorkspaceRequestDelete
	case db.WorkspaceRequestTypeTransferOwnership:
		requiredCap = model.CapWorkspaceRequestTransfer
	case db.WorkspaceRequestTypeMove:
		requiredCap = model.CapWorkspaceRequestMove
	default:
		return nil, wksperr.ErrForbidden
	}

	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, requiredCap, true)
	if err != nil {
		return nil, err
	}

	// Rule: Active children block archive and delete requests (§3 Rule 18 & §15 Open Decision 5).
	if reqType == db.WorkspaceRequestTypeArchive || reqType == db.WorkspaceRequestTypeDelete {
		childCount, err := service.repo.CountActiveChildren(ctx, &workspaceID)
		if err != nil {
			return nil, err
		}
		if childCount > 0 {
			return nil, wksperr.ErrActiveChildren
		}
	}

	// Rule: For ownership transfer, target user must already be an active admin of THIS workspace (§3 Rule 9).
	if reqType == db.WorkspaceRequestTypeTransferOwnership {
		if targetUserID == nil {
			return nil, wksperr.ErrInvalidRequestTarget
		}
		targetMember, err := service.repo.GetActiveMember(ctx, workspaceID, *targetUserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, wksperr.ErrInvalidRequestTarget
			}
			return nil, err
		}
		if targetMember.Tier != db.WorkspaceTierAdmin {
			return nil, wksperr.ErrInvalidRequestTarget
		}
	}

	// Rule: For move, new parent must exist and belong to the same root organization tree.
	if reqType == db.WorkspaceRequestTypeMove {
		if newParentID == nil {
			return nil, wksperr.ErrForbidden
		}
		currentWs, err := service.repo.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		newParent, err := service.repo.GetWorkspaceByID(ctx, *newParentID)
		if err != nil {
			return nil, err
		}
		if currentWs.RootID != newParent.RootID {
			return nil, wksperr.ErrForbidden // Cannot move across different organizations.
		}
	}

	// Default 14-day expiry.
	expiresAt := time.Now().Add(14 * 24 * time.Hour)

	var createdRequest db.WorkspaceRequest
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error
		createdRequest, txErr = txRepo.CreateWorkspaceRequest(
			ctx,
			workspaceID,
			reqType,
			callerID,
			reason,
			targetUserID,
			newParentID,
			expiresAt,
			false, // soleApprover default
		)
		if txErr != nil {
			return txErr
		}

		auditMeta := map[string]interface{}{"request_id": createdRequest.ID, "type": reqType, "reason": reason}
		_, txErr = txRepo.CreateAuditLog(
			ctx,
			workspaceID,
			&callerID,
			"request.created",
			"request",
			&createdRequest.ID,
			auditMeta,
		)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &createdRequest, nil
}

// ListPendingRequests retrieves pending governance requests awaiting the caller's oversight decision.
//
// Structural Separation of Duties (§3 Rule 11 & §8):
//  - Returns requests for child workspaces where caller possesses oversight on the IMMEDIATE parent.
//  - Automatically filters out requests authored by the caller (requester != approver).
func (service *WorkspaceService) ListPendingRequests(
	ctx context.Context,
	callerID uuid.UUID,
) ([]db.ListPendingRequestsForApproverRow, error) {
	return service.repo.ListPendingRequestsForApprover(ctx, callerID)
}

// ListWorkspaceRequests retrieves all pending requests opened inside a specific workspace.
func (service *WorkspaceService) ListWorkspaceRequests(
	ctx context.Context,
	workspaceID, callerID uuid.UUID,
) ([]db.ListPendingRequestsForWorkspaceRow, error) {
	// Guard: Caller needs view permissions.
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceView, false)
	if err != nil {
		return nil, err
	}

	return service.repo.ListPendingRequestsForWorkspace(ctx, workspaceID)
}

// ApproveRequest executes a pending Two-Key Control request.
//
// Business Rules & Guarantees (§3 & §8):
//  1. Status & Expiration: Request must be in "pending" status and not expired.
//  2. Separation of Duties (§3 Rule 11): Requester CANNOT approve their own request (unless sole_approver = true).
//  3. Oversight Authority: Approver must possess "workspace.oversee_child" on the immediate parent workspace.
//  4. Atomic State Execution:
//     - "archive": flips workspace status to "archived".
//     - "delete": flips workspace status to "pending_deletion" (schedules 30-day grace period).
//     - "transfer_ownership": demotes existing owner to admin, promotes target admin to owner.
//     - "move": updates parent_id and atomically rewrites closure table ancestry.
//     - Marks request status as "approved".
//     - Emits structured audit log entries.
func (service *WorkspaceService) ApproveRequest(
	ctx context.Context,
	requestID, callerID uuid.UUID,
) error {
	request, err := service.repo.GetWorkspaceRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return wksperr.ErrRequestNotFound
		}
		return err
	}

	if request.Status != db.WorkspaceRequestStatusPending || time.Now().After(request.ExpiresAt) {
		return wksperr.ErrRequestNotFound
	}

	// Rule 11: Requester != Approver.
	if request.RequestedBy == callerID && !request.SoleApprover {
		return wksperr.ErrRequesterIsApprover
	}

	// Verify approver holds oversight on the immediate parent workspace.
	parentID, err := service.repo.GetImmediateParentClosure(ctx, request.WorkspaceID)
	if err != nil {
		return err
	}

	_, err = service.evaluator.EnforceAccess(ctx, callerID, parentID, model.CapWorkspaceOverseeChild, false)
	if err != nil {
		return err
	}

	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		// 1. Execute requested state mutation
		switch request.Type {
		case db.WorkspaceRequestTypeArchive:
			if _, txErr := txRepo.SetWorkspaceStatus(ctx, request.WorkspaceID, db.WorkspaceStatusArchived); txErr != nil {
				return txErr
			}

		case db.WorkspaceRequestTypeDelete:
			if _, txErr := txRepo.SetWorkspaceStatus(ctx, request.WorkspaceID, db.WorkspaceStatusPendingDeletion); txErr != nil {
				return txErr
			}

		case db.WorkspaceRequestTypeTransferOwnership:
			if !request.TargetUserID.Valid {
				return wksperr.ErrInvalidRequestTarget
			}
			targetID := uuid.UUID(request.TargetUserID.Bytes)
			currentOwner, txErr := txRepo.GetActiveOwner(ctx, request.WorkspaceID)
			if txErr != nil {
				return txErr
			}
			if txErr := txRepo.TransferOwnershipInWorkspace(ctx, request.WorkspaceID, currentOwner.UserID, targetID); txErr != nil {
				return txErr
			}

		case db.WorkspaceRequestTypeMove:
			if !request.NewParentID.Valid {
				return wksperr.ErrForbidden
			}
			newParentUUID := uuid.UUID(request.NewParentID.Bytes)

			// Step A: Disconnect old ancestors.
			if txErr := txRepo.DeleteSubtreeClosureForMove(ctx, request.WorkspaceID); txErr != nil {
				return txErr
			}
			// Step B: Connect new ancestors.
			if txErr := txRepo.RebuildClosureForMovedSubtree(ctx, request.WorkspaceID, newParentUUID); txErr != nil {
				return txErr
			}
			// Step C: Update parent_id pointer on workspace.
			if _, txErr := txRepo.MoveWorkspace(ctx, request.WorkspaceID, &newParentUUID); txErr != nil {
				return txErr
			}
		}

		// 2. Mark request as approved
		if _, txErr := txRepo.DecideWorkspaceRequest(ctx, requestID, db.WorkspaceRequestStatusApproved, &callerID); txErr != nil {
			return txErr
		}

		// 3. Record audit trail
		auditMeta := map[string]interface{}{"request_id": requestID, "type": request.Type, "approved_by": callerID}
		_, txErr := txRepo.CreateAuditLog(
			ctx,
			request.WorkspaceID,
			&callerID,
			"request.approved",
			"request",
			&requestID,
			auditMeta,
		)
		return txErr
	})
}

// RejectRequest denies a pending Two-Key Control request.
func (service *WorkspaceService) RejectRequest(
	ctx context.Context,
	requestID, callerID uuid.UUID,
	reason *string,
) error {
	request, err := service.repo.GetWorkspaceRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return wksperr.ErrRequestNotFound
		}
		return err
	}

	if request.Status != db.WorkspaceRequestStatusPending || time.Now().After(request.ExpiresAt) {
		return wksperr.ErrRequestNotFound
	}

	// Rule 11: Requester != Decider.
	if request.RequestedBy == callerID && !request.SoleApprover {
		return wksperr.ErrRequesterIsApprover
	}

	// Verify oversight authority on immediate parent.
	parentID, err := service.repo.GetImmediateParentClosure(ctx, request.WorkspaceID)
	if err != nil {
		return err
	}

	_, err = service.evaluator.EnforceAccess(ctx, callerID, parentID, model.CapWorkspaceOverseeChild, false)
	if err != nil {
		return err
	}

	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		if _, txErr := txRepo.DecideWorkspaceRequest(ctx, requestID, db.WorkspaceRequestStatusRejected, &callerID); txErr != nil {
			return txErr
		}

		auditMeta := map[string]interface{}{"request_id": requestID, "rejection_reason": reason}
		_, txErr := txRepo.CreateAuditLog(
			ctx,
			request.WorkspaceID,
			&callerID,
			"request.rejected",
			"request",
			&requestID,
			auditMeta,
		)
		return txErr
	})
}

// CancelRequest allows the original author to withdraw a pending request.
func (service *WorkspaceService) CancelRequest(
	ctx context.Context,
	requestID, callerID uuid.UUID,
) error {
	return service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		req, txErr := txRepo.CancelWorkspaceRequest(ctx, requestID, callerID)
		if txErr != nil {
			return txErr
		}

		_, txErr = txRepo.CreateAuditLog(
			ctx,
			req.WorkspaceID,
			&callerID,
			"request.cancelled",
			"request",
			&requestID,
			nil,
		)
		return txErr
	})
}
