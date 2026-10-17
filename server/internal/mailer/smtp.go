// Package mailer implements transactional email delivery.
package mailer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"crypto/tls"
)

// SMTPMailer implements model.Mailer by talking directly to an SMTP server.
//
// It exists so a project can deliver mail with an existing mail account — a
// Gmail address plus an app password, for instance — rather than signing up
// for a transactional-mail provider first.
type SMTPMailer struct {
	host string
	port string

	// username/password authenticate against the server. Both are sent
	// verbatim, so a value with a space in it must be quoted by whoever
	// supplies it; nothing here rewrites credentials.
	username string
	password string

	// from is the envelope and header sender. With Gmail this must be the
	// address that owns the app password, or the server rejects the message.
	from string

	// tlsConfig is the certificate verification used on STARTTLS. Nil means
	// the standard behaviour — verify against the system roots, with ServerName
	// set to host, which is what every public server needs.
	//
	// A non-nil value is for an internal relay presenting a certificate from a
	// private CA, where the public roots will not have it.
	tlsConfig *tls.Config

	// appBaseURL builds the invitation link. See SendInvite.
	appBaseURL string
}

// NewSMTPMailer constructs a Mailer that delivers through host:port.
//
// port is empty, default is 587 — the submission port that expects STARTTLS.
// Gmail also offers 465 with implicit TLS, but 587 is the one net/smtp
// negotiates in the way this type is written.
func NewSMTPMailer(host, port, username, password, from, appBaseURL string) *SMTPMailer {
	if port == "" {
		port = "587"
	}
	return &SMTPMailer{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		from:       from,
		appBaseURL: appBaseURL,
	}
}

// SendOTP sends an email verification code.
func (m *SMTPMailer) SendOTP(ctx context.Context, email, code string) error {
	subject := "Your Verification Code"
	body := fmt.Sprintf(
		"<p>Your verification code is: <strong>%s</strong></p>"+
			"<p>This code will expire in 15 minutes.</p>", code)

	if err := m.send(ctx, email, subject, body); err != nil {
		slog.Error("Failed to send OTP email via SMTP", "email", email, "error", err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}
	return nil
}

// SendInvite sends an invitation link with a token to a new user.
func (m *SMTPMailer) SendInvite(ctx context.Context, email, token string) error {
	// The token is in the URL fragment (#token=) so it never reaches the
	// server in the request line or an access log, and never appears in a
	// Referer header. This matches the Resend mailer exactly.
	inviteURL := fmt.Sprintf("%s/accept-invite#token=%s", m.appBaseURL, token)

	subject := "You've been invited to join Olona"
	body := fmt.Sprintf(
		"<p>You have been invited to join Olona!</p>"+
			"<p>Click the link below to set up your account:</p>"+
			"<p><a href=\"%s\">Accept Invitation</a></p>", inviteURL)

	if err := m.send(ctx, email, subject, body); err != nil {
		slog.Error("Failed to send Invite email via SMTP", "email", email, "error", err)
		return fmt.Errorf("failed to send Invite email: %w", err)
	}
	return nil
}

// SendPasswordReset sends a password reset code.
func (m *SMTPMailer) SendPasswordReset(ctx context.Context, email, code string) error {
	subject := "Password Reset Request"
	body := fmt.Sprintf(
		"<p>Your password reset code is: <strong>%s</strong></p>"+
			"<p>If you did not request this, please ignore this email.</p>", code)

	if err := m.send(ctx, email, subject, body); err != nil {
		slog.Error("Failed to send Password Reset email via SMTP", "email", email, "error", err)
		return fmt.Errorf("failed to send Password Reset email: %w", err)
	}
	return nil
}

// send performs the full SMTP conversation and delivers one message.
//
// This is written by hand rather than delegating to smtp.SendMail for two
// reasons. smtp.SendMail dials with no timeout, so a blackholed SMTP host
// would hold a signup request open for minutes, and it takes no context, so a
// request that is already cancelled would still send mail nobody is waiting
// for. Dialing here makes both behave.
func (m *SMTPMailer) send(ctx context.Context, to, subject, html string) error {
	addr := net.JoinHostPort(m.host, m.port)

	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp handshake with %s: %w", addr, err)
	}
	defer client.Close()

	// STARTTLS before AUTH, always. PlainAuth refuses to send credentials
	// over a cleartext connection, so the order is a requirement rather than
	// a preference — and without it a Gmail app password would cross the
	// network in the clear.
	if ok, _ := client.Extension("STARTTLS"); ok {
		// ServerName is always populated, whether or not a caller supplied
		// their own config: without it the handshake fails outright with
		// "either ServerName or InsecureSkipVerify must be specified", which
		// is a confusing error to get from a host that was configured
		// correctly except for this one field.
		cfg := &tls.Config{ServerName: m.host}
		if m.tlsConfig != nil {
			cfg = m.tlsConfig.Clone()
		}
		if cfg.ServerName == "" {
			cfg.ServerName = m.host
		}
		if err := client.StartTLS(cfg); err != nil {
			return fmt.Errorf("smtp STARTTLS with %s: %w", m.host, err)
		}
	}

	if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("smtp MAIL FROM %s: %w", m.from, err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO %s: %w", to, err)
	}

	// Honour cancellation between each round trip. The context already
	// covered the dial; this stops a request that timed out client-side from
	// finishing a delivery it no longer needs.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("smtp send cancelled: %w", err)
	}

	data, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}

	msg := buildMessage(m.from, to, subject, html)
	if _, err := io.WriteString(data, msg); err != nil {
		data.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := data.Close(); err != nil {
		return fmt.Errorf("smtp finish DATA: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("smtp send cancelled: %w", err)
	}
	return client.Quit()
}

// buildMessage renders the on-wire message.
//
// CRLF line endings are not optional: RFC 5321 says lines end with CRLF, and
// a server that sees bare LF may reject the message or fold it somewhere the
// HTML breaks.
func buildMessage(from, to, subject, html string) string {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	// Build everything in LF, then convert once at the end. Converting twice
	// would turn an existing CRLF into CRCRLF, which the receiver sees as an
	// empty line inside a header.
	body := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(html)
	return strings.ReplaceAll(strings.Join(append(headers, "", body), "\n"), "\n", "\r\n")
}
