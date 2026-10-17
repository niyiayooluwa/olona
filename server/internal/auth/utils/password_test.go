package utils

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestDummyPasswordCompareDoesNotShortCircuit guards requirement AUTH-17.
//
// A malformed dummyHash would make Go's bcrypt return early from newFromHash
// without running the expensive key schedule, so the "user not found" path
// would return measurably faster than a real password check and the mitigation
// would become a user-enumeration oracle. bcrypt itself does not reject the
// whole class of malformed hashes: it validates only length, version and cost,
// never the trailing base64 character.
//
// The check below is deliberately structural rather than purely a stopwatch.
// A timing assertion on a shared CI runner is flaky; a duration ratio with a
// generous bound still catches the real failure mode (a ~250ms key schedule
// collapsing to microseconds) while tolerating scheduler noise.
func TestDummyPasswordCompareDoesNotShortCircuit(t *testing.T) {
	t.Run("dummy hash is a structurally valid bcrypt hash", func(t *testing.T) {
		if !strings.HasPrefix(dummyHash, "$2a$12$") {
			t.Fatalf("dummyHash must be a cost-12 bcrypt hash, got prefix of %q", prefix(dummyHash, 7))
		}
		if len(dummyHash) != 60 {
			t.Fatalf("dummyHash must be 60 characters, got %d", len(dummyHash))
		}
		// Cost must be one bcrypt will actually accept, otherwise
		// newFromHash bails out before doing any work.
		cost, err := bcrypt.Cost([]byte(dummyHash))
		if err != nil {
			t.Fatalf("bcrypt could not parse dummyHash, which would silently disable timing equalization: %v", err)
		}
		if cost != 12 {
			t.Fatalf("dummyHash cost = %d, want 12 to match PasswordConfig.BcryptCost", cost)
		}
	})

	t.Run("dummy comparison costs roughly the same as a real one", func(t *testing.T) {
		real, err := bcrypt.GenerateFromPassword([]byte("a-real-password"), 12)
		if err != nil {
			t.Fatalf("could not generate reference hash: %v", err)
		}

		// Warm up so first-call lazy initialization is not measured.
		DummyPasswordCompare("warmup")
		_ = bcrypt.CompareHashAndPassword(real, []byte("warmup"))

		dummyStart := time.Now()
		DummyPasswordCompare("some-attempted-password")
		dummyElapsed := time.Since(dummyStart)

		realStart := time.Now()
		_ = bcrypt.CompareHashAndPassword(real, []byte("some-attempted-password"))
		realElapsed := time.Since(realStart)

		// The dummy path must do the same expensive work, so it cannot be
		// dramatically cheaper. 10x leaves enormous headroom for CI noise
		// while still failing if the key schedule is skipped entirely.
		if dummyElapsed < realElapsed/10 {
			t.Fatalf("DummyPasswordCompare took %v vs %v for a real comparison; "+
				"it is short-circuiting and would leak whether an email exists",
				dummyElapsed.Round(time.Microsecond), realElapsed.Round(time.Microsecond))
		}
	})

	t.Run("dummy hash never verifies a guessable password", func(t *testing.T) {
		// Guards against someone regenerating the constant with
		// bcrypt.GenerateFromPassword on a guessable string, which would let
		// an attacker satisfy the dummy comparison. The real plaintext is 32
		// CSPRNG bytes, so none of these can match.
		for _, candidate := range []string{
			"",
			"password",
			"password123",
			"timing-equalizer-not-a-credential",
			"dummyHash",
		} {
			if bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(candidate)) == nil {
				t.Fatalf("dummyHash must not verify the guessable password %q", candidate)
			}
		}
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("round-trips a valid password", func(t *testing.T) {
		hash, err := HashPassword("correct horse battery staple", 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !CheckPassword(hash, "correct horse battery staple") {
			t.Fatal("expected the generated hash to verify its own password")
		}
		if CheckPassword(hash, "wrong password") {
			t.Fatal("expected a wrong password not to verify")
		}
	})

	t.Run("rejects an out-of-range cost instead of silently downgrading", func(t *testing.T) {
		// Falling back to bcrypt.DefaultCost (10) here would quietly weaken a
		// deployment configured for 12, with no signal that anything changed.
		for _, cost := range []int{bcrypt.MinCost - 1, bcrypt.MaxCost + 1, 0, -1} {
			_, err := HashPassword("password", cost)
			var invalidCost *InvalidCostError
			if !errors.As(err, &invalidCost) {
				t.Fatalf("HashPassword(cost=%d) error = %v, want *InvalidCostError", cost, err)
			}
			if invalidCost.Cost != cost {
				t.Fatalf("InvalidCostError.Cost = %d, want %d", invalidCost.Cost, cost)
			}
		}
	})

	t.Run("accepts the minimum cost", func(t *testing.T) {
		// Only MinCost is exercised. bcrypt.MaxCost is 31, so actually
		// hashing at it would run 2^31 rounds and never finish; the upper
		// bound is covered by the out-of-range test above instead.
		if _, err := HashPassword("password", bcrypt.MinCost); err != nil {
			t.Fatalf("HashPassword(cost=%d) unexpected error: %v", bcrypt.MinCost, err)
		}
	})
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("swordfish", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !CheckPassword(hash, "swordfish") {
		t.Fatal("expected the correct password to verify")
	}
	if CheckPassword(hash, "Swordfish") {
		t.Fatal("expected verification to be case-sensitive")
	}
	if CheckPassword("not-a-hash", "swordfish") {
		t.Fatal("expected a malformed hash not to verify")
	}
}

// prefix returns at most n leading bytes of s, for error messages.
func prefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
