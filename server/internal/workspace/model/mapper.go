package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "server/internal/platform/db/generated"
)

// WorkspaceView represents a public JSON representation of a workspace entity.
type WorkspaceView struct {
	ID              uuid.UUID  `json:"id"`
	ParentID        *uuid.UUID `json:"parent_id"`
	RootID          uuid.UUID  `json:"root_id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	Status          string     `json:"status"`
	SinglePrincipal bool       `json:"single_principal"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at"`
	SuspendedAt     *time.Time `json:"suspended_at"`
	SuspendedBy     *uuid.UUID `json:"suspended_by"`
	DeleteAfter     *time.Time `json:"delete_after"`
}

// MemberSummary represents an active workspace member with their public profile info.
type MemberSummary struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	FirstName   *string   `json:"first_name"`
	LastName    *string   `json:"last_name"`
	DisplayName *string   `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	JoinedAt    time.Time `json:"joined_at"`
}

// InvitationSummary represents an invitation formatted for API responses.
type InvitationSummary struct {
	ID           uuid.UUID `json:"id"`
	WorkspaceID  uuid.UUID `json:"workspace_id"`
	Email        string    `json:"email"`
	Tier         string    `json:"tier"`
	Capabilities []string  `json:"capabilities"`
	InvitedBy    uuid.UUID `json:"invited_by"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// PendingRequestSummary represents a pending request in the overseer approval inbox.
type PendingRequestSummary struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	WorkspaceName string     `json:"workspace_name"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	RequestedBy   uuid.UUID  `json:"requested_by"`
	RequesterName *string    `json:"requester_name"`
	Reason        *string    `json:"reason"`
	TargetUserID  *uuid.UUID `json:"target_user_id"`
	NewParentID   *uuid.UUID `json:"new_parent_id"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// WorkspaceRequestSummary represents a request opened inside a workspace.
type WorkspaceRequestSummary struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	RequestedBy   uuid.UUID  `json:"requested_by"`
	RequesterName *string    `json:"requester_name"`
	Reason        *string    `json:"reason"`
	TargetUserID  *uuid.UUID `json:"target_user_id"`
	NewParentID   *uuid.UUID `json:"new_parent_id"`
	ExpiresAt     time.Time  `json:"expires_at"`
	SoleApprover  bool       `json:"sole_approver"`
	CreatedAt     time.Time  `json:"created_at"`
}

// RequestView represents a single governance request detail view.
type RequestView struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	RequestedBy  uuid.UUID  `json:"requested_by"`
	Reason       *string    `json:"reason"`
	TargetUserID *uuid.UUID `json:"target_user_id"`
	NewParentID  *uuid.UUID `json:"new_parent_id"`
	DecidedBy    *uuid.UUID `json:"decided_by"`
	DecidedAt    *time.Time `json:"decided_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	SoleApprover bool       `json:"sole_approver"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AuditLogSummary represents a sanitized, client-safe audit log row.
type AuditLogSummary struct {
	ID          uuid.UUID              `json:"id"`
	WorkspaceID uuid.UUID              `json:"workspace_id"`
	ActorID     *uuid.UUID             `json:"actor_id"`
	ActorEmail  *string                `json:"actor_email"`
	ActorName   *string                `json:"actor_name"`
	Action      string                 `json:"action"`
	TargetType  string                 `json:"target_type"`
	TargetID    *uuid.UUID             `json:"target_id"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TextToPtr converts a nullable pgtype.Text into a *string.
func TextToPtr(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	return &text.String
}

// UUIDToPtr converts a nullable pgtype.UUID into a *uuid.UUID.
func UUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// MapWorkspace converts a db.Workspace record into a clean WorkspaceView.
func MapWorkspace(ws *db.Workspace) *WorkspaceView {
	if ws == nil {
		return nil
	}
	return &WorkspaceView{
		ID:              ws.ID,
		ParentID:        UUIDToPtr(ws.ParentID),
		RootID:          ws.RootID,
		Name:            ws.Name,
		Description:     ws.Description,
		Status:          string(ws.Status),
		SinglePrincipal: ws.SinglePrincipal,
		CreatedBy:       ws.CreatedBy,
		CreatedAt:       ws.CreatedAt,
		UpdatedAt:       ws.UpdatedAt,
		ArchivedAt:      ws.ArchivedAt,
		SuspendedAt:     ws.SuspendedAt,
		SuspendedBy:     UUIDToPtr(ws.SuspendedBy),
		DeleteAfter:     ws.DeleteAfter,
	}
}

// MapMembers converts raw active member database rows into public MemberSummary models.
func MapMembers(rows []db.ListActiveMembersRow) []MemberSummary {
	result := make([]MemberSummary, 0, len(rows))
	for _, r := range rows {
		result = append(result, MemberSummary{
			ID:          r.ID,
			WorkspaceID: r.WorkspaceID,
			UserID:      r.UserID,
			Email:       r.Email,
			FirstName:   TextToPtr(r.FirstName),
			LastName:    TextToPtr(r.LastName),
			DisplayName: TextToPtr(r.DisplayName),
			AvatarURL:   TextToPtr(r.AvatarUrl),
			Tier:        string(r.Tier),
			Status:      string(r.Status),
			JoinedAt:    r.JoinedAt,
		})
	}
	return result
}

// MapInvitation converts a raw invitation record into an InvitationSummary.
func MapInvitation(inv *db.WorkspaceInvitation) *InvitationSummary {
	if inv == nil {
		return nil
	}
	var caps []string
	if len(inv.Capabilities) > 0 {
		_ = json.Unmarshal(inv.Capabilities, &caps)
	}
	return &InvitationSummary{
		ID:           inv.ID,
		WorkspaceID:  inv.WorkspaceID,
		Email:        inv.Email,
		Tier:         string(inv.Tier),
		Capabilities: caps,
		InvitedBy:    inv.InvitedBy,
		ExpiresAt:    inv.ExpiresAt,
		CreatedAt:    inv.CreatedAt,
	}
}

// MapInvitations converts a slice of raw invitations.
func MapInvitations(rows []db.WorkspaceInvitation) []InvitationSummary {
	result := make([]InvitationSummary, 0, len(rows))
	for i := range rows {
		result = append(result, *MapInvitation(&rows[i]))
	}
	return result
}

// MapRequest converts a db.WorkspaceRequest to RequestView.
func MapRequest(req *db.WorkspaceRequest) *RequestView {
	if req == nil {
		return nil
	}
	return &RequestView{
		ID:           req.ID,
		WorkspaceID:  req.WorkspaceID,
		Type:         string(req.Type),
		Status:       string(req.Status),
		RequestedBy:  req.RequestedBy,
		Reason:       req.Reason,
		TargetUserID: UUIDToPtr(req.TargetUserID),
		NewParentID:  UUIDToPtr(req.NewParentID),
		DecidedBy:    UUIDToPtr(req.DecidedBy),
		DecidedAt:    req.DecidedAt,
		ExpiresAt:    req.ExpiresAt,
		SoleApprover: req.SoleApprover,
		CreatedAt:    req.CreatedAt,
	}
}

// MapPendingRequestsForApprover maps rows for the overseer inbox.
func MapPendingRequestsForApprover(rows []db.ListPendingRequestsForApproverRow) []PendingRequestSummary {
	result := make([]PendingRequestSummary, 0, len(rows))
	for _, r := range rows {
		result = append(result, PendingRequestSummary{
			ID:            r.ID,
			WorkspaceID:   r.WorkspaceID,
			WorkspaceName: r.WorkspaceName,
			Type:          string(r.Type),
			Status:        string(r.Status),
			RequestedBy:   r.RequestedBy,
			RequesterName: TextToPtr(r.RequesterName),
			Reason:        r.Reason,
			TargetUserID:  UUIDToPtr(r.TargetUserID),
			NewParentID:   UUIDToPtr(r.NewParentID),
			ExpiresAt:     r.ExpiresAt,
			CreatedAt:     r.CreatedAt,
		})
	}
	return result
}

// MapPendingRequestsForWorkspace maps rows for a specific workspace request list.
func MapPendingRequestsForWorkspace(rows []db.ListPendingRequestsForWorkspaceRow) []WorkspaceRequestSummary {
	result := make([]WorkspaceRequestSummary, 0, len(rows))
	for _, r := range rows {
		result = append(result, WorkspaceRequestSummary{
			ID:            r.ID,
			WorkspaceID:   r.WorkspaceID,
			Type:          string(r.Type),
			Status:        string(r.Status),
			RequestedBy:   r.RequestedBy,
			RequesterName: TextToPtr(r.RequesterName),
			Reason:        r.Reason,
			TargetUserID:  UUIDToPtr(r.TargetUserID),
			NewParentID:   UUIDToPtr(r.NewParentID),
			ExpiresAt:     r.ExpiresAt,
			SoleApprover:  r.SoleApprover,
			CreatedAt:     r.CreatedAt,
		})
	}
	return result
}

// MapAuditLogs converts database audit rows into JSON-safe AuditLogSummary models.
func MapAuditLogs(rows []db.ListAuditLogsForWorkspaceRow) []AuditLogSummary {
	result := make([]AuditLogSummary, 0, len(rows))
	for _, r := range rows {
		var meta map[string]interface{}
		if len(r.Metadata) > 0 {
			_ = json.Unmarshal(r.Metadata, &meta)
		}
		result = append(result, AuditLogSummary{
			ID:          r.ID,
			WorkspaceID: r.WorkspaceID,
			ActorID:     UUIDToPtr(r.ActorID),
			ActorEmail:  TextToPtr(r.ActorEmail),
			ActorName:   TextToPtr(r.ActorName),
			Action:      r.Action,
			TargetType:  r.TargetType,
			TargetID:    UUIDToPtr(r.TargetID),
			Metadata:    meta,
			CreatedAt:   r.CreatedAt,
		})
	}
	return result
}
