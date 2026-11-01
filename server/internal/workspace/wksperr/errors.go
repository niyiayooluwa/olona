// Package wksperr defines sentinel domain errors for Olona workspace governance.
package wksperr

import "errors"

// Sentinel errors for the workspace domain.
var (
	ErrWorkspaceNotFound       = errors.New("workspace not found")
	ErrForbidden               = errors.New("you do not have permission to perform this action")
	ErrWorkspaceSuspended      = errors.New("workspace is suspended")
	ErrWorkspaceArchived       = errors.New("workspace is archived")
	ErrActiveChildren          = errors.New("workspace has active children and cannot be archived or deleted")
	ErrInvalidRelocation       = errors.New("target owner must be an active admin of the parent workspace")
	ErrCannotOrphanParent      = errors.New("parent owner cannot be promoted to child owner")
	ErrRequesterIsApprover     = errors.New("requester cannot approve their own request")
	ErrInvitationNotFound      = errors.New("invitation not found or expired")
	ErrInvitationEmailMismatch = errors.New("authenticated user email does not match invitation recipient")
	ErrCannotLeaveAsOwner      = errors.New("the sole active owner cannot leave without transferring ownership")
	ErrRequestNotFound         = errors.New("request not found or already decided")
	ErrInvalidRequestTarget    = errors.New("target user must be an active admin of the workspace")
)
