// Package config centralizes all startup configuration read from the environment.
//
// Architectural Philosophy:
// Rather than scattering os.Getenv calls through main.go, services and workers
// — where a missing variable becomes a runtime surprise several layers deep —
// every environment-derived value is parsed and validated once, during boot.
//
// If anything required is missing, Load returns an error listing every problem
// at once rather than the first one, so a broken .env takes one restart to
// diagnose instead of ten.
//
// Rule of thumb: if a component needs a configuration value, add it here and
// pass it down. Business logic packages must never call os.Getenv.
package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// defaultAuthRateLimitPerMinute is the number of requests per minute a single
// client may make to the unauthenticated auth endpoints when
// RATE_LIMIT_PER_MINUTE is unset.
//
// 5 is a deliberate default, not an arbitrary one. These endpoints are the ones
// that are reachable without a session and can burn server CPU (reset-password
// runs a bcrypt hash) or spend money (forgot-password sends mail), so the
// limiter is the first line of defence and 5 is far below any rate a real user
// or a real frontend produces. A deployment that raises it is accepting a
// weaker limit in exchange for shared-IP or client-behind-a-proxy conditions;
// that is a legitimate trade, but it should be a decision someone made
// deliberately rather than a number nobody looked at.
const defaultAuthRateLimitPerMinute = 5

// minJWTSecretBytes is the shortest signing secret Load will accept.
//
// HS256 security is bounded by the entropy of the key, so a 4-character
// secret is brute-forceable offline by anyone holding a token and a wordlist.
// 32 bytes is the output size of crypto/rand, which is the floor for a value
// intended to be unguessable.
const minJWTSecretBytes = 32

// defaultDevOrigins is the CORS allowlist used when CORS_ALLOWED_ORIGINS is
// unset.
//
// These are the Flutter web (localhost:3000) and Vite (localhost:5173) dev
// servers. The list is deliberately NOT defaulted to ["*"]: a wildcard is safe
// only while AllowCredentials is false, and a wildcard default means a
// forgotten env var silently opens the API to every origin the moment anyone
// flips that flag. Failing toward a narrow list is the safer default.
var defaultDevOrigins = []string{"http://localhost:3000", "http://localhost:5173"}

// Config holds every environment-derived value the application needs at startup.
type Config struct {
	// Port is the TCP port the HTTP server binds. Optional; defaults to
	// "8080". A missing PORT should not be fatal the way a missing
	// DATABASE_URL is — it is not a security-relevant value.
	Port string

	// DatabaseURL is the PostgreSQL connection string, e.g.
	// postgres://user:pass@host:5432/dbname?sslmode=disable.
	//
	// Required, and never defaulted. A hardcoded fallback connection string
	// would ship well-known credentials (postgres/postgres) into a
	// deployment that believed it was configured, and would default the
	// connection to sslmode=disable, carrying credentials and password
	// hashes in cleartext.
	DatabaseURL string

	// JWTSecret signs and verifies access tokens. Required and must be at
	// least minJWTSecretBytes bytes. Never defaulted: a fallback signing key
	// is a complete authentication bypass, because an attacker holding it can
	// mint a valid token for any user ID without touching the database.
	JWTSecret string

	// ResendAPIKey authenticates with the Resend transactional email API.
	//
	// Optional. When absent the application falls back to a no-op mailer that
	// logs message bodies to stdout, which is what makes local development
	// possible without a Resend account.
	//
	// The failure mode is real, so it is loud rather than fatal: a no-op
	// mailer accepts signups, logs the OTP to stdout and returns 201, so
	// nobody receives a verification code and nothing in the response
	// indicates why. Boot is not blocked — a developer who has not signed up
	// for an email provider should still be able to run the server — but the
	// fallback is announced at WARN. A deployment that has quietly lost its
	// key is then distinguishable from a healthy one at a glance, instead of
	// sitting there for the lifetime of the deployment failing quietly.
	ResendAPIKey string

	// ResendFromAddress is the sender address Resend is configured to send
	// from.
	//
	// Required only when ResendAPIKey is set, because Resend rejects
	// unverified senders. Requiring it unconditionally would block a no-key
	// local setup over a value nothing would ever read. When the key IS set
	// and this is missing, that is a real misconfiguration and Load fails —
	// otherwise the deployment boots, then fails on the first real signup.
	ResendFromAddress string

	// MailConfigured reports whether a real mail transport will be used.
	//
	// Load derives this rather than letting callers re-derive it, so the
	// decision lives in exactly one place. It is false when ResendAPIKey is
	// unset, meaning OTP codes and invitation links go to a no-op mailer and
	// are written to the log instead of being delivered.
	MailConfigured bool

	// AllowedOrigins is the CORS allowlist. Optional; defaults to
	// defaultDevOrigins. Set CORS_ALLOWED_ORIGINS to a comma-separated list
	// in staging and production.
	AllowedOrigins []string

	// TrustedProxyCIDRs lists the CIDR ranges of reverse proxies whose
	// X-Forwarded-For header may be believed (e.g. "10.0.0.0/8,172.16.0.0/12").
	//
	// When empty, the server is assumed to be directly on the internet and
	// RemoteAddr is the client IP. When set, X-Forwarded-For is walked
	// right-to-left, skipping hops inside these ranges, and the first
	// untrusted address is used as the client IP.
	//
	// This is what makes the rate limiter correct behind a load balancer. With
	// no trusted proxies configured, every request appears to originate from
	// the proxy, so all clients share one rate-limit bucket and five login
	// attempts from anyone locks out every user.
	TrustedProxyCIDRs []string

	// AppBaseURL is the public origin of the frontend, used to build links
	// that are emailed to users — currently the invitation link.
	//
	// Optional; defaults to http://localhost:3000 for local development.
	// Load forces https everywhere else, so an invite token is never delivered
	// over cleartext or embedded in a URL that a browser will keep in history
	// and forward in a Referer header.
	AppBaseURL string

	// SMTPHost is the mail submission server to send through, e.g.
	// smtp.gmail.com. Optional: unset the value falls back to Resend when
	// RESEND_API_KEY is present, and to the no-op mailer otherwise.
	//
	// When set, SMTPUsername, SMTPPassword and SMTPFrom are all required —
	// half-configured SMTP would otherwise accept a signup, fail partway
	// through sending the verification code, and report success to a user who
	// never receives it.
	SMTPHost string

	// SMTPPort is the submission port. Optional; defaults to "587", which is
	// the port that negotiates STARTTLS.
	SMTPPort string

	// SMTPUsername authenticates against SMTPHost. Required when SMTPHost is set.
	SMTPUsername string

	// SMTPPassword authenticates against SMTPHost. For Gmail this is an app
	// password rather than the account password — Google rejects plain
	// passwords on SMTP for accounts with two-step verification enabled.
	// Required when SMTPHost is set.
	SMTPPassword string

	// SMTPFrom is the sender address. Required when SMTPHost is set, and must
	// be the address that owns SMTPUsername — Gmail rejects a From that does
	// not match the authenticated account.
	SMTPFrom string

	// SMTPConfigured reports whether SMTP transport will be used.
	//
	// Derived here rather than at each call site so the decision to prefer
	// SMTP over Resend, or over the no-op fallback, is made in exactly one
	// place and cannot disagree with itself in two places.
	SMTPConfigured bool

	// GoogleClientID is the OAuth client ID used to verify Google ID tokens.
	// Optional today: the OAuth provider registry is empty and no OAuth route
	// is mounted, so an unset value is inert. It is read here so that wiring
	// the provider is a config change rather than a code change.
	GoogleClientID string

	// AuthRateLimitPerMinute caps requests per minute, per resolved client IP,
	// across the unauthenticated /auth endpoints.
	//
	// Optional; defaults to defaultAuthRateLimitPerMinute (5). A value below 1
	// falls back to the default, because zero does not mean "unlimited" to the
	// rate limiter — it means every auth request comes back 429.
	//
	// This is the one setting that has to move between environments. Every client
	// behind a proxy shares one resolved IP unless TRUSTED_PROXY_CIDRS says
	// otherwise, so a deployment where that is not configured correctly needs a
	// higher limit or the first few users lock everyone else out.
	AuthRateLimitPerMinute int
}

// Load reads .env if present, then reads and validates every environment
// variable into a Config.
//
// It returns an error describing every problem at once — not just the first —
// so a person repairing their .env does not have to run the binary repeatedly
// to discover each missing or invalid variable one at a time.
func Load() (*Config, error) {
	// A missing .env is normal in production, where real configuration
	// arrives through the process environment. It is only worth mentioning.
	if err := godotenv.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "config: no .env file found, reading from process environment")
	}

	cfg := &Config{
		Port:              os.Getenv("PORT"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		ResendFromAddress: os.Getenv("RESEND_FROM_ADDRESS"),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          os.Getenv("SMTP_PORT"),
		SMTPUsername:      os.Getenv("SMTP_USERNAME"),
		SMTPPassword:      os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:          os.Getenv("SMTP_FROM"),
		GoogleClientID:    os.Getenv("GOOGLE_CLIENT_ID"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = splitAndTrim(origins)
	} else {
		cfg.AllowedOrigins = defaultDevOrigins
	}

	if cidrs := os.Getenv("TRUSTED_PROXY_CIDRS"); cidrs != "" {
		cfg.TrustedProxyCIDRs = splitAndTrim(cidrs)
	}

	if base := os.Getenv("APP_BASE_URL"); base != "" {
		cfg.AppBaseURL = strings.TrimRight(base, "/")
	} else {
		cfg.AppBaseURL = "http://localhost:3000"
	}

	// Derived once, here, so the mailer decision is not re-derived by each
	// caller and the two can never disagree.
	cfg.MailConfigured = cfg.ResendAPIKey != ""

	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}
	cfg.SMTPConfigured = cfg.SMTPHost != ""

	// Optional tuning. Unset, or anything that does not parse as a positive
	// whole number, keeps the default — this is not a value worth failing a
	// boot over, and a bad number must not silently become zero, which would
	// reject every auth request rather than permit them.
	cfg.AuthRateLimitPerMinute = defaultAuthRateLimitPerMinute
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("RATE_LIMIT_PER_MINUTE"))); err == nil && v > 0 {
		cfg.AuthRateLimitPerMinute = v
	}

	return cfg, cfg.validate()
}

// validate collects every configuration problem rather than stopping at the
// first, so one boot reports the full picture. Problems are sorted by variable
// name because ranging over a Go map has randomized order, and a diffable,
// deterministic error message is easier to act on.
func (c *Config) validate() error {
	var problems []string

	// Required variables, reported as a set so an operator sees every gap.
	required := map[string]string{
		"DATABASE_URL": c.DatabaseURL,
		"JWT_SECRET":   c.JWTSecret,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("%s is required but unset", name))
		}
	}

	// RESEND_FROM_ADDRESS is required only when a Resend key is present. A
	// missing key falls back to a no-op mailer that logs to stdout, so
	// demanding a sender address in that case would block a legitimate local
	// setup over a value that would never be used. With a key set, though, a
	// missing sender is a real misconfiguration: Resend rejects unverified
	// senders, so the deployment would boot and then fail on the first real
	// signup instead of here.
	if c.ResendAPIKey != "" && strings.TrimSpace(c.ResendFromAddress) == "" {
		problems = append(problems,
			"RESEND_FROM_ADDRESS is required when RESEND_API_KEY is set")
	}

	// Same reasoning as RESEND_FROM_ADDRESS above: SMTP is opt-in through
	// SMTP_HOST, and once it is opted in a missing credential is a real
	// misconfiguration rather than something to fall back from. Without this
	// the server boots, then fails on the first verification email, having
	// already told the user it was sent.
	if c.SMTPHost != "" {
		for name, value := range map[string]string{
			"SMTP_USERNAME": c.SMTPUsername,
			"SMTP_PASSWORD": c.SMTPPassword,
			"SMTP_FROM":     c.SMTPFrom,
		} {
			if strings.TrimSpace(value) == "" {
				problems = append(problems, name+" is required when SMTP_HOST is set")
			}
		}
	}

	// "Required" is not the same as "unguessable". A one-character secret
	// satisfies the check above and still allows offline brute force of every
	// token ever signed, so the length is enforced separately with a message
	// that says how to fix it.
	if n := len(c.JWTSecret); n > 0 && n < minJWTSecretBytes {
		problems = append(problems, fmt.Sprintf(
			"JWT_SECRET is %d bytes, which is too short to resist offline brute force; "+
				"generate at least %d random bytes, for example: openssl rand -base64 %d",
			n, minJWTSecretBytes, minJWTSecretBytes))
	}

	// Emailed links must not travel in cleartext. localhost is exempt because
	// that traffic never leaves the machine.
	if !strings.HasPrefix(c.AppBaseURL, "http://localhost") &&
		!strings.HasPrefix(c.AppBaseURL, "https://") {
		problems = append(problems, fmt.Sprintf(
			"APP_BASE_URL %q must use https (http is allowed only for localhost)", c.AppBaseURL))
	}

	sort.Strings(problems)
	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// splitAndTrim parses a comma-separated environment variable, discarding
// empty entries so a trailing comma ("a,b,") does not produce a blank origin
// or a malformed CIDR.
func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
