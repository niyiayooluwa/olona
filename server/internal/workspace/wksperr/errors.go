package wksperr

import "errors"

// Sentinel errors for the workspace domain.
var (
	ErrWorkspaceNotFound   = errors.New("workspace not found")
	ErrForbidden           = errors.New("you do not have permission to perform this action")
	ErrWorkspaceSuspended  = errors.New("workspace is suspended")
	ErrWorkspaceArchived   = errors.New("workspace is archived")
	ErrActiveChildren      = errors.New("workspace has active children and cannot be archived or deleted")
	ErrInvalidRelocation   = errors.New("target owner must be an active admin of the parent workspace")
	ErrCannotOrphanParent  = errors.New("parent owner cannot be promoted to child owner")
	ErrRequesterIsApprover = errors.New("requester cannot approve their own request")
)
