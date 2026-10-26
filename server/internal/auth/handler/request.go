package handler

// Request bodies accepted by the auth handlers. Response shapes live in
// response.go; the errors handlers produce are mapped in internal/auth/httperr.

// SignupRequest holds the necessary fields to register a new user.
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// VerifyRequest contains the email and the 6-digit OTP code to verify an account.
type VerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// ResendOTPRequest is used to request a new verification code.
type ResendOTPRequest struct {
	Email string `json:"email"`
}

// LoginRequest requires email and password for authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// AUTH 24 STUB
	//DeviceID string `json:"device_id"`
}

// RefreshRequest is used to exchange a valid refresh token for a new token pair.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	// AUTH 24 STUB
	// DeviceID     string `json:"device_id"`
}

// LogoutRequest requires the refresh token to revoke it upon logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ForgotPasswordRequest takes an email to send a reset link/code to.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest requires the email, reset code and a new password.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordRequest is used by authenticated users to update their password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// UpdateProfileRequest holds optional fields to update user profile information.
type UpdateProfileRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}
