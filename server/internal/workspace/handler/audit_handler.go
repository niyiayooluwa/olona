package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/internal/auth/middleware"
	"server/internal/workspace/httperr"
	"server/internal/workspace/model"
	"server/internal/workspace/wksperr"
)

// ListAuditLogs godoc
// @Summary      Get workspace audit logs
// @Description  Returns paginated, immutable audit log events for the workspace. Requires workspace.view_audit capability.
// @Tags         audit
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        limit query int false "Page size (default 50, max 100)"
// @Param        offset query int false "Offset (default 0)"
// @Success      200 {object} AuditListEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/audit [get]
func (h *WorkspaceHandler) ListAuditLogs(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	limit := int32(50)
	if rawLimit := c.QueryParam("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	offset := int32(0)
	if rawOffset := c.QueryParam("offset"); rawOffset != "" {
		if parsed, err := strconv.Atoi(rawOffset); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	callerID := middleware.GetUserID(c)
	logs, err := h.svc.ListAuditLogs(c.Request().Context(), workspaceID, callerID, limit, offset)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, AuditListEnvelope{Data: model.MapAuditLogs(logs)})
}
