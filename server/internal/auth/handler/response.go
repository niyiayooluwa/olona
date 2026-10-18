package handler

import "server/internal/auth/model"

// Success envelopes for the auth domain. The shared failure envelope and the
// "exactly one of data / error" rule live in internal/platform/apiresponse.

// TokenPairResponse represents the JWT access and refresh tokens.
type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TokenPairEnvelope wraps an access/refresh token pair.
type TokenPairEnvelope struct {
	Data TokenPairResponse `json:"data"`
}

// ProfileEnvelope wraps a user profile.
type ProfileEnvelope struct {
	Data *model.UserProfile `json:"data"`
}

// SignupEnvelope wraps the result of a signup attempt.
type SignupEnvelope struct {
	Data SignupData `json:"data"`
}

// SignupData tells the client whether signup is complete or waiting on email
// verification. Tokens are present only when verification is not required, so
// the client can proceed without a second round trip.
type SignupData struct {
	VerificationRequired bool               `json:"verification_required"`
	Tokens               *TokenPairResponse `json:"tokens,omitempty"`
}

// UsernameCheckEnvelope wraps the username availability result.
type UsernameCheckEnvelope struct {
	Data UsernameCheck `json:"data"`
}

// UsernameCheck reports whether a username is free.
type UsernameCheck struct {
	Available bool `json:"available"`
}

// tokenPair builds the standard token envelope.
func tokenPair(access, refresh string) TokenPairEnvelope {
	return TokenPairEnvelope{Data: TokenPairResponse{AccessToken: access, RefreshToken: refresh}}
}
