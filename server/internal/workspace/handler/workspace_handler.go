// Package handler implements the HTTP presentation layer for workspaces and governance.
package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/internal/auth/middleware"
	_ "server/internal/platform/apiresponse"
	"server/internal/workspace/httperr"
	"server/internal/workspace/model"
	"server/internal/workspace/service"
	"server/internal/workspace/wksperr"
)

// WorkspaceHandler exposes HTTP routes for workspace lifecycle, governance, and audit.
type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

// NewWorkspaceHandler constructs the workspace domain HTTP handler.
func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

// CreateWorkspace godoc
// @Summary      Create root or child workspace
// @Description  Creates a root organization workspace (if parent_id is omitted) or a child subsidiary workspace.
// @Description  When creating a child, the target owner is atomically relocated from the parent.
// @Tags         workspaces
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body CreateWorkspaceRequest true "Workspace creation parameters"
// @Success      201 {object} WorkspaceRawEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body or target owner not admin"
// @Failure      403 {object} apiresponse.ErrorEnvelope "caller lacks create_child capability"
// @Failure      404 {object} apiresponse.ErrorEnvelope "parent workspace not found"
// @Failure      409 {object} apiresponse.ErrorEnvelope "parent owner cannot be relocated"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces [post]
func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	var req CreateWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}
	if req.Name == "" {
		return httperr.Fail(c, wksperr.ErrForbidden)
	}

	callerID := middleware.GetUserID(c)
	ctx := c.Request().Context()

	// Case 1: Root Organization Creation (depth 0, no parent).
	if req.ParentID == nil {
		ws, err := h.svc.CreateRoot(ctx, req.Name, req.Description, callerID)
		if err != nil {
			return httperr.Fail(c, err)
		}
		return c.JSON(http.StatusCreated, WorkspaceRawEnvelope{Data: model.MapWorkspace(ws)})
	}

	// Case 2: Child Subsidiary Workspace Creation.
	targetOwner := callerID
	if req.OwnerUserID != nil {
		targetOwner = *req.OwnerUserID
	}

	ws, err := h.svc.CreateChild(ctx, req.Name, req.Description, *req.ParentID, targetOwner, callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusCreated, WorkspaceRawEnvelope{Data: model.MapWorkspace(ws)})
}

// ListWorkspaces godoc
// @Summary      List visible workspaces
// @Description  Returns the workspace switcher list containing all workspaces where the caller has membership or oversight.
// @Tags         workspaces
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} WorkspaceListEnvelope
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces [get]
func (h *WorkspaceHandler) ListWorkspaces(c echo.Context) error {
	callerID := middleware.GetUserID(c)
	workspaces, err := h.svc.ListWorkspaces(c.Request().Context(), callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, WorkspaceListEnvelope{Data: workspaces})
}

// GetWorkspace godoc
// @Summary      Get workspace details
// @Description  Retrieves workspace metadata along with caller's effective access (tier, origin, and capabilities).
// @Tags         workspaces
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} WorkspaceEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "no access to workspace"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id} [get]
func (h *WorkspaceHandler) GetWorkspace(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	detail, err := h.svc.GetWorkspace(c.Request().Context(), workspaceID, callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, WorkspaceEnvelope{Data: detail})
}

// UpdateWorkspace godoc
// @Summary      Update workspace settings
// @Description  Updates workspace name and/or description. Requires workspace.manage_settings capability.
// @Tags         workspaces
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        request body UpdateWorkspaceRequest true "Update settings"
// @Success      200 {object} WorkspaceRawEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied or workspace suspended"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id} [patch]
func (h *WorkspaceHandler) UpdateWorkspace(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	var req UpdateWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	callerID := middleware.GetUserID(c)
	ws, err := h.svc.UpdateWorkspace(c.Request().Context(), workspaceID, callerID, req.Name, req.Description)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, WorkspaceRawEnvelope{Data: model.MapWorkspace(ws)})
}

// SuspendWorkspace godoc
// @Summary      Suspend a child workspace
// @Description  Immediately freezes a child workspace via parent oversight authority. Reversible via resume.
// @Tags         workspaces
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} WorkspaceRawEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "caller lacks oversight authority"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/suspend [post]
func (h *WorkspaceHandler) SuspendWorkspace(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	ws, err := h.svc.SuspendWorkspace(c.Request().Context(), workspaceID, callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, WorkspaceRawEnvelope{Data: model.MapWorkspace(ws)})
}

// ResumeWorkspace godoc
// @Summary      Resume a suspended workspace
// @Description  Unfreezes a previously suspended child workspace via parent oversight authority.
// @Tags         workspaces
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} WorkspaceRawEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "caller lacks oversight authority"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/resume [post]
func (h *WorkspaceHandler) ResumeWorkspace(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	ws, err := h.svc.ResumeWorkspace(c.Request().Context(), workspaceID, callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, WorkspaceRawEnvelope{Data: model.MapWorkspace(ws)})
}
