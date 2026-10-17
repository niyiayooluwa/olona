package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "server/internal/db/generated"
)

// fakeTx is a test double that satisfies pgx.Tx. Only the lifecycle methods the
// WithTx implementation calls are meaningful; the rest panic if reached, since
// nothing here runs SQL against a real server.
type fakeTx struct {
	pgx.Tx

	beginCalled   bool
	commitCalls   int
	rollbackCalls int
	queryRowCalls int

	commitErr   error
	rollbackErr error
}

func (t *fakeTx) Commit(ctx context.Context) error {
	t.beginCalled = true
	t.commitCalls++
	return t.commitErr
}

func (t *fakeTx) Rollback(ctx context.Context) error {
	t.rollbackCalls++
	return t.rollbackErr
}

func (t *fakeTx) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *fakeTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("fakeTx: Query not supported")
}

func (t *fakeTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	t.queryRowCalls++
	return errRow{}
}

// errRow makes any statement fail, so tests only ever assert on routing.
type errRow struct{}

func (errRow) Scan(dest ...interface{}) error { return errors.New("fakeTx: Scan not supported") }

var _ db.DBTX = (*fakeTx)(nil)

// fakeBeginner hands out a fixed tx so tests can observe the transaction
// lifecycle without a database.
type fakeBeginner struct {
	tx       *fakeTx
	beginErr error
	calls    int
}

func (b *fakeBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	b.calls++
	if b.beginErr != nil {
		return nil, b.beginErr
	}
	return b.tx, nil
}

func TestWithTxRequiresATransactionSource(t *testing.T) {
	// Without WithTxSource the repository is still usable for single
	// statements, but any caller reaching for a transaction must get a clear
	// error rather than a silent run outside one.
	repo := NewAuthRepository(&fakeQuerier{})

	err := repo.WithTx(context.Background(), func(AuthRepository) error {
		t.Fatal("fn must not run when transactions are unconfigured")
		return nil
	})

	if !errors.Is(err, ErrTransactionsUnsupported) {
		t.Fatalf("WithTx() = %v, want ErrTransactionsUnsupported", err)
	}
}

func TestWithTxCommitsOnSuccess(t *testing.T) {
	tx := &fakeTx{}
	beginner := &fakeBeginner{tx: tx}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	ran := false
	err := repo.WithTx(context.Background(), func(AuthRepository) error {
		ran = true
		return nil
	})

	if err != nil {
		t.Fatalf("WithTx() = %v, want nil", err)
	}
	if !ran {
		t.Error("fn was not called")
	}
	if tx.commitCalls != 1 {
		t.Errorf("Commit called %d times, want 1", tx.commitCalls)
	}
	if tx.rollbackCalls != 0 {
		t.Errorf("Rollback called %d times, want 0", tx.rollbackCalls)
	}
}

func TestWithTxRollsBackWhenFnFails(t *testing.T) {
	tx := &fakeTx{}
	beginner := &fakeBeginner{tx: tx}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	sentinel := errors.New("business rule rejected the change")

	err := repo.WithTx(context.Background(), func(AuthRepository) error {
		return sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Fatalf("WithTx() = %v, want it to propagate %v", err, sentinel)
	}
	if tx.rollbackCalls != 1 {
		t.Errorf("Rollback called %d times, want 1", tx.rollbackCalls)
	}
	if tx.commitCalls != 0 {
		t.Errorf("Commit called %d times, want 0", tx.commitCalls)
	}
}

func TestWithTxPassesATransactionBackedRepository(t *testing.T) {
	// The repo handed to fn must be backed by the tx, not the original
	// querier. If it were the original, statements would autocommit and the
	// row lock that motivates this whole mechanism would still be released
	// early — the exact bug WithTx exists to prevent.
	tx := &fakeTx{}
	beginner := &fakeBeginner{tx: tx}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	var got AuthRepository
	err := repo.WithTx(context.Background(), func(txRepo AuthRepository) error {
		got = txRepo
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx() = %v, want nil", err)
	}

	impl, ok := got.(*authRepositoryImpl)
	if !ok {
		t.Fatalf("fn received %T, want *authRepositoryImpl", got)
	}
	// db.Queries keeps its DBTX unexported, so prove the wiring behaviourally:
	// issue a statement through the repo fn was given and check the tx saw it.
	_, stmtErr := got.CheckUsernameExists(context.Background(), "probe")
	if stmtErr == nil {
		t.Error("expected the probe statement to fail against fakeTx")
	}
	if tx.queryRowCalls != 1 {
		t.Errorf("tx saw %d statements, want 1 — the repo fn was given is not backed by the transaction", tx.queryRowCalls)
	}
	if impl.beginner != nil {
		t.Error("fn's repo must not expose a transaction source, so nested WithTx cannot silently start a second transaction")
	}
}

func TestWithTxSurfacesBeginFailure(t *testing.T) {
	beginErr := errors.New("connection pool exhausted")
	beginner := &fakeBeginner{beginErr: beginErr}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	err := repo.WithTx(context.Background(), func(AuthRepository) error {
		t.Fatal("fn must not run when the transaction cannot be started")
		return nil
	})

	if !errors.Is(err, beginErr) {
		t.Fatalf("WithTx() = %v, want it to wrap %v", err, beginErr)
	}
}

func TestWithTxRollsBackOnPanic(t *testing.T) {
	// A handler that panics must not leave the OTP row locked for the rest of
	// the session's lifetime, so the deferred rollback has to run on the way out.
	tx := &fakeTx{}
	beginner := &fakeBeginner{tx: tx}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected the panic to propagate to the caller")
			}
		}()
		_ = repo.WithTx(context.Background(), func(AuthRepository) error {
			panic("handler exploded mid-flow")
		})
	}()

	if tx.rollbackCalls != 1 {
		t.Errorf("Rollback called %d times, want 1", tx.rollbackCalls)
	}
	if tx.commitCalls != 0 {
		t.Errorf("Commit called %d times, want 0", tx.commitCalls)
	}
}

func TestWithTxDoesNotCommitTwice(t *testing.T) {
	// pgx.ErrTxClosed after a successful commit is expected and ignored. A
	// second Commit would be a real bug, so assert the call count.
	tx := &fakeTx{rollbackErr: pgx.ErrTxClosed}
	beginner := &fakeBeginner{tx: tx}
	repo := NewAuthRepository(&fakeQuerier{}, WithTxSource(beginner))

	if err := repo.WithTx(context.Background(), func(AuthRepository) error { return nil }); err != nil {
		t.Fatalf("WithTx() = %v, want nil", err)
	}

	if tx.commitCalls != 1 {
		t.Errorf("Commit called %d times, want 1", tx.commitCalls)
	}
}
