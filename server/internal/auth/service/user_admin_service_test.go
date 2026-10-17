package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestUserAdminService_SuspendUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	testUserID := uuid.New()

	suspended := false
	tokensRevoked := false

	mockRepo.SuspendUserFn = func(ctx context.Context, id uuid.UUID) error {
		if id == testUserID {
			suspended = true
		}
		return nil
	}
	mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
		if userID == testUserID {
			tokensRevoked = true
		}
		return nil
	}

	adminService := NewUserAdminService(mockRepo)
	err := adminService.SuspendUser(ctx, testUserID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !suspended || !tokensRevoked {
		t.Fatalf("expected suspended=%v and tokensRevoked=%v", suspended, tokensRevoked)
	}
}

func TestUserAdminService_UnsuspendUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	testUserID := uuid.New()
	unsuspended := false

	mockRepo.UnsuspendUserFn = func(ctx context.Context, id uuid.UUID) error {
		if id == testUserID {
			unsuspended = true
		}
		return nil
	}

	adminService := NewUserAdminService(mockRepo)
	err := adminService.UnsuspendUser(ctx, testUserID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !unsuspended {
		t.Fatal("expected UnsuspendUser to be called")
	}
}

func TestUserAdminService_DeactivateUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	testUserID := uuid.New()

	deactivated := false
	tokensRevoked := false

	mockRepo.DeactivateUserFn = func(ctx context.Context, id uuid.UUID) error {
		if id == testUserID {
			deactivated = true
		}
		return nil
	}
	mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
		if userID == testUserID {
			tokensRevoked = true
		}
		return nil
	}

	adminService := NewUserAdminService(mockRepo)
	err := adminService.DeactivateUser(ctx, testUserID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deactivated || !tokensRevoked {
		t.Fatalf("expected deactivated=%v and tokensRevoked=%v", deactivated, tokensRevoked)
	}
}

func TestUserAdminService_DeleteUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	testUserID := uuid.New()

	tokensRevoked := false
	profileDeleted := false
	userDeleted := false

	mockRepo.RevokeAllUserRefreshTokensFn = func(ctx context.Context, userID uuid.UUID) error {
		if userID == testUserID {
			tokensRevoked = true
		}
		return nil
	}
	mockRepo.DeleteProfileFn = func(ctx context.Context, userID uuid.UUID) error {
		if userID == testUserID {
			profileDeleted = true
		}
		return nil
	}
	mockRepo.DeleteUserFn = func(ctx context.Context, id uuid.UUID) error {
		if id == testUserID {
			userDeleted = true
		}
		return nil
	}

	adminService := NewUserAdminService(mockRepo)
	err := adminService.DeleteUser(ctx, testUserID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tokensRevoked || !profileDeleted || !userDeleted {
		t.Fatalf("expected tokensRevoked=%v, profileDeleted=%v, userDeleted=%v",
			tokensRevoked, profileDeleted, userDeleted)
	}
}
