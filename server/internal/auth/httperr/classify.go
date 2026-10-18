// Package httperr maps the auth domain's sentinel errors to the HTTP status,
// wire code and human message the API returns for each.
//
// It lives in its own package because two layers need the same answer: the
// handlers, which return a domain error from a service call, and the auth
// middleware, which refuses a request for a suspended or deactivated account.
// Keeping the mapping in one place is what stops the two from drifting apart.
package httperr

import (
	"errors"
	"net/http"

	"server/internal/auth/autherr"
	"server/internal/auth/token"
	"server/internal/platform/apiresponse"

	"github.com/labstack/echo/v4"
)

// Fail writes the envelope error reply for a domain error, letting Classify
// choose the status and code.
func Fail(c echo.Context, err error) error {
	status, code, message := Classify(err)
	return apiresponse.Reject(c, status, code, message)
}

// InvalidBody is the reply for a body that cannot be parsed. It is a helper so
// that every Bind failure produces an identical reply.
func InvalidBody(c echo.Context) error {
	return apiresponse.Reject(c, http.StatusBadRequest, "invalid_body", "Could not read the request.")
}

// Classify maps a domain error to its HTTP status and wire code + message.
//
// The code set is the contract in docs/server/HTTP Response Contract.md.
// Adding a sentinel error means adding a row there in the same commit, and the
// default branch is the information-disclosure guard: an unrecognised error
// must never reach the client as anything but a fixed string.
func Classify(err error) (status int, code, message string) {
	switch {
	// 400 — the client can fix the request.
	case errors.Is(err, autherr.ErrPasswordTooShort):
		return http.StatusBadRequest, "password_too_short", "Password is too short."
	case errors.Is(err, autherr.ErrPasswordTooLong):
		return http.StatusBadRequest, "password_too_long", "Password is too long."
	case errors.Is(err, autherr.ErrPasswordSame):
		return http.StatusBadRequest, "password_same", "New password must differ from the current one."
	case errors.Is(err, autherr.ErrInvalidEmail):
		return http.StatusBadRequest, "invalid_email", "That email address doesn't look right."
	case errors.Is(err, autherr.ErrInvalidUsername):
		return http.StatusBadRequest, "invalid_username", "Username must be 3-30 letters, numbers or underscores."
	case errors.Is(err, autherr.ErrInvalidProfileName):
		return http.StatusBadRequest, "invalid_profile_name", "That name is too long."
	case errors.Is(err, autherr.ErrInvalidAvatarURL):
		return http.StatusBadRequest, "invalid_avatar_url", "Avatar URL must start with http:// or https://."

	// 401 — not authenticated, or deliberately ambiguous.
	case errors.Is(err, autherr.ErrInvalidCredentials):
		return http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect."
	case errors.Is(err, token.ErrInvalidToken):
		return http.StatusUnauthorized, "invalid_token", "Your session has expired. Please sign in again."
	case errors.Is(err, autherr.ErrRefreshTokenInvalid):
		return http.StatusUnauthorized, "refresh_token_invalid", "Your session has expired. Please sign in again."
	case errors.Is(err, autherr.ErrRefreshTokenReused):
		return http.StatusUnauthorized, "refresh_token_reused", "Your session was ended for security reasons. Please sign in again."
	case errors.Is(err, autherr.ErrEmailNotVerified):
		return http.StatusUnauthorized, "email_not_verified", "Please verify your email first."
	case errors.Is(err, autherr.ErrOTPIncorrect):
		return http.StatusUnauthorized, "otp_incorrect", "That code is incorrect."
	case errors.Is(err, autherr.ErrOTPExpired):
		return http.StatusUnauthorized, "otp_expired", "That code has expired. Request a new one."
	case errors.Is(err, autherr.ErrOTPNotFound):
		return http.StatusUnauthorized, "otp_not_found", "No active code found. Request a new one."
	case errors.Is(err, autherr.ErrInvitationNotFound):
		return http.StatusUnauthorized, "invitation_not_found", "That invitation couldn't be found."
	case errors.Is(err, autherr.ErrInvitationExpired):
		return http.StatusUnauthorized, "invitation_expired", "That invitation has expired."
	case errors.Is(err, autherr.ErrInvitationAlreadyAccepted):
		return http.StatusUnauthorized, "invitation_already_accepted", "That invitation has already been used."
	case errors.Is(err, autherr.ErrOAuthAccount):
		return http.StatusUnauthorized, "oauth_account", "This account signs in with a provider. Use that instead."
	case errors.Is(err, autherr.ErrOAuthProviderNotSupported):
		return http.StatusUnauthorized, "oauth_provider_not_supported", "That sign-in provider isn't supported."
	case errors.Is(err, autherr.ErrOAuthTokenInvalid):
		return http.StatusUnauthorized, "oauth_token_invalid", "We couldn't verify that sign-in. Try again."
	case errors.Is(err, autherr.ErrPasswordLoginDisabled):
		return http.StatusUnauthorized, "password_login_disabled", "Password sign-in is turned off for this account."
	case errors.Is(err, autherr.ErrRegistrationDisabled):
		return http.StatusUnauthorized, "registration_disabled", "Public sign-up is currently closed."

	// 403 — authenticated, but not allowed.
	case errors.Is(err, autherr.ErrAccountSuspended):
		return http.StatusForbidden, "account_suspended", "This account is suspended."
	case errors.Is(err, autherr.ErrAccountDeactivated):
		return http.StatusForbidden, "account_deactivated", "This account has been deactivated."

	// 409 — conflicts with existing state.
	case errors.Is(err, autherr.ErrEmailAlreadyRegistered):
		return http.StatusConflict, "email_already_registered", "That email is already registered."
	case errors.Is(err, autherr.ErrUsernameTaken):
		return http.StatusConflict, "username_taken", "That username is already taken."

	// 429 — a limit was hit.
	case errors.Is(err, autherr.ErrOTPMaxAttempts):
		return http.StatusTooManyRequests, "otp_max_attempts", "Too many attempts. Request a new code."
	case errors.Is(err, autherr.ErrOTPCooldown):
		return http.StatusTooManyRequests, "otp_cooldown", "Please wait before requesting another code."

	default:
		return http.StatusInternalServerError, "internal_error", "Something went wrong on our end."
	}
}
