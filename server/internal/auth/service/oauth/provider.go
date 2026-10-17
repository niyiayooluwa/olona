// Package oauth provides third-party OAuth/OIDC token verification providers.
package oauth

import "context"

// Identity represents the verified user profile returned by an OAuth provider.
type Identity struct {
	ProviderID string // The unique identifier from the provider (e.g. Google sub)
	Email      string
	FirstName  string
	LastName   string
	AvatarURL  string
}

// Provider abstracts third-party token verification so adding new providers
// (Apple, GitHub, etc.) requires zero changes to the core auth service.
type Provider interface {
	VerifyToken(ctx context.Context, idToken string) (*Identity, error)
}
