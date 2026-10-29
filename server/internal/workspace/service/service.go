package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	db "server/internal/platform/db/generated"
	"server/internal/workspace/model"
	"server/internal/workspace/repository"
	"server/internal/workspace/wksperr"
)

type WorkspaceService struct {
	repo      repository.WorkspaceRepository
	evaluator *AccessEvaluator
}

func NewWorkspaceService(repo repository.WorkspaceRepository, evaluator *AccessEvaluator) *WorkspaceService {
	return &WorkspaceService{
		repo:      repo,
		evaluator: evaluator,
	}
}

func (service *WorkspaceService) CreateRoot(ctx context.Context, name string, description *string, creatorUserID uuid.UUID) (*db.Workspace, error) {
	var createdWorkspace db.Workspace
	workspaceID := uuid.New()

	err := service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error

		// Create the workspace
		createdWorkspace, txErr = txRepo.CreateRootWorkspace(ctx, workspaceID, name, description, creatorUserID)
		if txErr != nil {
			return txErr
		}

		// Insert the root closure row
		if txErr = txRepo.InsertRootClosure(ctx, workspaceID); txErr != nil {
			return txErr
		}

		// Add creator as Owner with the Owner preset capabilities
		_, txErr = txRepo.AddWorkspaceMember(ctx, workspaceID, creatorUserID, db.WorkspaceTierOwner, model.OwnerCapabilities, nil)
		if txErr != nil {
			return txErr
		}

		// Record append only audit log
		auditMetadata := map[string]interface{}{"name": name, "description": description, "is_root": true}
		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &creatorUserID, "workspace.created", "workspace", &workspaceID, auditMetadata)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return &createdWorkspace, nil
}

func (service *WorkspaceService) CreateChild(ctx context.Context, name string, description *string, parentID, targetOwnerUserID, callerUserID uuid.UUID) (*db.Workspace, error) {
	// Guard check: caller must have CapWorkspaceCreateChild on the parent workspace
	_, err := service.evaluator.EnforceAccess(ctx, callerUserID, parentID, model.CapWorkspaceCreateChild, true)
	if err != nil {
		return nil, err
	}

	// Fetch parent workspace to get its root_id and verify it's active
	parentWorkspace, err := service.repo.GetWorkspaceByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parentWorkspace.Status != "active" {
		return nil, wksperr.ErrWorkspaceArchived
	}

	// Single-principal check: if caller is the owner and target is themselves
	isSinglePrincipal := (targetOwnerUserID == callerUserID)
	if !isSinglePrincipal {
		// Rule 5: target owner must already be an active admin of the parent
		targetMember, err := service.repo.GetActiveMember(ctx, parentID, targetOwnerUserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, wksperr.ErrInvalidRelocation
			}
			return nil, err
		}
		if targetMember.Tier != db.WorkspaceTierAdmin {
			return nil, wksperr.ErrInvalidRelocation
		}

		// Rule 6: target owner must NOT be the parent's owner (cannot orphan parent)
		if targetMember.Tier == db.WorkspaceTierOwner {
			return nil, wksperr.ErrCannotOrphanParent
		}
	}

	workspaceID := uuid.New()
	var createdWorkspace db.Workspace

	// 4. Atomic Relocation Transaction
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error

		// Step A: Create child workspace with parent's root_id
		createdWorkspace, txErr = txRepo.CreateChildWorkspace(ctx, workspaceID, parentID, parentWorkspace.RootID, name, description, isSinglePrincipal, callerUserID)
		if txErr != nil {
			return txErr
		}

		// Step B: Clone parent ancestry into closure table + add self row
		if txErr = txRepo.InsertChildClosure(ctx, workspaceID, parentID); txErr != nil {
			return txErr
		}

		// Step C: Add target user as Owner of the new child
		_, txErr = txRepo.AddWorkspaceMember(ctx, workspaceID, targetOwnerUserID, db.WorkspaceTierOwner, model.OwnerCapabilities, &callerUserID)
		if txErr != nil {
			return txErr
		}

		// Step D: THE RELOCATION! (Unless single-principal, remove target's admin seat from parent)
		if !isSinglePrincipal {
			if txErr = txRepo.RemoveMember(ctx, parentID, targetOwnerUserID); txErr != nil {
				return txErr
			}

			// Parent audit log for relocation
			parentAuditMeta := map[string]interface{}{"relocated_to": workspaceID, "promoted_user": targetOwnerUserID}
			_, txErr = txRepo.CreateAuditLog(ctx, parentID, &callerUserID, "member.relocated", "user", &targetOwnerUserID, parentAuditMeta)
			if txErr != nil {
				return txErr
			}
		}

		// Step E: Child audit log for workspace creation
		childAuditMeta := map[string]interface{}{"name": name, "parent_id": parentID, "owner_id": targetOwnerUserID}
		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &callerUserID, "workspace.created", "workspace", &workspaceID, childAuditMeta)
		return txErr
	})

	if err != nil {
		return nil, err
	}
	return &createdWorkspace, nil
}
