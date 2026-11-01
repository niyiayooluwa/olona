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

// ListMembers godoc
// @Summary      List active workspace members
// @Description  Returns active members of the workspace with user profile details. Requires workspace.view capability.
// @Tags         members
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} MemberListEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/members [get]
func (h *WorkspaceHandler) ListMembers(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	members, err := h.svc.ListMembers(c.Request().Context(), callerID, workspaceID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, MemberListEnvelope{Data: model.MapMembers(members)})
}

// InviteMember godoc
// @Summary      Invite user to workspace
// @Description  Sends an invitation for a user to join with an admin, member, or viewer tier. Owner tier cannot be invited.
// @Tags         members
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        request body InviteMemberRequest true "Invitation parameters"
// @Success      201 {object} InvitationEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      403 {object} apiresponse.ErrorEnvelope "cannot invite owner or lacking manage_members capability"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/invitations [post]
func (h *WorkspaceHandler) InviteMember(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	var req InviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	callerID := middleware.GetUserID(c)
	inv, rawToken, err := h.svc.InviteMember(c.Request().Context(), callerID, workspaceID, req.Email, db.WorkspaceTier(req.Tier), req.Capabilities)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusCreated, InvitationEnvelope{Data: InvitationData{Invitation: model.MapInvitation(inv), Token: rawToken}})
}

// ListPendingInvitations godoc
// @Summary      List outstanding invitations
// @Description  Returns active, unaccepted, unrevoked invitations for the workspace. Requires workspace.manage_members.
// @Tags         members
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Success      200 {object} InvitationListEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "workspace not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/invitations [get]
func (h *WorkspaceHandler) ListPendingInvitations(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}

	callerID := middleware.GetUserID(c)
	invitations, err := h.svc.ListPendingInvitations(c.Request().Context(), callerID, workspaceID)
	if err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, InvitationListEnvelope{Data: model.MapInvitations(invitations)})
}

// RevokeInvitation godoc
// @Summary      Revoke pending invitation
// @Description  Revokes an active invitation token so it can no longer be used to join.
// @Tags         members
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        invitationId path string true "Invitation UUID"
// @Success      200 {object} MessageEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "invitation not found"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/invitations/{invitationId} [delete]
func (h *WorkspaceHandler) RevokeInvitation(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}
	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrInvitationNotFound)
	}

	callerID := middleware.GetUserID(c)
	if err := h.svc.RevokeInvitation(c.Request().Context(), callerID, workspaceID, invitationID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Invitation revoked successfully."))
}

// AcceptInvitation godoc
// @Summary      Accept workspace invitation
// @Description  Consumes an invitation token and adds caller to workspace. Caller email must match recipient email.
// @Tags         members
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body AcceptInvitationRequest true "Invitation token"
// @Success      200 {object} MessageEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      403 {object} apiresponse.ErrorEnvelope "email does not match invitation recipient"
// @Failure      404 {object} apiresponse.ErrorEnvelope "invitation not found or expired"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/invitations/accept [post]
func (h *WorkspaceHandler) AcceptInvitation(c echo.Context) error {
	var req AcceptInvitationRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	actor := middleware.GetActor(c)
	if err := h.svc.AcceptInvitation(c.Request().Context(), req.Token, actor.Email, actor.UserID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Joined workspace successfully."))
}

// UpdateMemberRole godoc
// @Summary      Update member role
// @Description  Changes a member's tier and capability set. Cannot demote sole owner or promote directly to owner.
// @Tags         members
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        userId path string true "Target User UUID"
// @Param        request body UpdateMemberRoleRequest true "New role configuration"
// @Success      200 {object} MessageEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "member not found"
// @Failure      409 {object} apiresponse.ErrorEnvelope "cannot demote sole owner"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/members/{userId} [patch]
func (h *WorkspaceHandler) UpdateMemberRole(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrInvalidRequestTarget)
	}

	var req UpdateMemberRoleRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	callerID := middleware.GetUserID(c)
	if _, err := h.svc.UpdateMemberRole(c.Request().Context(), callerID, workspaceID, targetUserID, db.WorkspaceTier(req.Tier), req.Capabilities); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Member role updated successfully."))
}

// RemoveMember godoc
// @Summary      Remove a member or leave workspace
// @Description  Removes an active member from the workspace, or allows caller to voluntarily leave. Sole owner cannot leave.
// @Tags         members
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Workspace UUID"
// @Param        userId path string true "Target User UUID"
// @Success      200 {object} MessageEnvelope
// @Failure      403 {object} apiresponse.ErrorEnvelope "permission denied"
// @Failure      404 {object} apiresponse.ErrorEnvelope "member not found"
// @Failure      409 {object} apiresponse.ErrorEnvelope "sole owner cannot leave without transfer"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /v1/workspaces/{id}/members/{userId} [delete]
func (h *WorkspaceHandler) RemoveMember(c echo.Context) error {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrWorkspaceNotFound)
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return httperr.Fail(c, wksperr.ErrInvalidRequestTarget)
	}

	callerID := middleware.GetUserID(c)
	if err := h.svc.RemoveMember(c.Request().Context(), callerID, workspaceID, targetUserID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.JSON(http.StatusOK, msg("Member removed successfully."))
}
