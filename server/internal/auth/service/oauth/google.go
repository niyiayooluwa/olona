package oauth

import (
	"context"
	"errors"

	"google.golang.org/api/idtoken"
)

// ErrInvalidGoogleToken is returned when Google rejects or fails to parse the ID token.
var ErrInvalidGoogleToken = errors.New("invalid google id token")

// GoogleProvider validates Google OpenID Connect ID tokens.
type GoogleProvider struct {
	clientID string
}

// NewGoogleProvider creates a new Google OAuth validator for the given client ID.
func NewGoogleProvider(clientID string) *GoogleProvider {
	return &GoogleProvider{clientID: clientID}
}

// VerifyToken validates the Google ID token and extracts the user identity.
func (provider *GoogleProvider) VerifyToken(ctx context.Context, tokenString string) (*Identity, error) {
	payload, err := idtoken.Validate(ctx, tokenString, provider.clientID)
	if err != nil {
		return nil, ErrInvalidGoogleToken
	}

	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return nil, errors.New("email claim missing from google token")
	}

	firstName, _ := payload.Claims["given_name"].(string)
	lastName, _ := payload.Claims["family_name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return &Identity{
		ProviderID: payload.Subject,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		AvatarURL:  picture,
	}, nil
}
