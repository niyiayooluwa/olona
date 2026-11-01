package handler

import "server/internal/workspace/model"

// Success envelopes for the workspace domain.
// All responses mirror the API response contract wrapping payload data in {"data": ...}.

// WorkspaceEnvelope wraps detailed workspace metadata joined with caller effective access.
type WorkspaceEnvelope struct {
	Data *model.WorkspaceDetail `json:"data"`
}

// WorkspaceRawEnvelope wraps a workspace presentation view.
type WorkspaceRawEnvelope struct {
	Data *model.WorkspaceView `json:"data"`
}

// WorkspaceListEnvelope wraps the array of workspaces visible to the caller (workspace switcher).
type WorkspaceListEnvelope struct {
	Data []model.WorkspaceSummary `json:"data"`
}

// MemberListEnvelope wraps the list of active members in a workspace.
type MemberListEnvelope struct {
	Data []model.MemberSummary `json:"data"`
}

// InvitationData bundles an invitation summary with the raw token for distribution.
type InvitationData struct {
	Invitation *model.InvitationSummary `json:"invitation"`
	Token      string                   `json:"token"`
}

// InvitationEnvelope wraps the created invitation data.
type InvitationEnvelope struct {
	Data InvitationData `json:"data"`
}

// InvitationListEnvelope wraps outstanding pending invitations for a workspace.
type InvitationListEnvelope struct {
	Data []model.InvitationSummary `json:"data"`
}

// RequestEnvelope wraps a single governance request record.
type RequestEnvelope struct {
	Data *model.RequestView `json:"data"`
}

// RequestListEnvelope wraps a slice of workspace governance requests.
type RequestListEnvelope struct {
	Data []model.WorkspaceRequestSummary `json:"data"`
}

// PendingInboxEnvelope wraps pending child requests awaiting overseer approval.
type PendingInboxEnvelope struct {
	Data []model.PendingRequestSummary `json:"data"`
}

// AuditListEnvelope wraps paginated immutable audit logs for a workspace.
type AuditListEnvelope struct {
	Data []model.AuditLogSummary `json:"data"`
}

// MessageResponse holds a standard success message string.
type MessageResponse struct {
	Message string `json:"message"`
}

// MessageEnvelope wraps a simple human-readable confirmation message.
type MessageEnvelope struct {
	Data MessageResponse `json:"data"`
}

// msg builds a standard confirmation response envelope.
func msg(message string) MessageEnvelope {
	return MessageEnvelope{Data: MessageResponse{Message: message}}
}
