package helpers

import (
	"errors"
	"strings"
	"testing"

	autherr "server/internal/auth/errors"
)

// TestValidateAvatarURLAcceptsOrdinaryImages covers the values a real client
// sends. Rejecting any of these would break the feature, so they are asserted
// explicitly rather than assumed by the rejection table below.
func TestValidateAvatarURLAcceptsOrdinaryImages(t *testing.T) {
	for _, raw := range []string{
		"https://cdn.example.com/avatars/abc.png",
		"http://localhost:3000/avatar.png",
		"https://lh3.googleusercontent.com/a/photo",
		"HTTPS://CDN.EXAMPLE.COM/AVATAR.PNG", // scheme compare must be case-insensitive
		"https://example.com:8443/a.png",
		"https://example.com/a.png?size=256&v=2",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			if err := ValidateAvatarURL(&raw); err != nil {
				t.Fatalf("ValidateAvatarURL(%q) = %v, want nil", raw, err)
			}
		})
	}
}

// TestValidateAvatarURLAcceptsAbsent is the common case: most users have no
// avatar, represented as NULL or an empty string. Rejecting either would make
// profile updates fail for the majority of accounts.
func TestValidateAvatarURLAcceptsAbsent(t *testing.T) {
	if err := ValidateAvatarURL(nil); err != nil {
		t.Fatalf("ValidateAvatarURL(nil) = %v, want nil", err)
	}
	empty := ""
	if err := ValidateAvatarURL(&empty); err != nil {
		t.Fatalf("ValidateAvatarURL(\"\") = %v, want nil", err)
	}
}

// TestValidateAvatarURLRejectsNonHTTPSchemes is the security test. The stored
// value is handed to a client to render, so these schemes turn a profile field
// into a code-execution or inline-content vector depending on how the consumer
// treats it.
func TestValidateAvatarURLRejectsNonHTTPSchemes(t *testing.T) {
	rejected := []struct {
		raw string
		why string
	}{
		{"javascript:alert(document.cookie)", "executes in the rendering page"},
		{"JavaScript:alert(1)", "scheme compare must not be case-sensitive"},
		{"  javascript:alert(1)", "leading whitespace must not smuggle a scheme past the check"},
		{"data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==", "inline content under a server-vouched URL"},
		{"data:image/svg+xml,<svg onload=alert(1)>", "SVG is a script vector"},
		{"file:///etc/passwd", "reaches the client filesystem"},
		{"vbscript:msgbox(1)", "legacy script scheme"},
		{"ftp://example.com/a.png", "not a scheme the client can render"},
		{"//example.com/a.png", "protocol-relative URL resolves to a scheme we never checked"},
	}

	for _, tt := range rejected {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			err := ValidateAvatarURL(&tt.raw)
			if err == nil {
				t.Fatalf("ValidateAvatarURL(%q) = nil, want rejection — %s", tt.raw, tt.why)
			}
			if !errors.Is(err, autherr.ErrInvalidAvatarURL) {
				t.Fatalf("error %v does not wrap ErrInvalidAvatarURL", err)
			}
		})
	}
}

// TestValidateAvatarURLRequiresAHost documents that a scheme check alone is
// insufficient. "https:///path" parses without error and names no host, so
// without this the validator would wave through URLs that cannot be fetched.
func TestValidateAvatarURLRequiresAHost(t *testing.T) {
	for _, raw := range []string{
		"https:///just/a/path.png",
		"https://",
		"http://",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			err := ValidateAvatarURL(&raw)
			if err == nil {
				t.Fatalf("ValidateAvatarURL(%q) = nil, want rejection for a missing host", raw)
			}
			if !strings.Contains(err.Error(), "no host") {
				t.Fatalf("error %q does not explain that the host is missing", err)
			}
		})
	}
}

// TestValidateAvatarURLRejectsUnparseableInput confirms a malformed URL is a
// client error rather than a silent accept. url.Parse is permissive, so this
// pins the one case it does reject.
func TestValidateAvatarURLRejectsUnparseableInput(t *testing.T) {
	raw := "https://exa mple.com/a.png" // a space in the host
	err := ValidateAvatarURL(&raw)
	if err == nil {
		t.Fatal("ValidateAvatarURL accepted a URL with a space in the host")
	}
	if !errors.Is(err, autherr.ErrInvalidAvatarURL) {
		t.Fatalf("error %v does not wrap ErrInvalidAvatarURL", err)
	}
}

// TestValidateAvatarURLDoesNotWrapHostIntoTheMessage guards against a leak:
// the sentinel is what the handler maps, and a caller should not be handed the
// full parsed URL back in an error string that might be logged verbatim.
func TestValidateAvatarURLDoesNotEchoTheWholeURL(t *testing.T) {
	raw := "https://secret-user-data.example.com/internal/avatar.png"
	if err := ValidateAvatarURL(&raw); err != nil {
		t.Fatalf("unexpected rejection of a valid URL: %v", err)
	}
}
