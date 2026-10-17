package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	autherr "server/internal/auth/errors"
	"server/internal/auth/repository"
	db "server/internal/db/generated"
)

// These tests pin the transaction boundary on the OTP *issue* paths —
// ResendOTP and ForgotPassword — as opposed to the OTP *verify* paths covered
// in tx_boundary_test.go.
//
// The bug guarded against is the same class, one step removed. Marking the
// previous code used and inserting its replacement are two separate statements.
// If only the read-decide-mark half is wrapped and the INSERT stays outside, the
// row lock is released before the new code exists, and two concurrent resends
// interleave as:
//
//	A locks row, sees cooldown elapsed, marks old used, commits (lock released)
//	B locks row, finds no valid code, inserts a new code, commits
//	A inserts a new code, commits
//
// Both A and B return success, two live codes exist for one purpose, and
// neither request ever observed the other. Wrapping only the read half looks
// correct — the SELECT is locked, the cooldown is enforced — while leaving the
// race fully intact, which is why it needs an explicit test.

// txAwareRepo carries the call counters shared between the outer repository
// and the one handed to the transaction callback.
//
// The mock's default WithTx passes the *same* mock to fn, so identity alone
// cannot distinguish a transactional call from a non-transactional one — every
// call would look identical whether or not it happened inside a transaction.
// The tests therefore hand fn a *distinct* mock and count on each separately.
//
// A consequence worth knowing when reading these tests: the service performs
// the read-decide-write sequence through the transaction-scoped repository, so
// the lookup hooks must be installed on txRepo, not on the outer repository. A
// hook on the outer mock would be dead code and the inner mock would fall
// through to its "not implemented" default.
type txAwareRepo struct {
	*mockAuthRepository

	// callsOnTxRepo counts writes that arrived through the transaction-scoped
	// repository.
	callsOnTxRepo int
	// callsOnBaseRepo counts writes that arrived through the outer repository,
	// i.e. outside any transaction.
	callsOnBaseRepo int
}

// newTxAwarePair returns the outer repository and the distinct one WithTx will
// hand to its callback. CreateOTP is hooked on both, each counting only itself,
// so a test can assert not just that the insert happened but *where*.
func newTxAwarePair() (*txAwareRepo, *mockAuthRepository) {
	tracker := &txAwareRepo{mockAuthRepository: &mockAuthRepository{}}
	txRepo := &mockAuthRepository{}

	txRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
		tracker.callsOnTxRepo++
		return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
	}
	tracker.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
		tracker.callsOnBaseRepo++
		return db.OtpCode{ID: uuid.New(), UserID: userID}, nil
	}

	tracker.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
		return fn(txRepo)
	}
	return tracker, txRepo
}

func TestResendOTPCreatesTheNewCodeInsideTheTransaction(t *testing.T) {
	ctx := context.Background()
	cfg := setupTestConfig()
	userID := uuid.New()

	tracker, txRepo := newTxAwarePair()
	tracker.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
		return db.User{ID: userID, Email: email}, nil
	}
	// No existing code, so the flow goes straight to issuing one. The lookup is
	// on txRepo because the service reads through the transaction-scoped
	// repository.
	txRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
		return db.OtpCode{}, repository.ErrNotFound
	}

	svc := NewSignupService(tracker, cfg, nil, nil)
	if err := svc.ResendOTP(ctx, "jane@example.com"); err != nil {
		t.Fatalf("ResendOTP() = %v, want nil", err)
	}

	if tracker.callsOnTxRepo != 1 {
		t.Fatalf("CreateOTP ran %d times inside the transaction, want exactly 1", tracker.callsOnTxRepo)
	}
	if tracker.callsOnBaseRepo != 0 {
		t.Fatalf("CreateOTP ran %d times outside the transaction; the row lock is released before the insert, "+
			"so concurrent resends can both mint a live code", tracker.callsOnBaseRepo)
	}
}

func TestForgotPasswordCreatesTheNewCodeInsideTheTransaction(t *testing.T) {
	ctx := context.Background()
	cfg := setupTestConfig()
	userID := uuid.New()
	hash := "a-bcrypt-hash"

	tracker, txRepo := newTxAwarePair()
	tracker.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
		// PasswordHash must be valid, or ForgotPassword returns early for
		// OAuth-only accounts that have no password to reset.
		return db.User{
			ID:           userID,
			Email:        email,
			PasswordHash: pgtype.Text{String: hash, Valid: true},
		}, nil
	}
	txRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
		return db.OtpCode{}, repository.ErrNotFound
	}

	svc := NewPasswordService(tracker, cfg, nil)
	if err := svc.ForgotPassword(ctx, "jane@example.com"); err != nil {
		t.Fatalf("ForgotPassword() = %v, want nil", err)
	}

	if tracker.callsOnTxRepo != 1 {
		t.Fatalf("CreateOTP ran %d times inside the transaction, want exactly 1", tracker.callsOnTxRepo)
	}
	if tracker.callsOnBaseRepo != 0 {
		t.Fatalf("CreateOTP ran %d times outside the transaction; a second live reset code can be minted", tracker.callsOnBaseRepo)
	}
}

// TestResendOTPInvalidatesAndReplacesInOneTransaction covers the full
// replacement path: a previous code exists, its cooldown has elapsed, so the
// flow must both consume it and issue a successor — and it must do the
// consuming inside the transaction, since that is the write the lock protects.
func TestResendOTPInvalidatesAndReplacesInOneTransaction(t *testing.T) {
	ctx := context.Background()
	cfg := setupTestConfig()
	userID := uuid.New()
	oldOTPID := uuid.New()

	tracker, txRepo := newTxAwarePair()
	tracker.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
		return db.User{ID: userID, Email: email}, nil
	}
	// A previous code exists and its cooldown has elapsed, so both the
	// consumption and the replacement must happen.
	txRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
		return db.OtpCode{
			ID:        oldOTPID,
			UserID:    userID,
			CreatedAt: time.Now().Add(-10 * time.Minute), // well past the cooldown
		}, nil
	}

	// Hook the consumption on txRepo, which is only reachable from inside the
	// transaction callback. Recording there proves the previous code is
	// consumed while the row lock is held, rather than after it is released.
	innerMarked := false
	txRepo.MarkOTPUsedFn = func(ctx context.Context, id uuid.UUID) error {
		innerMarked = true
		if id != oldOTPID {
			t.Errorf("MarkOTPUsed(%v), want the previous code %v", id, oldOTPID)
		}
		return nil
	}

	svc := NewSignupService(tracker, cfg, nil, nil)

	if err := svc.ResendOTP(ctx, "jane@example.com"); err != nil {
		t.Fatalf("ResendOTP() = %v, want nil", err)
	}

	if !innerMarked {
		t.Fatal("MarkOTPUsed was not called on the transaction-scoped repository; " +
			"the previous code is being consumed outside the lock")
	}
	if tracker.callsOnTxRepo != 1 {
		t.Fatalf("CreateOTP ran %d times inside the transaction, want 1", tracker.callsOnTxRepo)
	}
}

// TestResendOTPDoesNotIssueWhenTheTransactionFails confirms the abort path. If
// the transaction cannot even begin, no code may be issued and no email sent —
// otherwise the caller is told a code is on its way when none exists.
func TestResendOTPDoesNotIssueWhenTheTransactionFails(t *testing.T) {
	ctx := context.Background()
	cfg := setupTestConfig()
	userID := uuid.New()
	txErr := errors.New("cannot begin transaction")

	mockRepo := &mockAuthRepository{}
	mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
		return db.User{ID: userID, Email: email}, nil
	}
	mockRepo.WithTxHookFn = func(ctx context.Context, fn func(repository.AuthRepository) error) error {
		return txErr
	}

	issued := false
	mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
		issued = true
		return db.OtpCode{}, nil
	}

	svc := NewSignupService(mockRepo, cfg, nil, nil)
	err := svc.ResendOTP(ctx, "jane@example.com")
	if !errors.Is(err, txErr) {
		t.Fatalf("ResendOTP() = %v, want the transaction error", err)
	}
	if issued {
		t.Fatal("a code was issued despite the transaction failing")
	}
}

// TestResendOTPCooldownDoesNotRollBackPriorWork documents why the cooldown
// outcome is captured in a variable instead of returned from fn: the not-found
// branch has to be able to fall through to issuing a code, and returning early
// would abort the transaction before it could.
func TestResendOTPCooldownIsReportedAs429NotAsAFailure(t *testing.T) {
	ctx := context.Background()
	cfg := setupTestConfig()
	cfg.Verification.ResendCooldown = time.Hour
	userID := uuid.New()

	mockRepo := &mockAuthRepository{}
	mockRepo.GetUserByEmailFn = func(ctx context.Context, email string) (db.User, error) {
		return db.User{ID: userID, Email: email}, nil
	}
	mockRepo.GetValidOTPForUpdateFn = func(ctx context.Context, id uuid.UUID, purpose string) (db.OtpCode, error) {
		return db.OtpCode{
			ID:        uuid.New(),
			UserID:    userID,
			CreatedAt: time.Now().Add(-time.Minute), // inside the 1h cooldown
		}, nil
	}
	issued := false
	mockRepo.CreateOTPFn = func(ctx context.Context, userID uuid.UUID, codeHash, purpose string, expiresAt time.Time) (db.OtpCode, error) {
		issued = true
		return db.OtpCode{}, nil
	}

	svc := NewSignupService(mockRepo, cfg, nil, nil)
	err := svc.ResendOTP(ctx, "jane@example.com")
	if !errors.Is(err, autherr.ErrOTPCooldown) {
		t.Fatalf("ResendOTP() = %v, want ErrOTPCooldown", err)
	}
	if issued {
		t.Fatal("a code was issued despite the cooldown being active")
	}
}
