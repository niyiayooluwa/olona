package config

import (
	"strings"
	"testing"
)

// setEnv populates the process environment for one test and registers cleanup
// that restores the previous value of every key it touched, so tests do not
// leak configuration into each other or into the developer's own shell.
func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
}

// clearEnv neutralizes every variable Load reads, so a test starts from a
// known-empty state regardless of what the developer has exported in their
// shell or left in server/.env.
//
// The values are set to the empty string rather than unset. Load treats empty
// as absent, and — more importantly — godotenv.Load() only populates variables
// that are not already present in the environment. Setting them to "" therefore
// also blocks the repository's real .env from leaking into the test, which
// unsetting would not.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PORT", "DATABASE_URL", "JWT_SECRET", "RESEND_API_KEY",
		"RESEND_FROM_ADDRESS", "CORS_ALLOWED_ORIGINS", "TRUSTED_PROXY_CIDRS",
		"APP_BASE_URL", "GOOGLE_CLIENT_ID", "RATE_LIMIT_PER_MINUTE",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM",
	} {
		t.Setenv(k, "")
	}
}

// validEnv is a minimal environment that passes validation, used as the base
// for tests that vary one field at a time.
func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":        "postgres://u:p@localhost:5432/olona?sslmode=disable",
		"JWT_SECRET":          strings.Repeat("k", 32),
		"RESEND_API_KEY":      "re_test",
		"RESEND_FROM_ADDRESS": "onboarding@resend.dev",
	}
}

func TestLoadAcceptsAValidEnvironment(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
		t.Fatal("Load() returned an empty required value")
	}
}

// TestLoadDefaultsToDevOrigins is the load-bearing test for the wildcard
// question: with CORS_ALLOWED_ORIGINS unset the allowlist must be the narrow
// dev pair, never "*".
func TestLoadDefaultsToDevOrigins(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			t.Fatal("AllowedOrigins defaulted to a wildcard; a forgotten env var must not open CORS to every origin")
		}
	}
	if len(cfg.AllowedOrigins) != len(defaultDevOrigins) {
		t.Fatalf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, defaultDevOrigins)
	}
}

func TestLoadParsesCommaSeparatedLists(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["CORS_ALLOWED_ORIGINS"] = "https://a.example, https://b.example ,"
	env["TRUSTED_PROXY_CIDRS"] = "10.0.0.0/8,172.16.0.0/12"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %v, want 2 entries", cfg.AllowedOrigins)
	}
	if cfg.AllowedOrigins[1] != "https://b.example" {
		t.Fatalf("AllowedOrigins[1] = %q, want trimmed value", cfg.AllowedOrigins[1])
	}
	if len(cfg.TrustedProxyCIDRs) != 2 {
		t.Fatalf("TrustedProxyCIDRs = %v, want 2 entries", cfg.TrustedProxyCIDRs)
	}
}

// TestLoadRejectsEveryMissingVariableAtOnce confirms validate reports the full
// set rather than the first. An operator repairing .env should not have to
// restart the binary once per missing key.
func TestLoadRejectsEveryMissingVariableAtOnce(t *testing.T) {
	clearEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want failure with no configuration set")
	}
	msg := err.Error()
	for _, want := range []string{
		"DATABASE_URL", "JWT_SECRET",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error does not mention %s:\n%s", want, msg)
		}
	}
}

// TestLoadAllowsMissingResendKey is the load-bearing test for the no-op
// mailer fallback. Someone without a Resend account must still be able to run
// the server, so an absent key is a warning, not a boot failure.
//
// It is also the reason a silent fallback would be dangerous, which is why
// MailConfigured is asserted here: the caller must be able to tell that no
// real mail will be sent.
func TestLoadAllowsMissingResendKey(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	delete(env, "RESEND_API_KEY")
	delete(env, "RESEND_FROM_ADDRESS")
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil so a developer without a Resend key can still boot", err)
	}
	if cfg.ResendAPIKey != "" {
		t.Fatalf("ResendAPIKey = %q, want empty", cfg.ResendAPIKey)
	}
	if cfg.MailConfigured {
		t.Fatal("MailConfigured = true with no Resend key; the caller would believe mail is being sent")
	}
}

func TestLoadReportsMailConfiguredWhenTheKeyIsPresent(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if !cfg.MailConfigured {
		t.Fatal("MailConfigured = false with a Resend key set; the caller would silently use the no-op mailer")
	}
}

// TestLoadRequiresFromAddressOnlyWhenKeyIsPresent pins the conditional. With a
// key set, a missing sender is a genuine misconfiguration — Resend rejects
// unverified senders, so the deployment would boot and then fail on the first
// real signup. Without a key, demanding one would block local setup over a
// value nothing would read.
func TestLoadRequiresFromAddressOnlyWhenKeyIsPresent(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["RESEND_API_KEY"] = "re_test"
	env["RESEND_FROM_ADDRESS"] = ""
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want failure for a Resend key with no sender address")
	}
	if !strings.Contains(err.Error(), "RESEND_FROM_ADDRESS") {
		t.Fatalf("error does not mention RESEND_FROM_ADDRESS:\n%s", err)
	}
}

// TestLoadRejectsShortJWTSecret is the check a sibling service's loader is missing:
// "required" alone lets a one-character secret through.
func TestLoadRejectsShortJWTSecret(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["JWT_SECRET"] = "short"
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want failure for a JWT secret below the minimum length")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("error does not mention JWT_SECRET:\n%s", err)
	}
}

func TestLoadRejectsCleartextAppBaseURL(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["APP_BASE_URL"] = "http://app.olona.example"
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want failure for a non-https APP_BASE_URL")
	}
	if !strings.Contains(err.Error(), "APP_BASE_URL") {
		t.Fatalf("error does not mention APP_BASE_URL:\n%s", err)
	}
}

// TestLoadAllowsLocalhostAppBaseURL documents the exemption: local development
// traffic never leaves the machine, so requiring https there would be noise.
func TestLoadAllowsLocalhostAppBaseURL(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["APP_BASE_URL"] = "http://localhost:3000"
	setEnv(t, env)

	if _, err := Load(); err != nil {
		t.Fatalf("Load() = %v, want nil for a localhost APP_BASE_URL", err)
	}
}

func TestLoadDefaultsPortAndAppBaseURL(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.AppBaseURL != "http://localhost:3000" {
		t.Fatalf("AppBaseURL = %q, want the localhost default", cfg.AppBaseURL)
	}
}

func TestLoadTrimsTrailingSlashFromAppBaseURL(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["APP_BASE_URL"] = "https://app.olona.example/"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.AppBaseURL != "https://app.olona.example" {
		t.Fatalf("AppBaseURL = %q, want the trailing slash removed", cfg.AppBaseURL)
	}
}

func TestLoadDefaultsTheRateLimit(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.AuthRateLimitPerMinute != 5 {
		t.Fatalf("AuthRateLimitPerMinute = %d, want the default 5", cfg.AuthRateLimitPerMinute)
	}
}

func TestLoadReadsTheRateLimitFromTheEnvironment(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["RATE_LIMIT_PER_MINUTE"] = "60"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.AuthRateLimitPerMinute != 60 {
		t.Fatalf("AuthRateLimitPerMinute = %d, want 60", cfg.AuthRateLimitPerMinute)
	}
}

func TestLoadFallsBackWhenTheRateLimitIsUnusable(t *testing.T) {
	// Zero would not mean "unlimited" — it means every auth request is
	// rejected, so a blank or mistyped value must not be able to reach the
	// limiter.
	for _, raw := range []string{"", "0", "-1", "lots"} {
		clearEnv(t)
		env := validEnv()
		env["RATE_LIMIT_PER_MINUTE"] = raw
		setEnv(t, env)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() with %q = %v, want nil", raw, err)
		}
		if cfg.AuthRateLimitPerMinute != 5 {
			t.Errorf("AuthRateLimitPerMinute = %q, want the default 5", raw)
		}
	}
}

func TestLoadReadsSMTPSettingsAndDefaultsThePort(t *testing.T) {
	clearEnv(t)
	env := validEnv()
	env["SMTP_HOST"] = "smtp.gmail.com"
	env["SMTP_USERNAME"] = "me@gmail.com"
	env["SMTP_PASSWORD"] = "abcd efgh ijkl mnop"
	env["SMTP_FROM"] = "me@gmail.com"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if !cfg.SMTPConfigured {
		t.Error("SMTPConfigured = false, want true when SMTP_HOST is set")
	}
	if cfg.SMTPPort != "587" {
		t.Errorf("SMTPPort = %q, want the default 587", cfg.SMTPPort)
	}
}

func TestLoadRequiresEverySMTPCredentialWhenTheHostIsSet(t *testing.T) {
	// A half-configured transport has to fail at boot rather than on the
	// first verification email — by then signup has already returned 201 and
	// told someone to check their inbox.
	for _, missing := range []string{"SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM"} {
		clearEnv(t)
		env := validEnv()
		env["SMTP_HOST"] = "smtp.gmail.com"
		env["SMTP_USERNAME"] = "me@gmail.com"
		env["SMTP_PASSWORD"] = "abcdefghijklmnop"
		env["SMTP_FROM"] = "me@gmail.com"
		env[missing] = ""
		setEnv(t, env)

		_, err := Load()
		if err == nil {
			t.Fatalf("Load() with %s unset = nil, want an error", missing)
		}
		if !strings.Contains(err.Error(), missing) {
			t.Errorf("Load() = %v, want it to name %s", err, missing)
		}
	}
}

func TestLoadAllowsSMTPToBeAbsentEntirely(t *testing.T) {
	clearEnv(t)
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.SMTPConfigured {
		t.Error("SMTPConfigured = true with no SMTP_HOST; the no-op/Resend fallback must still be reachable")
	}
}
