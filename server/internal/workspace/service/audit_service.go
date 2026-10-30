package service

import (
	"context"

	"github.com/google/uuid"

	db "server/internal/platform/db/generated"
	"server/internal/workspace/model"
)

// ListAuditLogs retrieves the immutable audit trail for a workspace with pagination.
//
// Gated on "workspace.view_audit" capability.
func (service *WorkspaceService) ListAuditLogs(
	ctx context.Context,
	workspaceID, callerID uuid.UUID,
	limit, offset int32,
) ([]db.ListAuditLogsForWorkspaceRow, error) {
	// Guard: Caller needs view_audit capability (either as direct member or ancestor oversight).
	_, err := service.evaluator.EnforceAccess(ctx, callerID, workspaceID, model.CapWorkspaceViewAudit, false)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return service.repo.ListAuditLogsForWorkspace(ctx, workspaceID, limit, offset)
}
