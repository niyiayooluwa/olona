package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/auth/autherr"
	"server/internal/auth/httperr"
	"server/internal/auth/token"
	"server/internal/platform/apiresponse"

	"github.com/labstack/echo/v4"
)

// newTestContext builds an echo context wired to a recorder, so a test can call
// fail directly and assert on the status and body it produced.
func newTestContext(t *testing.T) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	rec := httptest.NewRecorder()
	return e.NewContext(httptest.NewRequest(http.MethodPost, "/auth/verify", nil), rec), rec
}

// TestFailMapsEverySentinel locks in the status AND wire code for each domain
// error.
//
// The point of this table is coverage, not each individual row. Before it
// existed, the old handleError switch covered 8 of 23 sentinel errors and
// silently answered 500 for the rest — including ErrOTPMaxAttempts, the OTP
// brute-force lockout, and ErrRefreshTokenReused, the token-theft detector.
// Those are security events, and reporting them as server faults makes an
// attack in progress indistinguishable from an outage in the 5xx metrics.
//
// Any new sentinel error added to the autherr package must get a row here. The
// final case asserts nothing falls through to 500 by accident. The code column
// is the contract in docs/server/HTTP Response Contract.md §5 — the client
// branches on it, so a drift here is a client bug.
func TestFailMapsEverySentinel(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		want     int
		wantCode string
	}{
		// 409 — conflicts with existing state. AUTH-21 requires 409 on a
		// unique-email collision, so this is specified behavior.
		{"email already registered", autherr.ErrEmailAlreadyRegistered, http.StatusConflict, "email_already_registered"},
		{"username taken", autherr.ErrUsernameTaken, http.StatusConflict, "username_taken"},

		// 429 — a limit was hit. These must be visible as throttling, not as
		// server faults.
		{"otp max attempts", autherr.ErrOTPMaxAttempts, http.StatusTooManyRequests, "otp_max_attempts"},
		{"otp cooldown", autherr.ErrOTPCooldown, http.StatusTooManyRequests, "otp_cooldown"},

		// 403 — valid credential, account may not act.
		{"account suspended", autherr.ErrAccountSuspended, http.StatusForbidden, "account_suspended"},
		{"account deactivated", autherr.ErrAccountDeactivated, http.StatusForbidden, "account_deactivated"},

		// 401 — the credential is not valid. ErrEmailNotVerified and
		// ErrOAuthAccount describe a real account, so a 403 here would confirm
		// it exists to anyone who can supply an email.
		{"invalid credentials", autherr.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
		{"otp incorrect", autherr.ErrOTPIncorrect, http.StatusUnauthorized, "otp_incorrect"},
		{"otp expired", autherr.ErrOTPExpired, http.StatusUnauthorized, "otp_expired"},
		{"otp not found", autherr.ErrOTPNotFound, http.StatusUnauthorized, "otp_not_found"},
		{"email not verified", autherr.ErrEmailNotVerified, http.StatusUnauthorized, "email_not_verified"},
		{"oauth account", autherr.ErrOAuthAccount, http.StatusUnauthorized, "oauth_account"},
		{"refresh token invalid", autherr.ErrRefreshTokenInvalid, http.StatusUnauthorized, "refresh_token_invalid"},
		{"refresh token reused", autherr.ErrRefreshTokenReused, http.StatusUnauthorized, "refresh_token_reused"},
		{"invitation not found", autherr.ErrInvitationNotFound, http.StatusUnauthorized, "invitation_not_found"},
		{"invitation expired", autherr.ErrInvitationExpired, http.StatusUnauthorized, "invitation_expired"},
		{"invitation already accepted", autherr.ErrInvitationAlreadyAccepted, http.StatusUnauthorized, "invitation_already_accepted"},
		{"oauth provider unsupported", autherr.ErrOAuthProviderNotSupported, http.StatusUnauthorized, "oauth_provider_not_supported"},
		{"oauth token invalid", autherr.ErrOAuthTokenInvalid, http.StatusUnauthorized, "oauth_token_invalid"},
		{"password login disabled", autherr.ErrPasswordLoginDisabled, http.StatusUnauthorized, "password_login_disabled"},
		{"registration disabled", autherr.ErrRegistrationDisabled, http.StatusUnauthorized, "registration_disabled"},
		{"jwt invalid", token.ErrInvalidToken, http.StatusUnauthorized, "invalid_token"},

		// 400 — the request is wrong and the client can correct it.
		{"password too short", autherr.ErrPasswordTooShort, http.StatusBadRequest, "password_too_short"},
		{"password too long", autherr.ErrPasswordTooLong, http.StatusBadRequest, "password_too_long"},
		{"password same", autherr.ErrPasswordSame, http.StatusBadRequest, "password_same"},
		{"invalid email", autherr.ErrInvalidEmail, http.StatusBadRequest, "invalid_email"},
		{"invalid username", autherr.ErrInvalidUsername, http.StatusBadRequest, "invalid_username"},
		{"invalid profile name", autherr.ErrInvalidProfileName, http.StatusBadRequest, "invalid_profile_name"},
		{"invalid avatar url", autherr.ErrInvalidAvatarURL, http.StatusBadRequest, "invalid_avatar_url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(t)
			if err := httperr.Fail(c, tt.err); err != nil {
				t.Fatalf("fail() = %v, want nil", err)
			}
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body.String())
			}

			var got apiresponse.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not the error envelope: %v (body: %s)", err, rec.Body.String())
			}
			if got.Error.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if got.Error.Message == "" {
				t.Fatal("message must not be empty; the client shows it to a human")
			}
			// An auth failure must never carry field-level details: they would
			// confirm things like whether an email is registered.
			if got.Error.Details != nil {
				t.Fatalf("details must be absent on an auth failure, got %+v", got.Error.Details)
			}
		})
	}
}

// TestFailMatchesWrappedSentinels confirms the mapping survives error
// wrapping. Services return errors from deeper layers, so a sentinel that
// arrives wrapped must still resolve — errors.Is, not ==, is what makes that
// work, and this is the test that would catch someone "simplifying" it.
func TestFailMatchesWrappedSentinels(t *testing.T) {
	c, rec := newTestContext(t)
	wrapped := fmt.Errorf("verifying otp: %w", autherr.ErrOTPMaxAttempts)

	if err := httperr.Fail(c, wrapped); err != nil {
		t.Fatalf("fail() = %v, want nil", err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d for a wrapped sentinel, want 429", rec.Code)
	}
}

// TestFailDoesNotLeakInternalDetail is the information-disclosure guard. An
// unrecognized error must produce a fixed string; if the default branch ever
// returned err.Error(), a database error containing a connection string or a
// query fragment would reach the client.
func TestFailDoesNotLeakInternalDetail(t *testing.T) {
	c, rec := newTestContext(t)
	internal := errors.New("pq: FATAL password authentication failed for user \"postgres\" (SQLSTATE 28P01) at /var/lib/postgresql/data/pg_hba.conf:42")

	if err := httperr.Fail(c, internal); err != nil {
		t.Fatalf("fail() = %v, want nil", err)
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
