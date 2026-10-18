// Package handler implements the HTTP presentation layer for authentication.
package handler

import (
	"errors"
	"net/http"

	"server/internal/auth/autherr"
	"server/internal/auth/httperr"
	"server/internal/auth/middleware"
	"server/internal/auth/model"
	"server/internal/auth/service"
	"server/internal/auth/validate"
	"server/internal/platform/apiresponse"

	"github.com/labstack/echo/v4"
)

// AuthHandler exposes HTTP routes for all authentication domain services.
type AuthHandler struct {
	svc *service.Services
}

// NewAuthHandler constructs the HTTP handler for the auth module.
func NewAuthHandler(svc *service.Services) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Signup godoc
// @Summary      Create a new account
// @Description  Registers a new user and sends an OTP for email verification.
// @Description  A duplicate email is NOT reported as an error. The handler
// @Description  returns 202 with an identical body to a successful 201 so that
// @Description  the endpoint cannot be used to discover which addresses have
// @Description  accounts. Clients must treat 202 as "check your inbox" exactly
// @Description  as they treat 201 — no error branch exists for that case.
// @Description  A taken username is reported honestly as 409: it says nothing
// @Description  about any email address, and no account is created when it is
// @Description  returned, so masking it would tell the caller a code was sent
// @Description  for an account that does not exist.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body SignupRequest true "Signup details"
// @Success      201 {object} SignupEnvelope
// @Failure      202 {object} SignupEnvelope "duplicate email, masked as success"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "registration disabled"
// @Failure      409 {object} apiresponse.ErrorEnvelope "username already taken"
// @Failure      429 {object} apiresponse.ErrorEnvelope "too many requests"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /auth/signup [post]
func (h *AuthHandler) Signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := validate.ValidateEmail(req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	profileInput := model.ProfileInput{}
	if req.Username != "" {
		profileInput.Username = &req.Username
	}

	tokens, err := h.svc.Signup.Signup(c.Request().Context(), req.Email, req.Password, profileInput)
	if err != nil {
		// Email only. ErrUsernameTaken used to be masked here too, and that
		// was a lost signup rather than a protection: the username check is a
		// SELECT that runs before the transaction opens, so a collision aborts
		// with no user, no profile and no OTP on record — yet the caller was
		// told a verification code was on its way. Nothing was enumerable from
		// a taken username, so the mask bought no security and cost the user
		// the only account they were trying to create. Fall through to fail,
		// which reports it as 409.
		if errors.Is(err, autherr.ErrEmailAlreadyRegistered) {
			return c.JSON(http.StatusAccepted, SignupEnvelope{Data: SignupData{VerificationRequired: true}})
		}
		return httperr.Fail(c, err)
	}

	if tokens != nil {
		// Open registration where verification is not required.
		return c.JSON(http.StatusCreated, SignupEnvelope{Data: SignupData{
			VerificationRequired: false,
			Tokens:               &TokenPairResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken},
		}})
	}

	return c.JSON(http.StatusCreated, SignupEnvelope{Data: SignupData{VerificationRequired: true}})
}

// Verify godoc
// @Summary      Confirm signup OTP and activate the account
// @Description  Verifies the 6-digit code sent to the user's email. Returns a token pair upon success.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body VerifyRequest true "Email and OTP code"
// @Success      200 {object} TokenPairEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "invalid or expired OTP"
// @Failure      429 {object} apiresponse.ErrorEnvelope "too many attempts"
// @Failure      500 {object} apiresponse.ErrorEnvelope "internal error"
// @Router       /auth/verify [post]
func (h *AuthHandler) Verify(c echo.Context) error {
	var req VerifyRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := validate.ValidateEmail(req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	tokens, err := h.svc.Signup.VerifyEmail(c.Request().Context(), req.Email, req.Code)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, tokenPair(tokens.AccessToken, tokens.RefreshToken))
}

// ResendOTP godoc
// @Summary      Resend verification OTP
// @Description  Sends a new verification OTP to the user's email.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResendOTPRequest true "Email to resend OTP"
// @Success      204 "No Content"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      429 {object} apiresponse.ErrorEnvelope "cooldown active"
// @Router       /auth/resend-otp [post]
func (h *AuthHandler) ResendOTP(c echo.Context) error {
	var req ResendOTPRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := validate.ValidateEmail(req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	if err := h.svc.Signup.ResendOTP(c.Request().Context(), req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// Login godoc
// @Summary      Login to account
// @Description  Authenticates a user and returns a token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} TokenPairEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := validate.ValidateEmail(req.Email); err != nil {
		// Do not return ErrInvalidEmail directly to avoid enumeration, use InvalidCredentials
		return httperr.Fail(c, autherr.ErrInvalidCredentials)
	}

	tokens, err := h.svc.Session.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, tokenPair(tokens.AccessToken, tokens.RefreshToken))
}

// Refresh godoc
// @Summary      Refresh token pair
// @Description  Exchanges a valid refresh token for a new token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200 {object} TokenPairEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "invalid or expired token"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	tokens, err := h.svc.Session.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, tokenPair(tokens.AccessToken, tokens.RefreshToken))
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the provided refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LogoutRequest true "Refresh token to revoke"
// @Success      204 "No Content"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var req LogoutRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := h.svc.Session.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		return httperr.Fail(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Sends a password reset OTP to the email if it exists. Always
// @Description  answers 204, whether or not the address has an account, so the
// @Description  endpoint cannot be used to enumerate registered emails.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ForgotPasswordRequest true "Email address"
// @Success      204 "No Content"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	// Do not leak whether the email is invalid vs not found: same 204 either way.
	if err := validate.ValidateEmail(req.Email); err != nil {
		return c.NoContent(http.StatusNoContent)
	}

	if err := h.svc.Password.ForgotPassword(c.Request().Context(), req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Resets user password using the OTP received via email.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "Reset details"
// @Success      204 "No Content"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "invalid or expired OTP"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	if err := validate.ValidateEmail(req.Email); err != nil {
		return httperr.Fail(c, err)
	}

	if err := h.svc.Password.ResetPassword(c.Request().Context(), req.Email, req.Code, req.NewPassword); err != nil {
		return httperr.Fail(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Allows an authenticated user to update their password.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body ChangePasswordRequest true "Password details"
// @Security     BearerAuth
// @Success      204 "No Content"
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Router       /v1/users/me/password [post]
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	userID := middleware.GetUserID(c)
	if err := h.svc.Password.ChangePassword(c.Request().Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		return httperr.Fail(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Retrieves the profile of the authenticated user.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} ProfileEnvelope
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Router       /v1/users/me [get]
func (h *AuthHandler) GetProfile(c echo.Context) error {
	userID := middleware.GetUserID(c)
	profile, err := h.svc.Profile.GetProfile(c.Request().Context(), userID)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, ProfileEnvelope{Data: profile})
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates optional fields of the authenticated user's profile.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body UpdateProfileRequest true "Profile details"
// @Security     BearerAuth
// @Success      200 {object} ProfileEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Failure      409 {object} apiresponse.ErrorEnvelope "username already claimed"
// @Router       /v1/users/me [patch]
func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	userID := middleware.GetUserID(c)

	// Convert DTO to model
	input := model.ProfileInput{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	}

	profile, err := h.svc.Profile.UpdateProfile(c.Request().Context(), userID, input)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, ProfileEnvelope{Data: profile})
}

// CheckUsername godoc
// @Summary      Check username availability
// @Description  Checks if a username is available for registration.
// @Tags         auth
// @Produce      json
// @Param        username query string true "Username to check"
// @Success      200 {object} UsernameCheckEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "missing parameter"
// @Router       /auth/check-username [get]
func (h *AuthHandler) CheckUsername(c echo.Context) error {
	username := c.QueryParam("username")
	if username == "" {
		return apiresponse.Reject(c, http.StatusBadRequest, "missing_parameter", "A username is required.")
	}

	available, err := h.svc.Profile.CheckUsernameAvailable(c.Request().Context(), username)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, UsernameCheckEnvelope{Data: UsernameCheck{Available: available}})
}

// DeleteProfile godoc
// @Summary      Delete profile
// @Description  Allows a user to permanently delete their own account.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      204 "No Content"
// @Failure      401 {object} apiresponse.ErrorEnvelope "unauthorized"
// @Router       /v1/users/me [delete]
func (h *AuthHandler) DeleteProfile(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.svc.Profile.DeleteProfile(c.Request().Context(), userID); err != nil {
		return httperr.Fail(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// AcceptInvite godoc
// @Summary      Accept invitation
// @Description  Creates an account using an invite token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body AcceptInviteRequest true "Invite details"
// @Success      200 {object} TokenPairEnvelope
// @Failure      400 {object} apiresponse.ErrorEnvelope "invalid body"
// @Failure      401 {object} apiresponse.ErrorEnvelope "invalid token"
// @Failure      409 {object} apiresponse.ErrorEnvelope "username already claimed"
// @Router       /auth/accept-invite [post]
func (h *AuthHandler) AcceptInvite(c echo.Context) error {
	var req AcceptInviteRequest
	if err := c.Bind(&req); err != nil {
		return httperr.InvalidBody(c)
	}

	profileInput := model.ProfileInput{}
	if req.Username != "" {
		profileInput.Username = &req.Username
	}

	tokens, err := h.svc.Signup.AcceptInvite(c.Request().Context(), req.Token, req.Password, profileInput)
	if err != nil {
		return httperr.Fail(c, err)
	}

	return c.JSON(http.StatusOK, tokenPair(tokens.AccessToken, tokens.RefreshToken))
}
