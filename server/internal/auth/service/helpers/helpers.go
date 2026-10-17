// Package helpers provides internal utility and mapping functions for auth services.
package helpers

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"server/internal/auth/config"
	autherr "server/internal/auth/errors"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/utils"
	db "server/internal/db/generated"
)

// IssueTokenPair generates a signed HS256 access token and a cryptographic CSPRNG refresh token.
// The raw refresh token is returned to the caller, while its SHA-256 hash is persisted in the database.
func IssueTokenPair(
	ctx context.Context,
	repo repository.AuthRepository,
	jwtIssuer *utils.JWTIssuer,
	sessionConfig config.SessionConfig,
	userID uuid.UUID,
) (*model.TokenPair, error) {
	// 1. Issue signed HS256 JWT access token with configured TTL
	accessToken, err := jwtIssuer.Issue(userID, sessionConfig.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	// 2. Generate secure 32-byte CSPRNG refresh token and compute its SHA-256 hash
	rawRefreshToken, tokenHash, err := utils.GenerateRandomToken()
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

// ValidateEmail ensures the provided string is structurally a valid email address.
func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return autherr.ErrInvalidEmail
	}
	return nil
}

// ValidatePassword enforces password policy constraints.
func ValidatePassword(password string, minLength int) error {
	if len(password) < minLength {
		return autherr.ErrPasswordTooShort
	}
	// bcrypt max length is 72 bytes. Check bytes, not runes.
	if len(password) > 72 {
		return autherr.ErrPasswordTooLong
	}
	return nil
}

// ValidateUsername enforces username format and length constraints.
// Usernames must be lowercase alphanumeric and underscores.
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 {
		return autherr.ErrInvalidUsername
	}
	for _, r := range username {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return autherr.ErrInvalidUsername
		}
	}
	return nil
}

// ValidateProfileName ensures profile fields (first name, last name, display name) do not exceed database limits.
func ValidateProfileName(name string) error {
	if len(name) > 255 {
		return autherr.ErrInvalidProfileName
	}
	return nil
}

// NormalizeEmail canonicalizes an email string by converting it to lowercase and stripping leading/trailing whitespace.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// TextToPtr converts an sqlc-generated nullable pgtype.Text column into a Go *string pointer.
func TextToPtr(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	return &text.String
}

// ValidateAvatarURL rejects an avatar URL that is not a plain http(s) URL.
//
// The value is stored verbatim and later handed to a client to render, so an
// unchecked string becomes an injection surface the moment any consumer treats
// it as a URI rather than as opaque text. The schemes worth rejecting:
//
//   - javascript: executes in the page that renders it.
//   - data: lets a caller supply arbitrary inline content, including markup,
//     under a URL the server itself vouched for.
//   - file: and other local schemes reach the client filesystem.
//
// An empty or absent URL is accepted, because "no avatar" is the common case
// and is represented as NULL rather than as an empty string.
//
// This is validation, not sanitization. It checks the scheme and requires a
// host, but deliberately does not fetch the URL to see where it resolves —
// doing so would turn a profile update into a server-side request forgery
// primitive. The stored value remains untrusted input; only its shape is
// constrained here.
func ValidateAvatarURL(raw *string) error {
	if raw == nil || *raw == "" {
		return nil
	}

	parsed, err := url.Parse(*raw)
	if err != nil {
		return fmt.Errorf("%w: %v", autherr.ErrInvalidAvatarURL, err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("%w: scheme %q is not allowed", autherr.ErrInvalidAvatarURL, parsed.Scheme)
	}

	// A scheme alone is not enough: "https:///path" parses cleanly and has no
	// host to fetch from, so requiring one closes the gap.
	if parsed.Host == "" {
		return fmt.Errorf("%w: URL has no host", autherr.ErrInvalidAvatarURL)
	}

	return nil
}

// MapToUserProfile converts internal database records (db.User and db.Profile) into the public domain UserProfile representation.
func MapToUserProfile(user db.User, profile db.Profile) *model.UserProfile {
	return &model.UserProfile{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   TextToPtr(profile.FirstName),
		LastName:    TextToPtr(profile.LastName),
		Username:    TextToPtr(profile.Username),
		DisplayName: TextToPtr(profile.DisplayName),
		AvatarURL:   TextToPtr(profile.AvatarUrl),
		IsVerified:  user.EmailVerifiedAt != nil,
	}
}
