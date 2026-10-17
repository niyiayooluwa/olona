package model

import "github.com/google/uuid"

type Actor struct {
	UserID        uuid.UUID
	Role          Role
	IsSuspended   bool
	IsDeactivated bool
}
