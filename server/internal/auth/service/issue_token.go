package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"server/internal/auth/config"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/token"
)

// issueTokenPair generates a signed HS256 access token and a cryptographic CSPRNG refresh token.
// The raw refresh token is returned to the caller, while its SHA-256 hash is persisted in the database.
func issueTokenPair(
	ctx context.Context,
	repo repository.AuthRepository,
	jwtIssuer *token.JWTIssuer,
	sessionConfig config.SessionConfig,
	userID uuid.UUID,
) (*model.TokenPair, error) {
	// 1. Issue signed HS256 JWT access token with configured TTL
	accessToken, err := jwtIssuer.Issue(userID, sessionConfig.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	// 2. Generate secure 32-byte CSPRNG refresh token and compute its SHA-256 hash
	rawRefreshToken, tokenHash, err := token.GenerateRandomToken()
	if err != nil {
		return nil, err
	}

	// 3. Persist the refresh token hash with configured expiration window
	expiresAt := time.Now().Add(sessionConfig.RefreshTokenTTL)
	_, err = repo.CreateRefreshToken(ctx, userID, tokenHash, expiresAt)
	if err != nil {
		return nil, err
	}

	// 4. Return plaintext token pair to caller
	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
	}, nil
}
