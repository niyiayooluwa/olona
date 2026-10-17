package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	db "server/internal/db/generated"
)

func TestProfileService_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns combined user and profile data", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		verifiedAt := time.Now()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:              testUserID,
				Email:           "jane@example.com",
				EmailVerifiedAt: &verifiedAt,
			}, nil
		}
		mockRepo.GetProfileByUserIDFn = func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
			return db.Profile{
				UserID:      testUserID,
				FirstName:   pgtype.Text{String: "Jane", Valid: true},
				LastName:    pgtype.Text{String: "Doe", Valid: true},
				Username:    pgtype.Text{String: "janedoe", Valid: true},
				DisplayName: pgtype.Text{String: "JD", Valid: true},
				AvatarUrl:   pgtype.Text{String: "https://avatar.png", Valid: true},
			}, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		userProfile, err := profileService.GetProfile(ctx, testUserID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if userProfile == nil {
			t.Fatal("expected non-nil user profile")
		}
		if userProfile.Email != "jane@example.com" {
			t.Fatalf("expected email jane@example.com, got %s", userProfile.Email)
		}
		if userProfile.Username == nil || *userProfile.Username != "janedoe" {
			t.Fatalf("expected username janedoe, got %v", userProfile.Username)
		}
		if !userProfile.IsVerified {
			t.Fatal("expected IsVerified to be true")
		}
	})

	t.Run("gracefully handles missing profile record", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{
				ID:    testUserID,
				Email: "noprofile@example.com",
			}, nil
		}
		mockRepo.GetProfileByUserIDFn = func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
			return db.Profile{}, repository.ErrNotFound
		}

		profileService := NewProfileService(mockRepo, cfg)
		userProfile, err := profileService.GetProfile(ctx, testUserID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if userProfile == nil {
			t.Fatal("expected non-nil user profile")
		}
		if userProfile.Username != nil {
			t.Fatalf("expected nil username, got %v", userProfile.Username)
		}
	})

	t.Run("returns error when user is not found", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{}, repository.ErrNotFound
		}

		profileService := NewProfileService(mockRepo, cfg)
		_, err := profileService.GetProfile(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error for missing user, got nil")
		}
	})
}

func TestProfileService_UpdateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("successful profile update", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()

		newFirstName := "Alice"
		newUsername := "alicenew"

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return false, nil
		}
		mockRepo.UpdateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			return db.Profile{
				UserID:    userID,
				FirstName: pgtype.Text{String: *firstName, Valid: true},
				Username:  pgtype.Text{String: *username, Valid: true},
			}, nil
		}
		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{ID: testUserID, Email: "alice@example.com"}, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		userProfile, err := profileService.UpdateProfile(ctx, testUserID, model.ProfileInput{
			FirstName: &newFirstName,
			Username:  &newUsername,
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if userProfile == nil || userProfile.Username == nil || *userProfile.Username != "alicenew" {
			t.Fatalf("expected updated username alicenew, got %v", userProfile)
		}
	})

	t.Run("rejected when username is taken by another user", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		takenUsername := "otheruser"

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return true, nil
		}
		mockRepo.GetProfileByUserIDFn = func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
			return db.Profile{
				UserID:   testUserID,
				Username: pgtype.Text{String: "currentuser", Valid: true},
			}, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		_, err := profileService.UpdateProfile(ctx, testUserID, model.ProfileInput{
			Username: &takenUsername,
		})
		if !errors.Is(err, autherr.ErrUsernameTaken) {
			t.Fatalf("expected ErrUsernameTaken, got %v", err)
		}
	})

	t.Run("succeeds when username is kept identical by the same user", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()
		testUserID := uuid.New()
		sameUsername := "sameusername"

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return true, nil
		}
		mockRepo.GetProfileByUserIDFn = func(ctx context.Context, userID uuid.UUID) (db.Profile, error) {
			return db.Profile{
				UserID:   testUserID,
				Username: pgtype.Text{String: sameUsername, Valid: true},
			}, nil
		}
		mockRepo.UpdateProfileFn = func(ctx context.Context, userID uuid.UUID, firstName, lastName, username, displayName, profileURL, avatarURL *string) (db.Profile, error) {
			return db.Profile{
				UserID:   testUserID,
				Username: pgtype.Text{String: sameUsername, Valid: true},
			}, nil
		}
		mockRepo.GetUserByIDFn = func(ctx context.Context, id uuid.UUID) (db.User, error) {
			return db.User{ID: testUserID, Email: "same@example.com"}, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		userProfile, err := profileService.UpdateProfile(ctx, testUserID, model.ProfileInput{
			Username: &sameUsername,
		})
		if err != nil {
			t.Fatalf("unexpected error for self-retention: %v", err)
		}
		if userProfile == nil {
			t.Fatal("expected non-nil user profile")
		}
	})
}

func TestProfileService_CheckUsernameAvailable(t *testing.T) {
	ctx := context.Background()

	t.Run("returns false when username exists", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return true, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		available, err := profileService.CheckUsernameAvailable(ctx, "taken")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if available {
			t.Fatal("expected available to be false")
		}
	})

	t.Run("returns true when username does not exist", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		cfg := setupTestConfig()

		mockRepo.CheckUsernameExistsFn = func(ctx context.Context, username string) (bool, error) {
			return false, nil
		}

		profileService := NewProfileService(mockRepo, cfg)
		available, err := profileService.CheckUsernameAvailable(ctx, "free")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !available {
			t.Fatal("expected available to be true")
		}
	})
}

func TestProfileService_DeleteProfile(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	cfg := setupTestConfig()
	testUserID := uuid.New()
	deleted := false

	mockRepo.DeleteProfileFn = func(ctx context.Context, userID uuid.UUID) error {
		if userID == testUserID {
			deleted = true
		}
		return nil
	}

	profileService := NewProfileService(mockRepo, cfg)
	err := profileService.DeleteProfile(ctx, testUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected DeleteProfile to be called")
	}
}
