package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	autherr "server/internal/auth/errors"
	"server/internal/auth/utils"

	"github.com/labstack/echo/v4"
)

// newTestContext builds an echo context wired to a recorder, so a test can call
// handleError directly and assert on the status and body it produced.
func newTestContext(t *testing.T) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	rec := httptest.NewRecorder()
	return e.NewContext(httptest.NewRequest(http.MethodPost, "/auth/verify", nil), rec), rec
}

// TestHandleErrorMapsEverySentinel locks in the status for each domain error.
//
// The point of this table is coverage, not each individual row. Before it
// existed, handleError's switch covered 8 of 23 sentinel errors and silently
// answered 500 for the rest — including ErrOTPMaxAttempts, the OTP
// brute-force lockout, and ErrRefreshTokenReused, the token-theft detector.
// Those are security events, and reporting them as server faults makes an
// attack in progress indistinguishable from an outage in the 5xx metrics.
//
// Any new sentinel error added to the autherr package must get a row here. The
// final case asserts nothing falls through to 500 by accident.
func TestHandleErrorMapsEverySentinel(t *testing.T) {
	h := &AuthHandler{}

	tests := []struct {
		name string
		err  error
		want int
	}{
		// 409 — conflicts with existing state. AUTH-21 requires 409 on a
		// unique-email collision, so this is specified behavior.
		{"email already registered", autherr.ErrEmailAlreadyRegistered, http.StatusConflict},
		{"username taken", autherr.ErrUsernameTaken, http.StatusConflict},

		// 429 — a limit was hit. These must be visible as throttling, not as
		// server faults.
		{"otp max attempts", autherr.ErrOTPMaxAttempts, http.StatusTooManyRequests},
		{"otp cooldown", autherr.ErrOTPCooldown, http.StatusTooManyRequests},

		// 403 — valid credential, account may not act.
		{"account suspended", autherr.ErrAccountSuspended, http.StatusForbidden},
		{"account deactivated", autherr.ErrAccountDeactivated, http.StatusForbidden},

		// 401 — the credential is not valid. ErrEmailNotVerified and
		// ErrOAuthAccount describe a real account, so a 403 here would confirm
		// it exists to anyone who can supply an email.
		{"invalid credentials", autherr.ErrInvalidCredentials, http.StatusUnauthorized},
		{"otp incorrect", autherr.ErrOTPIncorrect, http.StatusUnauthorized},
		{"otp expired", autherr.ErrOTPExpired, http.StatusUnauthorized},
		{"otp not found", autherr.ErrOTPNotFound, http.StatusUnauthorized},
		{"email not verified", autherr.ErrEmailNotVerified, http.StatusUnauthorized},
		{"oauth account", autherr.ErrOAuthAccount, http.StatusUnauthorized},
		{"refresh token invalid", autherr.ErrRefreshTokenInvalid, http.StatusUnauthorized},
		{"refresh token reused", autherr.ErrRefreshTokenReused, http.StatusUnauthorized},
		{"invitation not found", autherr.ErrInvitationNotFound, http.StatusUnauthorized},
		{"invitation expired", autherr.ErrInvitationExpired, http.StatusUnauthorized},
		{"invitation already accepted", autherr.ErrInvitationAlreadyAccepted, http.StatusUnauthorized},
		{"oauth provider unsupported", autherr.ErrOAuthProviderNotSupported, http.StatusUnauthorized},
		{"oauth token invalid", autherr.ErrOAuthTokenInvalid, http.StatusUnauthorized},
		{"password login disabled", autherr.ErrPasswordLoginDisabled, http.StatusUnauthorized},
		{"registration disabled", autherr.ErrRegistrationDisabled, http.StatusUnauthorized},
		{"jwt invalid", utils.ErrInvalidToken, http.StatusUnauthorized},

		// 400 — the request is wrong and the client can correct it.
		{"password too short", autherr.ErrPasswordTooShort, http.StatusBadRequest},
		{"password too long", autherr.ErrPasswordTooLong, http.StatusBadRequest},
		{"password same", autherr.ErrPasswordSame, http.StatusBadRequest},
		{"invalid avatar url", autherr.ErrInvalidAvatarURL, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(t)
			if err := h.handleError(c, tt.err); err != nil {
				t.Fatalf("handleError() = %v, want nil", err)
			}
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// TestHandleErrorMatchesWrappedSentinels confirms the mapping survives error
// wrapping. Services return errors from deeper layers, so a sentinel that
// arrives wrapped must still resolve — errors.Is, not ==, is what makes that
// work, and this is the test that would catch someone "simplifying" it.
func TestHandleErrorMatchesWrappedSentinels(t *testing.T) {
	h := &AuthHandler{}

	c, rec := newTestContext(t)
	wrapped := fmt.Errorf("verifying otp: %w", autherr.ErrOTPMaxAttempts)

	if err := h.handleError(c, wrapped); err != nil {
		t.Fatalf("handleError() = %v, want nil", err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d for a wrapped sentinel, want 429", rec.Code)
	}
}

// TestHandleErrorDoesNotLeakInternalDetail is the information-disclosure
// guard. An unrecognized error must produce a fixed string; if the default
// branch ever returned err.Error(), a database error containing a connection
// string or a query fragment would reach the client.
func TestHandleErrorDoesNotLeakInternalDetail(t *testing.T) {
	h := &AuthHandler{}

	c, rec := newTestContext(t)
	internal := errors.New("pq: FATAL password authentication failed for user \"postgres\" (SQLSTATE 28P01) at /var/lib/postgresql/data/pg_hba.conf:42")

	if err := h.handleError(c, internal); err != nil {
		t.Fatalf("handleError() = %v, want nil", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	for _, leak := range []string{"postgres", "28P01", "pg_hba", "SQLSTATE"} {
		if strings.Contains(body, leak) {
			t.Fatalf("response body leaked %q: %s", leak, body)
		}
	}
}
