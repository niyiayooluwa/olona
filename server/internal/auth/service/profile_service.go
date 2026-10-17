package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service/helpers"
)

// ProfileService manages user profiles, identity details, and username availability.
type ProfileService struct {
	repo   repository.AuthRepository
	config config.AuthConfig
}

// NewProfileService constructs a new ProfileService.
func NewProfileService(
	repo repository.AuthRepository,
	cfg config.AuthConfig,
) *ProfileService {
	return &ProfileService{
		repo:   repo,
		config: cfg,
	}
}

// GetProfile retrieves the authenticated user's combined account and profile information.
//
// Behavior:
// - Fetches user credentials and verification status.
// - Fetches associated profile attributes (name, username, avatar).
// - Gracefully tolerates missing profile records (returns non-nil UserProfile with empty profile fields).
//
// Returns:
// - (*model.UserProfile, nil): Combined profile data.
// - (nil, error): User record not found or database read failure.
func (service *ProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	// 1. Fetch user account record
	user, err := service.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch associated profile record (gracefully tolerating ErrNotFound)
	profile, err := service.repo.GetProfileByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// 3. Map database models to domain UserProfile
	return helpers.MapToUserProfile(user, profile), nil
}

// UpdateProfile updates the profile attributes of an existing user.
//
// Validation & Uniqueness:
// - If ProfileConfig.EnableUsername is true and a new username is provided:
//   - Checks whether the proposed username is already taken by another user.
//   - If taken, allows the update only if it already belongs to the current user.
//   - If claimed by another user, returns ErrUsernameTaken.
//
// - Persists non-nil profile updates in the database.
//
// Returns:
// - (*model.UserProfile, nil): Updated profile data.
// - (nil, ErrUsernameTaken): Proposed username is claimed by another account.
// - (nil, error): Database failure.
func (service *ProfileService) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	input model.ProfileInput,
) (*model.UserProfile, error) {
	// 1. Validate username format and uniqueness if username feature is active
	if service.config.Profile.EnableUsername && input.Username != nil && *input.Username != "" {
		if err := helpers.ValidateUsername(*input.Username); err != nil {
			return nil, err
		}
		exists, err := service.repo.CheckUsernameExists(ctx, *input.Username)
		if err != nil {
			return nil, err
		}
		if exists {
			// Check if the username already belongs to the requesting user
			currentProfile, err := service.repo.GetProfileByUserID(ctx, userID)
			if err != nil || !currentProfile.Username.Valid || currentProfile.Username.String != *input.Username {
				return nil, autherr.ErrUsernameTaken
			}
		}
	}

	// 1a. Validate profile names
	if service.config.Profile.EnableName {
		if input.FirstName != nil {
			if err := helpers.ValidateProfileName(*input.FirstName); err != nil {
				return nil, err
			}
		}
		if input.LastName != nil {
			if err := helpers.ValidateProfileName(*input.LastName); err != nil {
				return nil, err
			}
		}
	}
	if service.config.Profile.EnableDisplayName {
		if input.DisplayName != nil {
			if err := helpers.ValidateProfileName(*input.DisplayName); err != nil {
				return nil, err
			}
		}
	}

	// 1b. Reject an avatar URL that is not plain http(s), before it reaches the
	// database. The value is stored verbatim and rendered by clients, so an
	// unchecked scheme (javascript:, data:) is a stored-injection surface.
	if err := helpers.ValidateAvatarURL(input.AvatarURL); err != nil {
		return nil, err
	}

	// 2. Persist profile changes in database
	profile, err := service.repo.UpdateProfile(
		ctx,
		userID,
		input.FirstName,
		input.LastName,
		input.Username,
		input.DisplayName,
		nil,
		input.AvatarURL,
	)
	if err != nil {
		return nil, err
	}

	// 3. Fetch user record to construct combined response
	user, err := service.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 4. Map updated database models to domain UserProfile
	return helpers.MapToUserProfile(user, profile), nil
}

// CheckUsernameAvailable queries the persistence layer to determine if a username is available.
// Returns true if the username is free to be registered, or false if already taken.
func (service *ProfileService) CheckUsernameAvailable(ctx context.Context, username string) (bool, error) {
	if err := helpers.ValidateUsername(username); err != nil {
		return false, err
	}
	// 1. Query repository for username existence
	exists, err := service.repo.CheckUsernameExists(ctx, username)
	if err != nil {
		return false, err
	}

	// 2. Return inverted existence (true = available)
	return !exists, nil
}

// DeleteProfile deletes a user's associated profile record while retaining their authentication credentials.
func (service *ProfileService) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	return service.repo.DeleteProfile(ctx, userID)
}
