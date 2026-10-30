// Package service implements domain business logic for Olona workspaces.
// It orchestrates access evaluation, transactions, state lifecycle changes,
// closure tree maintenance, atomic relocation, and audit trail generation.
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

// WorkspaceService handles all lifecycle, hierarchy, and oversight operations on workspaces.
type WorkspaceService struct {
	repo      repository.WorkspaceRepository
	evaluator *AccessEvaluator
}

// NewWorkspaceService constructs a new WorkspaceService.
func NewWorkspaceService(repo repository.WorkspaceRepository, evaluator *AccessEvaluator) *WorkspaceService {
	return &WorkspaceService{
		repo:      repo,
		evaluator: evaluator,
	}
}

// CreateRoot initializes a new standalone root organization workspace (depth 0).
//
// Business Rules & Guarantees:
//  1. Root Creation is Open (§3 Rule 2): Any authenticated user can create a root.
//  2. The workspace has no parent (parent_id = NULL) and its root_id points to itself.
//  3. Single Atomic Transaction:
//     - Inserts the workspace record into the database.
//     - Populates the closure table with the base self-reference: (workspaceID, workspaceID, depth = 0).
//     - Grants the creator the "owner" tier with the complete OwnerCapabilities preset.
//     - Emits an append-only audit trail entry ("workspace.created").
func (service *WorkspaceService) CreateRoot(ctx context.Context, name string, description *string, creatorUserID uuid.UUID) (*db.Workspace, error) {
	var createdWorkspace db.Workspace
	workspaceID := uuid.New()

	// All 4 operations must succeed together; any failure rolls back the entire tree setup.
	err := service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error

		// Step 1: Insert the root workspace record.
		createdWorkspace, txErr = txRepo.CreateRootWorkspace(ctx, workspaceID, name, description, creatorUserID)
		if txErr != nil {
			return txErr
		}

		// Step 2: Seed the closure table cheat-sheet (depth 0).
		if txErr = txRepo.InsertRootClosure(ctx, workspaceID); txErr != nil {
			return txErr
		}

		// Step 3: Add the creator as the sole active Owner with standard Owner capabilities.
		_, txErr = txRepo.AddWorkspaceMember(ctx, workspaceID, creatorUserID, db.WorkspaceTierOwner, model.OwnerCapabilities, nil)
		if txErr != nil {
			return txErr
		}

		// Step 4: Record an immutable audit log entry for organization creation.
		auditMetadata := map[string]interface{}{"name": name, "description": description, "is_root": true}
		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &creatorUserID, "workspace.created", "workspace", &workspaceID, auditMetadata)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return &createdWorkspace, nil
}

// CreateChild attaches a new subsidiary workspace under an existing parent workspace.
//
// Business Rules & Guarantees (v1 Specification §5 & §6):
//  1. Attachment Authority (§3 Rule 3): Caller must possess the "workspace.create_child" capability on the parent.
//  2. Parent Mutability (§3 Rule 20): The parent must be in the "active" lifecycle state.
//  3. Target Eligibility (§3 Rule 5 & Rule 6):
//     - Ownership is a PROMOTION, never an invite. Target owner must already be an active "admin" of the parent.
//     - Target owner must NOT be the parent's current owner (preventing parent from being orphaned).
//     - Exception (§5): If single-principal mode applies (creator is solo owner promoting self), the check is bypassed.
//  4. Atomic Relocation (§3 Rule 7):
//     - Creates the child workspace node inheriting parent's root_id.
//     - Copies parent ancestry into the closure table (depth + 1) + self row.
//     - Assigns target as child "owner".
//     - ATOMICALLY removes target's admin membership from the parent in the same transaction.
//     - Emits audit log entries on both parent ("member.relocated") and child ("workspace.created").
func (service *WorkspaceService) CreateChild(ctx context.Context, name string, description *string, parentID, targetOwnerUserID, callerUserID uuid.UUID) (*db.Workspace, error) {
	// Guard 1: Verify caller possesses the create_child capability on the parent, and that parent is mutable.
	_, err := service.evaluator.EnforceAccess(ctx, callerUserID, parentID, model.CapWorkspaceCreateChild, true)
	if err != nil {
		return nil, err
	}

	// Guard 2: Fetch parent to obtain its root_id and ensure it is not suspended or archived.
	parentWorkspace, err := service.repo.GetWorkspaceByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parentWorkspace.Status != "active" {
		return nil, wksperr.ErrWorkspaceArchived
	}

	// Guard 3: Single-principal exception vs. standard promotion validation.
	isSinglePrincipal := (targetOwnerUserID == callerUserID)
	if !isSinglePrincipal {
		// Rule 5: Candidate must already hold an active admin seat in the parent workspace.
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

		// Rule 6: Candidate cannot be the current owner of the parent (would orphan parent).
		if targetMember.Tier == db.WorkspaceTierOwner {
			return nil, wksperr.ErrCannotOrphanParent
		}
	}

	workspaceID := uuid.New()
	var createdWorkspace db.Workspace

	// Step 4: Execute the Atomic Relocation Transaction.
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error

		// Step A: Insert child workspace record with parent's root_id.
		createdWorkspace, txErr = txRepo.CreateChildWorkspace(ctx, workspaceID, parentID, parentWorkspace.RootID, name, description, isSinglePrincipal, callerUserID)
		if txErr != nil {
			return txErr
		}

		// Step B: Connect the child to the ancestry tree in workspace_closure.
		if txErr = txRepo.InsertChildClosure(ctx, workspaceID, parentID); txErr != nil {
			return txErr
		}

		// Step C: Seat the candidate as Owner of the child workspace.
		_, txErr = txRepo.AddWorkspaceMember(ctx, workspaceID, targetOwnerUserID, db.WorkspaceTierOwner, model.OwnerCapabilities, &callerUserID)
		if txErr != nil {
			return txErr
		}

		// Step D: The Relocation - Revoke candidate's parent seat so they cannot oversee their own supervisor.
		if !isSinglePrincipal {
			if txErr = txRepo.RemoveMember(ctx, parentID, targetOwnerUserID); txErr != nil {
				return txErr
			}

			// Parent audit log tracking the promotion out of the parent workspace.
			parentAuditMeta := map[string]interface{}{"relocated_to": workspaceID, "promoted_user": targetOwnerUserID}
			_, txErr = txRepo.CreateAuditLog(ctx, parentID, &callerUserID, "member.relocated", "user", &targetOwnerUserID, parentAuditMeta)
			if txErr != nil {
				return txErr
			}
		}

		// Step E: Child audit log documenting subsidiary initialization.
		childAuditMeta := map[string]interface{}{"name": name, "parent_id": parentID, "owner_id": targetOwnerUserID}
		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &callerUserID, "workspace.created", "workspace", &workspaceID, childAuditMeta)
		return txErr
	})

	if err != nil {
		return nil, err
	}
	return &createdWorkspace, nil
}

// GetWorkspace retrieves a workspace's details bundled with the caller's effective access envelope.
//
// Business Rules & Guarantees:
//  1. Visibility & Permissions: Caller must possess the "workspace.view" capability.
//  2. Non-mutating Access: CheckMutability is set to false so users may still view archived or suspended workspaces.
//  3. The response satisfies §11 of the spec, combining metadata and caller capabilities in a single round-trip.
func (service *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID, callerID uuid.UUID) (*model.WorkspaceDetail, error) {
	// Guard: Verify caller has view rights (either as a direct member or via ancestor oversight).
	access, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceView, false)
	if err != nil {
		return nil, err
	}

	// Fetch workspace entity metadata.
	workspace, err := service.repo.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	// Bundle workspace details with caller's effective permissions for UI hydration.
	details := model.WorkspaceDetail{Workspace: workspace, Access: *access}
	return &details, nil
}

// ListWorkspaces lists all workspaces accessible to the caller across the entire SaaS tree.
//
// Visibility Guarantee (§9 & §11):
//   - Returns workspaces where the caller is an active direct member.
//   - Returns workspaces where the caller holds ancestor oversight authority (owner/admin above the child).
//   - Sibling workspaces in branches where caller holds no membership/oversight are omitted entirely (stealth tenanting).
func (service *WorkspaceService) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]model.WorkspaceSummary, error) {
	rows, err := service.repo.ListVisibleWorkspacesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	summaries := make([]model.WorkspaceSummary, 0, len(rows))
	for _, row := range rows {
		var parentID *uuid.UUID
		if row.ParentID.Valid {
			id := uuid.UUID(row.ParentID.Bytes)
			parentID = &id
		}

		var memberTier *string
		if row.MemberTier.Valid {
			tierStr := string(row.MemberTier.WorkspaceTier)
			memberTier = &tierStr
		}

		summaries = append(summaries, model.WorkspaceSummary{
			ID:              row.ID,
			ParentID:        parentID,
			RootID:          row.RootID,
			Name:            row.Name,
			Status:          string(row.Status),
			SinglePrincipal: row.SinglePrincipal,
			MemberTier:      memberTier,
			HasOversight:    bool(row.HasOversight),
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}

	return summaries, nil
}

// UpdateWorkspace modifies non-charter workspace metadata (e.g. name, description).
//
// Business Rules & Guarantees:
//  1. Gated on "workspace.manage_settings" capability.
//  2. Mutability enforced: archived or suspended workspaces cannot be modified.
//  3. Transactional Audit: Updating the workspace and writing the audit entry commit atomically.
func (service *WorkspaceService) UpdateWorkspace(ctx context.Context, workspaceID, callerID uuid.UUID, name string, description *string) (*db.Workspace, error) {
	// Guard: Caller needs manage_settings capability; blocked if archived/suspended.
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceManageSettings, true)
	if err != nil {
		return nil, err
	}

	var updated db.Workspace
	// Execute mutation and audit log atomically.
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error
		updated, txErr = txRepo.UpdateWorkspaceName(ctx, workspaceID, name, description)
		if txErr != nil {
			return txErr
		}

		auditMeta := map[string]interface{}{"name": name, "description": description}
		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &callerID, "workspace.updated", "workspace", &workspaceID, auditMeta)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// SuspendWorkspace immediately freezes a descendant subsidiary workspace.
//
// Business Rules & Guarantees (§3 Rule 12 & §7):
//  1. Oversight Privilege: Must be initiated from an ancestor holding "workspace.oversee_child".
//  2. Immediate & Unilateral: Suspension does not require an approval request.
//  3. Effects: Members in the suspended workspace cannot perform operational write actions.
//  4. Transactional: Status flip and audit logging happen atomically.
func (service *WorkspaceService) SuspendWorkspace(ctx context.Context, workspaceID, callerID uuid.UUID) (*db.Workspace, error) {
	// Guard: Only ancestor oversight can suspend a child; checkMutability is false so active branches can be frozen.
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceOverseeChild, false)
	if err != nil {
		return nil, err
	}

	var suspended db.Workspace
	// Execute suspension and audit log atomically.
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error
		suspended, txErr = txRepo.SuspendWorkspace(ctx, workspaceID, &callerID)
		if txErr != nil {
			return txErr
		}

		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &callerID, "workspace.suspended", "workspace", &workspaceID, nil)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &suspended, nil
}

// ResumeWorkspace restores a suspended subsidiary workspace back to normal operations.
//
// Business Rules & Guarantees (§7):
//  1. Reversible: Head Office oversight may unfreeze any suspended descendant.
//  2. Status returns to "active", clearing suspension metadata.
//  3. Transactional: Status flip and audit logging happen atomically.
func (service *WorkspaceService) ResumeWorkspace(ctx context.Context, workspaceID, callerID uuid.UUID) (*db.Workspace, error) {
	// Guard: Caller needs oversight authority on the suspended branch.
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceOverseeChild, false)
	if err != nil {
		return nil, err
	}

	var resumed db.Workspace
	// Execute resume and audit log atomically.
	err = service.repo.WithTx(ctx, func(txRepo repository.WorkspaceRepository) error {
		var txErr error
		resumed, txErr = txRepo.ResumeWorkspace(ctx, workspaceID)
		if txErr != nil {
			return txErr
		}

		_, txErr = txRepo.CreateAuditLog(ctx, workspaceID, &callerID, "workspace.resumed", "workspace", &workspaceID, nil)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &resumed, nil
}
