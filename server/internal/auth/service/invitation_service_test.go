package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	autherr "server/internal/auth/errors"
	"server/internal/auth/repository"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

func TestInvitationService_Invite(t *testing.T) {
	ctx := context.Background()

	t.Run("success creates invitation and dispatches email", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}
		adminID := uuid.New()

		invitationCreated := false
		emailSent := false

		mockRepo.CreateInvitationFn = func(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error) {
			invitationCreated = true
			if invitedBy != adminID {
				t.Fatalf("expected invitedBy %v, got %v", adminID, invitedBy)
			}
			return db.Invitation{ID: uuid.New(), Email: email}, nil
		}
		mockMailerInstance.SendInviteFn = func(ctx context.Context, email, token string) error {
			emailSent = true
			return nil
		}

		invitationService := NewInvitationService(mockRepo, mockMailerInstance)
		err := invitationService.Invite(ctx, adminID, "newmember@example.com")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !invitationCreated || !emailSent {
			t.Fatalf("expected invitationCreated=%v, emailSent=%v", invitationCreated, emailSent)
		}
	})

	t.Run("mailer error propagates to caller", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		mockMailerInstance := &mockMailer{}

		mockRepo.CreateInvitationFn = func(ctx context.Context, email, tokenHash string, invitedBy uuid.UUID, expiresAt time.Time) (db.Invitation, error) {
			return db.Invitation{ID: uuid.New(), Email: email}, nil
		}
		mockMailerInstance.SendInviteFn = func(ctx context.Context, email, token string) error {
			return errors.New("smtp connection refused")
		}

		invitationService := NewInvitationService(mockRepo, mockMailerInstance)
		err := invitationService.Invite(ctx, uuid.New(), "newmember@example.com")
		if err == nil {
			t.Fatal("expected mailer error, got nil")
		}
	})
}

func TestInvitationService_GetInvitation(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns safe preview details", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		testInviteID := uuid.New()
		inviterID := uuid.New()
		rawToken := "preview-token-123"
		expiresAt := time.Now().Add(48 * time.Hour)

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:         testInviteID,
				Email:      "invited@example.com",
				InvitedBy:  inviterID,
				ExpiresAt:  expiresAt,
				AcceptedAt: nil,
			}, nil
		}

		invitationService := NewInvitationService(mockRepo, nil)
		details, err := invitationService.GetInvitation(ctx, rawToken)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if details == nil {
			t.Fatal("expected details to be non-nil")
		}
		if details.Email != "invited@example.com" || details.InvitedBy != inviterID {
			t.Fatalf("unexpected invitation details: %+v", details)
		}
	})

	t.Run("not found returns ErrInvitationNotFound", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{}, repository.ErrNotFound
		}

		invitationService := NewInvitationService(mockRepo, nil)
		_, err := invitationService.GetInvitation(ctx, "invalid-token")
		if !errors.Is(err, autherr.ErrInvitationNotFound) {
			t.Fatalf("expected ErrInvitationNotFound, got %v", err)
		}
	})

	t.Run("already accepted returns ErrInvitationAlreadyAccepted", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}
		acceptedAt := time.Now().Add(-1 * time.Hour)

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:         uuid.New(),
				AcceptedAt: &acceptedAt,
				ExpiresAt:  time.Now().Add(24 * time.Hour),
			}, nil
		}

		invitationService := NewInvitationService(mockRepo, nil)
		_, err := invitationService.GetInvitation(ctx, "accepted-token")
		if !errors.Is(err, autherr.ErrInvitationAlreadyAccepted) {
			t.Fatalf("expected ErrInvitationAlreadyAccepted, got %v", err)
		}
	})

	t.Run("expired returns ErrInvitationExpired", func(t *testing.T) {
		mockRepo := &mockAuthRepository{}

		mockRepo.GetInvitationByTokenFn = func(ctx context.Context, tokenHash string) (db.Invitation, error) {
			return db.Invitation{
				ID:        uuid.New(),
				ExpiresAt: time.Now().Add(-2 * time.Hour),
			}, nil
		}

		invitationService := NewInvitationService(mockRepo, nil)
		_, err := invitationService.GetInvitation(ctx, utils.HashToken("expired-token"))
		if !errors.Is(err, autherr.ErrInvitationExpired) {
			t.Fatalf("expected ErrInvitationExpired, got %v", err)
		}
	})
}

func TestInvitationService_RevokeInvitation(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockAuthRepository{}
	testInviteID := uuid.New()
	deleted := false

	mockRepo.DeleteInvitationFn = func(ctx context.Context, id uuid.UUID) error {
		if id == testInviteID {
			deleted = true
		}
		return nil
	}

	invitationService := NewInvitationService(mockRepo, nil)
	err := invitationService.RevokeInvitation(ctx, testInviteID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected DeleteInvitation to be called")
	}
}
