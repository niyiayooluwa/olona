// Package model defines domain entities, access evaluations, capability presets,
// and presentation view models for the workspace domain.
package model

import (
	db "server/internal/platform/db/generated"
	"time"

	"github.com/google/uuid"
)

type WorkspaceDetail struct {
	Workspace db.Workspace    `json:"workspace"`
	Access    EffectiveAccess `json:"access"`
}

type WorkspaceSummary struct {
	ID              uuid.UUID  `json:"id"`
	ParentID        *uuid.UUID `json:"parent_id"`
	RootID          uuid.UUID  `json:"root_id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	SinglePrincipal bool       `json:"single_principal"`
	MemberTier      *string    `json:"member_tier"`
	HasOversight    bool       `json:"has_oversight"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
