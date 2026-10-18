package repository

import (
	"github.com/jackc/pgx/v5/pgtype"

	"server/internal/auth/model"
	db "server/internal/platform/db/generated"
)

// TextToPtr converts an sqlc-generated nullable pgtype.Text column into a Go *string pointer.
func TextToPtr(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	return &text.String
}

// MapToUserProfile converts internal database records (db.User and db.Profile) into the public domain UserProfile representation.
func MapToUserProfile(user db.User, profile db.Profile) *model.UserProfile {
	return &model.UserProfile{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   TextToPtr(profile.FirstName),
		LastName:    TextToPtr(profile.LastName),
		Username:    TextToPtr(profile.Username),
		DisplayName: TextToPtr(profile.DisplayName),
		AvatarURL:   TextToPtr(profile.AvatarUrl),
		IsVerified:  user.EmailVerifiedAt != nil,
	}
}
