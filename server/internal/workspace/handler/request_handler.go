package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/internal/auth/middleware"
	db "server/internal/platform/db/generated"
	"server/internal/workspace/httperr"
	"server/internal/workspace/model"
	"server/internal/workspace/wksperr"
)

// CreateRequest godoc
// @Summary      Open a governance request
// @Description  Submits a Two-Key Control request (archive, delete, transfer_ownership, move). Requires matching request_* capability.
// @Tags         requests
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        request body CreateGovernanceRequest true "Request parameters"
// @Success      201 {object} RequestEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body or target user"
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      409 {object} apiresponse.ErrorEnvelope "active children block action"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/requests [post]
func (h *WorkspaceHandler) CreateRequest(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	var req CreateGovernanceRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	callerID := middleware.GetUserID(c)
	requestRow, err := h.svc.CreateRequest(c.Request().Context(), workspaceID, callerID, db.WorkspaceRequestType(req.Type), &req.Reason, req.TargetUserID, req.NewParentID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusCreated, RequestEnvelope{Data: model.MapRequest(requestRow)})
}

// ListWorkspaceRequests godoc
// @Summary      List requests for a workspace
// @Description  Returns all governance requests created for a workspace. Requires workspace.view capability.
// @Tags         requests
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} RequestListEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/requests [get]
func (h *WorkspaceHandler) ListWorkspaceRequests(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	requests, err := h.svc.ListWorkspaceRequests(c.Request().Context(), workspaceID, callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, RequestListEnvelope{Data: model.MapPendingRequestsForWorkspace(requests)})
}

// ListPendingRequests godoc
// @Summary      Head Office approval inbox
// @Description  Returns pending governance requests across all child workspaces that caller has oversight authority to approve.
// @Tags         requests
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} PendingInboxEnvelope
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/requests [get]
func (h *WorkspaceHandler) ListPendingRequests(c echo.Context) error {
	callerID := middleware.GetUserID(c)
	inbox, err := h.svc.ListPendingRequests(c.Request().Context(), callerID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, PendingInboxEnvelope{Data: model.MapPendingRequestsForApprover(inbox)})
}

// ApproveRequest godoc
// @Summary      Approve a governance request
// @Description  Approves a pending request and atomically executes the state change. Requester cannot approve their own request.
// @Tags         requests
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Request UUID"
// @Success      200 {object} MessageEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "requester cannot approve or lack oversight authority"
// @Failure      404 {object} apiresponse.ErrorEnvelope "request not found or already decided"
// @Failure      409 {object} apiresponse.ErrorEnvelope "active children conflict"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/requests/{id}/approve [post]
func (h *WorkspaceHandler) ApproveRequest(c echo.Context) error {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrRequestNotFound)
	}

	callerID := middleware.GetUserID(c)
	if err := h.svc.ApproveRequest(c.Request().Context(), requestID, callerID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Request approved successfully."))
}

// RejectRequest godoc
// @Summary      Reject a governance request
// @Description  Rejects a pending request with an optional explanation. Requester cannot reject their own request.
// @Tags         requests
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Request UUID"
// @Param        request body DecideRequest false "Rejection reason"
// @Success      200 {object} MessageEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "request not found or already decided"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/requests/{id}/reject [post]
func (h *WorkspaceHandler) RejectRequest(c echo.Context) error {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrRequestNotFound)
	}

	var req DecideRequest
	// Rejection reason is optional; ignoring EOF/empty body is safe
	_ = c.Bind(&req)

	callerID := middleware.GetUserID(c)
	if err := h.svc.RejectRequest(c.Request().Context(), requestID, callerID, req.Reason); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Request rejected successfully."))
}

// CancelRequest godoc
// @Summary      Cancel a governance request
// @Description  Cancels an open request. Only the user who originally created the request may cancel it.
// @Tags         requests
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Request UUID"
// @Success      200 {object} MessageEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "only original requester can cancel"
// @Failure      404 {object} apiresponse.ErrorEnvelope "request not found or already decided"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/requests/{id}/cancel [post]
func (h *WorkspaceHandler) CancelRequest(c echo.Context) error {
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrRequestNotFound)
	}

	callerID := middleware.GetUserID(c)
	if err := h.svc.CancelRequest(c.Request().Context(), requestID, callerID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Request cancelled successfully."))
}
