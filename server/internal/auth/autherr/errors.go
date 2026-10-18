// Package autherr defines domain sentinel errors for the authentication subsystem.
package autherr

import "errors"

// Credential and account status errors
var (
	// ErrInvalidEmail is returned when an email is structurally malformed.
	ErrInvalidEmail = errors.New("invalid email address format")

	// ErrInvalidCredentials is deliberately identical for "no such email" and "wrong password"
	// to prevent user enumeration attacks.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrAccountSuspended is returned when is_suspended is true.
	ErrAccountSuspended = errors.New("account is suspended")

	// ErrAccountDeactivated is returned when is_deactivated is true.
	ErrAccountDeactivated = errors.New("account has been deactivated")

	// ErrEmailNotVerified is returned when email verification is required but unverified.
	ErrEmailNotVerified = errors.New("email is not verified")

	// ErrOAuthAccount is returned when trying to password-login to an OAuth-only account.
	ErrOAuthAccount = errors.New("this account was created via OAuth, please sign in with your OAuth provider")

	// ErrPasswordLoginDisabled is returned when PasswordConfig.EnablePasswordLogin is false.
	ErrPasswordLoginDisabled = errors.New("password login is disabled")

	// ErrRegistrationDisabled is returned when public registration is not allowed.
	ErrRegistrationDisabled = errors.New("public registration is disabled")

	// ErrPasswordTooShort is returned when password length is below PasswordConfig.MinLength.
	ErrPasswordTooShort = errors.New("password does not meet minimum length requirement")

	// ErrPasswordTooLong is returned when password exceeds bcrypt's 72-byte limit.
	ErrPasswordTooLong = errors.New("password exceeds maximum allowed length of 72 bytes")

	// ErrEmailAlreadyRegistered is returned when registration encounters an already existing email.
	ErrEmailAlreadyRegistered = errors.New("email already registered")

	// ErrUsernameTaken is returned when the chosen username already exists.
	ErrUsernameTaken = errors.New("username is already taken")

	// ErrInvalidUsername is returned when a username violates format/length rules.
	ErrInvalidUsername = errors.New("username must be between 3 and 30 characters, and contain only letters, numbers, and underscores")

	// ErrInvalidProfileName is returned when a profile name field is too long.
	ErrInvalidProfileName = errors.New("profile name fields must not exceed 255 characters")

	// ErrPasswordSame is returned when trying to change password to the same password.
	ErrPasswordSame = errors.New("new password cannot be the same as old password")

	// ErrInvalidAvatarURL is returned when an avatar URL does not use the http
	// or https scheme.
	ErrInvalidAvatarURL = errors.New("avatar URL must be an http or https URL")
)

// Token errors
var (
	// ErrRefreshTokenInvalid is returned when a token is not found or expired.
	ErrRefreshTokenInvalid = errors.New("invalid or expired refresh token")

	// ErrRefreshTokenReused is returned when an already-revoked token is presented again (theft detected!).
	ErrRefreshTokenReused = errors.New("refresh token reuse detected; all sessions revoked")
)

// OTP errors
var (
	// ErrOTPNotFound is returned when no active OTP exists for the user/purpose.
	ErrOTPNotFound = errors.New("no active verification code found")

	// ErrOTPExpired is returned when the code validity window has passed.
	ErrOTPExpired = errors.New("verification code has expired, please request a new one")

	// ErrOTPMaxAttempts is returned when failed attempts exceed VerificationConfig.MaxOTPAttempts.
	ErrOTPMaxAttempts = errors.New("too many invalid attempts, please request a new code")

	// ErrOTPIncorrect is returned when the code doesn't match.
	ErrOTPIncorrect = errors.New("incorrect verification code")

	// ErrOTPCooldown is returned when a code was requested too recently.
	ErrOTPCooldown = errors.New("please wait before requesting another code")
)

// Invitation errors
var (
	// ErrInvitationNotFound is returned when the token doesn't exist.
	ErrInvitationNotFound = errors.New("invitation not found")

	// ErrInvitationExpired is returned when the invite token has passed its expiry.
	ErrInvitationExpired = errors.New("invitation has expired")

	// ErrInvitationAlreadyAccepted is returned if the invite token was already redeemed.
	ErrInvitationAlreadyAccepted = errors.New("invitation has already been accepted")
)

// OAuth errors
var (
	// ErrOAuthProviderNotSupported is returned when the requested provider is not in config.
	ErrOAuthProviderNotSupported = errors.New("oauth provider not supported")

	// ErrOAuthTokenInvalid is returned when provider token validation fails.
	ErrOAuthTokenInvalid = errors.New("invalid oauth token")
)
