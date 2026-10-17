package config

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigIsValid(t *testing.T) {
	// The shipped defaults are what production runs on. If these ever fail
	// validation, every deployment is broken, so this is the load-bearing case.
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("DefaultConfig() must be valid, got: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*AuthConfig)
		wantErr string
	}{
		{
			name:    "accepts the defaults",
			mutate:  func(*AuthConfig) {},
			wantErr: "",
		},
		{
			name:    "rejects a bcrypt cost below the bcrypt minimum",
			mutate:  func(c *AuthConfig) { c.Password.BcryptCost = 3 },
			wantErr: "BcryptCost is 3, outside the allowed range 4..31",
		},
		{
			name:    "rejects a bcrypt cost above the bcrypt maximum",
			mutate:  func(c *AuthConfig) { c.Password.BcryptCost = 32 },
			wantErr: "BcryptCost is 32, outside the allowed range 4..31",
		},
		{
			// The realistic misconfiguration: an env var that fails to parse
			// and lands on the zero value.
			name:    "rejects a zero bcrypt cost",
			mutate:  func(c *AuthConfig) { c.Password.BcryptCost = 0 },
			wantErr: "BcryptCost is 0, outside the allowed range 4..31",
		},
		{
			// Valid to bcrypt, but below the SEC-06 floor. Silent at runtime,
			// so it must be caught here.
			name:    "rejects a bcrypt cost below the SEC-06 minimum",
			mutate:  func(c *AuthConfig) { c.Password.BcryptCost = 10 },
			wantErr: "BcryptCost is 10, below the minimum of 12 required by SEC-06",
		},
		{
			name:    "rejects a zero password length",
			mutate:  func(c *AuthConfig) { c.Password.MinLength = 0 },
			wantErr: "Password.MinLength is 0, must be at least 1",
		},
		{
			// A zero attempt limit would let every OTP guess through the gate.
			name:    "rejects zero max OTP attempts",
			mutate:  func(c *AuthConfig) { c.Verification.MaxOTPAttempts = 0 },
			wantErr: "Verification.MaxOTPAttempts is 0, must be at least 1",
		},
		{
			name:    "rejects a negative max OTP attempts",
			mutate:  func(c *AuthConfig) { c.Verification.MaxOTPAttempts = -1 },
			wantErr: "Verification.MaxOTPAttempts is -1, must be at least 1",
		},
		{
			name:    "rejects a non-positive OTP TTL",
			mutate:  func(c *AuthConfig) { c.Verification.OTPTTL = 0 },
			wantErr: "Verification.OTPTTL is 0s, must be positive",
		},
		{
			name:    "rejects a non-positive access token TTL",
			mutate:  func(c *AuthConfig) { c.Session.AccessTokenTTL = 0 },
			wantErr: "Session.AccessTokenTTL is 0s, must be positive",
		},
		{
			// A refresh token that expires before the access token would
			// force a re-login on every access token expiry.
			name:    "rejects a refresh TTL shorter than the access TTL",
			mutate:  func(c *AuthConfig) { c.Session.RefreshTokenTTL = time.Minute },
			wantErr: "Session.RefreshTokenTTL (1m0s) must exceed AccessTokenTTL (15m0s)",
		},
		{
			name:    "rejects an unknown registration mode",
			mutate:  func(c *AuthConfig) { c.Registration.Mode = RegistrationMode("closed") },
			wantErr: `Registration.Mode is "closed", want "open" or "invite_only"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate() = nil, want an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	// Validating one field at a time would mean a deploy failing repeatedly,
	// one restart per bad value. All problems should surface together.
	cfg := DefaultConfig()
	cfg.Password.BcryptCost = 0
	cfg.Verification.MaxOTPAttempts = 0
	cfg.Registration.Mode = RegistrationMode("nope")

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected an error for an invalid config")
	}

	for _, want := range []string{"BcryptCost", "MaxOTPAttempts", "Registration.Mode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected the error to mention %s, got: %v", want, err)
		}
	}
}

func TestValidateAcceptsInviteOnlyRegistration(t *testing.T) {
	// Invite-only is the AUTH-10 target mode; it must not be flagged.
	cfg := DefaultConfig()
	cfg.Registration.Mode = RegistrationModeInviteOnly

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for invite_only", err)
	}
}
