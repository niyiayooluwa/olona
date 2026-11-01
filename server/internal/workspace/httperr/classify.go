// Package httperr maps the workspace domain's sentinel errors to the HTTP status,
// wire code, and human-readable message returned to API clients.
//
// Mirroring internal/auth/httperr, keeping error classification in a dedicated package
// ensures handlers and any future middleware produce consistent envelope error responses.
package httperr

import (
	"errors"
	"net/http"

	"server/internal/platform/apiresponse"
	"server/internal/workspace/repository"
	"server/internal/workspace/wksperr"

	"github.com/labstack/echo/v4"
)

// Fail writes the standardized envelope error response for a workspace domain error.
func Fail(c echo.Context, err error) error {
	status, code, message := Classify(err)
	return apiresponse.Reject(c, status, code, message)
}

// InvalidBody is a helper for returning an identical 400 response whenever c.Bind fails.
func InvalidBody(c echo.Context) error {
	return apiresponse.Reject(c, http.StatusBadRequest, "invalid_body", "Could not read the request.")
}

// Classify maps domain sentinel errors to (status, wireCode, humanMessage).
// Unrecognized errors default to an internal server error to prevent information disclosure.
func Classify(err error) (status int, code, message string) {
	switch {
	// 400 — Malformed or invalid parameters
	case errors.Is(err, wksperr.ErrInvalidRelocation):
		return http.StatusBadRequest, "invalid_relocation", "Target owner must be an active admin of the parent workspace."
	case errors.Is(err, wksperr.ErrInvalidRequestTarget):
		return http.StatusBadRequest, "invalid_request_target", "Target user must be an active admin of the workspace."

	// 403 — Forbidden due to lack of capabilities, lifecycle state, or governance rules
	case errors.Is(err, wksperr.ErrForbidden):
		return http.StatusForbidden, "forbidden", "You do not have permission to perform this action."
	case errors.Is(err, wksperr.ErrWorkspaceSuspended):
		return http.StatusForbidden, "workspace_suspended", "This workspace is suspended by parent oversight."
	case errors.Is(err, wksperr.ErrWorkspaceArchived):
		return http.StatusForbidden, "workspace_archived", "This workspace is archived and read-only."
	case errors.Is(err, wksperr.ErrRequesterIsApprover):
		return http.StatusForbidden, "requester_cannot_approve", "Requester cannot approve their own governance request."
	case errors.Is(err, wksperr.ErrInvitationEmailMismatch):
		return http.StatusForbidden, "invitation_email_mismatch", "Your authenticated email does not match the invitation recipient."

	// 404 — Not found
	case errors.Is(err, wksperr.ErrWorkspaceNotFound), errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound, "workspace_not_found", "Workspace not found."
	case errors.Is(err, wksperr.ErrInvitationNotFound):
		return http.StatusNotFound, "invitation_not_found", "Invitation not found or has expired."
	case errors.Is(err, wksperr.ErrRequestNotFound):
		return http.StatusNotFound, "request_not_found", "Governance request not found or has already been decided."

	// 409 — Conflict with tree, lifecycle, or sole-owner rules
	case errors.Is(err, wksperr.ErrActiveChildren):
		return http.StatusConflict, "active_children_block_action", "Workspace has active child workspaces and cannot be archived or deleted."
	case errors.Is(err, wksperr.ErrCannotOrphanParent):
		return http.StatusConflict, "cannot_orphan_parent", "The parent workspace owner cannot be relocated."
	case errors.Is(err, wksperr.ErrCannotLeaveAsOwner):
		return http.StatusConflict, "cannot_leave_as_owner", "The sole workspace owner cannot leave without transferring ownership."

	// 500 — Unhandled error
	default:
		return http.StatusInternalServerError, "internal_error", "An unexpected error occurred."
	}
}
