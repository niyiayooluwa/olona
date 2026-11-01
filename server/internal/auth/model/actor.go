package model

import "github.com/google/uuid"

type Actor struct {
	UserID        uuid.UUID
	Email         string
	Role          Role
	IsSuspended   bool
	IsDeactivated bool
}
