package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"server/internal/workspace/model"
	"server/internal/workspace/repository"
	wksperr "server/internal/workspace/wksperr"
)

// AccessEvaluator evaluates effective access and gates operations.
type AccessEvaluator struct {
	repo repository.WorkspaceRepository
}

// NewAccessEvaluator creates a new AccessEvaluator.
func NewAccessEvaluator(repo repository.WorkspaceRepository) *AccessEvaluator {
	return &AccessEvaluator{repo: repo}
}

// EvaluateAccess computes the caller's effective access (tier, via, capabilities) for a workspace.
func (e *AccessEvaluator) EvaluateAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*model.EffectiveAccess, error) {
	row, err := e.repo.GetEffectiveAccess(ctx, userID, workspaceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, wksperr.ErrWorkspaceNotFound
		}
		return nil, err
	}

	// Case 1: Direct Membership takes priority
	if row.DirectTier.Valid {
		tierStr := string(row.DirectTier.WorkspaceTier)
		caps, err := parseCapabilities(row.DirectCapabilities)
		if err != nil {
			return nil, err
		}

		return &model.EffectiveAccess{
			Tier:         &tierStr,
			Via:          "membership",
			Capabilities: caps,
		}, nil
	}

	// Case 2: Ancestor Oversight (no direct membership)
	if row.HasOversight {
		return &model.EffectiveAccess{
			Tier:         nil, // Oversight has no tier
			Via:          "oversight",
			Capabilities: model.OversightCapabilities,
		}, nil
	}

	// Case 3: Neither member nor oversight -> Not visible
	return nil, wksperr.ErrWorkspaceNotFound
}

// EnforceAccess is THE Master Guard Gate.
// It verifies:
// 1. Caller can see the workspace.
// 2. Caller has the specific required capability.
// 3. Workspace is in a mutable state (unless checkMutability is false, e.g. for viewing).
func (e *AccessEvaluator) EnforceAccess(
	ctx context.Context,
	userID, workspaceID uuid.UUID,
	requiredCapability string,
	checkMutability bool,
) (*model.EffectiveAccess, error) {
	// 1. Evaluate Effective Access
	access, err := e.EvaluateAccess(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}

	// 2. Check if the caller has the required capability
	if !hasCapability(access.Capabilities, requiredCapability) {
		return nil, wksperr.ErrForbidden
	}

	// 3. Optional Mutability Check (just like you had in hololine)
	if checkMutability {
		ws, err := e.repo.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return nil, err
		}

		if ws.Status == "suspended" && requiredCapability != model.CapWorkspaceOverseeChild {
			return nil, wksperr.ErrWorkspaceSuspended
		}

		if ws.Status == "archived" || ws.Status == "pending_deletion" || ws.Status == "deleted" {
			return nil, wksperr.ErrWorkspaceArchived
		}
	}

	return access, nil
}

// Helper to unmarshal PostgreSQL JSONB byte slice into []string
func parseCapabilities(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}

	var caps []string
	if err := json.Unmarshal(data, &caps); err != nil {
		return nil, err
	}
	return caps, nil
}

// Helper to check if a slice of capabilities contains the target capability
func hasCapability(capabilities []string, target string) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}
