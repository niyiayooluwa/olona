// Package model defines domain entities and data transfer models for authentication.
package model

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Mailer abstracts transactional email delivery.
type Mailer interface {
	SendOTP(ctx context.Context, email, code string) error
	SendInvite(ctx context.Context, email, token string) error
	SendPasswordReset(ctx context.Context, email, code string) error
}

// TokenPair represents an issued access and refresh token pair.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// InvitationDetails represents the visible information about an invite token.
type InvitationDetails struct {
	ID        uuid.UUID
	Email     string
	InvitedBy uuid.UUID
	ExpiresAt time.Time
	Role      Role
}

// ProfileInput carries optional user profile fields for signup or profile updates.
type ProfileInput struct {
	FirstName   *string
	LastName    *string
	Username    *string
	DisplayName *string
	AvatarURL   *string
}

// UserProfile represents the public identity of an authenticated user.
type UserProfile struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	FirstName   *string   `json:"first_name"`
	LastName    *string   `json:"last_name"`
	Username    *string   `json:"username"`
	DisplayName *string   `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	IsVerified  bool      `json:"is_verified"`
}
