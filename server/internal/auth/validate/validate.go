// Package validate holds the input checks the auth layer applies to
// user-supplied values before they reach the database or any consumer.
package validate

import (
	"fmt"
	"net/mail"
	"net/url"
	"strings"

	"server/internal/auth/autherr"
)

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
