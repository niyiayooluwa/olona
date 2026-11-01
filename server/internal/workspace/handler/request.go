package handler

import "github.com/google/uuid"

// Request bodies accepted by the workspace handlers.
// Response shapes live in response.go; error mapping lives in internal/workspace/httperr.

// CreateWorkspaceRequest holds parameters to initialize a root or child workspace.
// If ParentID is omitted, a root organization workspace is created.
// If ParentID is provided, a child subsidiary workspace is attached, and OwnerUserID
// must specify the admin being promoted to owner (unless solo founder single-principal mode).
type CreateWorkspaceRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	OwnerUserID *uuid.UUID `json:"owner_user_id,omitempty"`
}

// UpdateWorkspaceRequest holds mutable metadata for an existing workspace.
type UpdateWorkspaceRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// InviteMemberRequest contains details to issue an email invitation for a workspace.
type InviteMemberRequest struct {
	Email        string   `json:"email"`
	Tier         string   `json:"tier"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// UpdateMemberRoleRequest contains the target tier and capability bundle for a member.
type UpdateMemberRoleRequest struct {
	Tier         string   `json:"tier"`
	Capabilities []string `json:"capabilities"`
}

// AcceptInvitationRequest carries the single-use token received via invitation email.
type AcceptInvitationRequest struct {
	Token string `json:"token"`
}

// CreateGovernanceRequest holds parameters to submit a Two-Key Control request.
// Type must be one of: "archive", "delete", "transfer_ownership", "move".
type CreateGovernanceRequest struct {
	Type         string     `json:"type"`
	Reason       string     `json:"reason"`
	TargetUserID *uuid.UUID `json:"target_user_id,omitempty"`
	NewParentID  *uuid.UUID `json:"new_parent_id,omitempty"`
}

// DecideRequest contains an optional reason when rejecting a governance request.
type DecideRequest struct {
	Reason *string `json:"reason,omitempty"`
}
