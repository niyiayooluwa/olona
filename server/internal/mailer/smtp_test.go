package mailer

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// smtpRecord is what the fake server captured from one conversation.
type smtpRecord struct {
	from      string
	rcpt      []string
	message   string
	startTLS  bool
	authOnTLS bool
}

// fakeSMTP is a minimal SMTP server: enough of the protocol to accept a message
// and record what arrived.
type fakeSMTP struct {
	cert tls.Certificate

	mu   sync.Mutex
	rec  smtpRecord
	errs []error
}

// startFakeSMTP serves in plaintext on an ephemeral loopback port and returns
// the mailer pointed at it, plus a snapshot of what was received.
//
// Plaintext, because PlainAuth exempts loopback addresses. Advertising STARTTLS
// here would be handled by the real cipher; the TLS path is covered
// separately by startTLSSMTP, since that is the one Gmail actually takes.
func startFakeSMTP(t *testing.T) (*SMTPMailer, func() smtpRecord) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	s := &fakeSMTP{}
	go s.accept(ln, false)

	return newMailer(t, ln.Addr().String()), snapshot(s)
}

// startTLSSMTP serves a server that advertises STARTTLS, upgrades the
// connection, then requires AUTH over the encrypted channel — the order a real
// submission server enforces and the order Gmail takes.
//
// The client is given a config that trusts this test's own certificate, which
// is the same extension a deployment uses for a relay on a private CA.
func startTLSSMTP(t *testing.T) (*SMTPMailer, func() smtpRecord) {
	t.Helper()

	cert, pool := selfSignedCert(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	s := &fakeSMTP{cert: cert}
	go s.accept(ln, true)

	m := newMailer(t, ln.Addr().String())
	m.tlsConfig = &tls.Config{RootCAs: pool}
	return m, snapshot(s)
}

func snapshot(s *fakeSMTP) func() smtpRecord {
	return func() smtpRecord {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.rec
	}
}

func newMailer(t *testing.T, addr string) *SMTPMailer {
	t.Helper()
	host, port := splitAddr(t, addr)
	return NewSMTPMailer(host, port, "sender@example.com", "app-password",
		"sender@example.com", "https://app.example.com")
}

func splitAddr(t *testing.T, addr string) (host, port string) {
	t.Helper()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", addr, err)
	}
	return host, port
}

func (s *fakeSMTP) accept(ln net.Listener, startTLS bool) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed by cleanup
		}
		go func() {
			if err := s.serve(conn, startTLS); err != nil {
				s.mu.Lock()
				s.errs = append(s.errs, err)
				s.mu.Unlock()
			}
		}()
	}
}

func (s *fakeSMTP) serve(conn net.Conn, allowStartTLS bool) error {
	defer conn.Close()

	isTLS := false
	r := bufio.NewReader(conn)
	reply := func(line string) { _, _ = io.WriteString(conn, line+"\r\n") }
	reply("220 fake.test ESMTP")

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil
		}
		line = strings.TrimRight(line, "\r\n")
		verb := strings.ToUpper(strings.SplitN(line, " ", 2)[0])

		switch verb {
		case "EHLO", "HELO":
			switch {
			case allowStartTLS && !isTLS:
				_, _ = io.WriteString(conn, "250-fake.test\r\n250 STARTTLS\r\n")
			default:
				_, _ = io.WriteString(conn, "250-fake.test\r\n250 AUTH PLAIN\r\n")
			}

		case "STARTTLS":
			reply("220 Ready to start TLS")
			up := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{s.cert}})
			if err := up.Handshake(); err != nil {
				return err
			}
			// Swap both the connection and the reader. Leaving the plaintext
			// reader in place would silently discard the rest of the
			// handshake, and the test would pass for the wrong reason.
			conn = up
			r = bufio.NewReader(conn)
			isTLS = true

			s.mu.Lock()
			s.rec.startTLS = true
			s.mu.Unlock()

		case "AUTH":
			// Advertise AUTH only after the upgrade when TLS is in play, so a
			// client that authenticates before STARTTLS fails here rather than
			// quietly sending the password in clear.
			s.mu.Lock()
			s.rec.authOnTLS = isTLS
			s.mu.Unlock()
			reply("235 2.7.0 Authentication successful")

		case "MAIL":
			s.mu.Lock()
			s.rec.from = angleAddr(line)
			s.mu.Unlock()
			reply("250 OK")

		case "RCPT":
			// Refuse a recipient that names an unknown mailbox, so the client's
			// error path has something real to report. Without it a bug that
			// swallowed RCPT failures would pass — and signup would tell someone
			// to check an inbox that was never contacted.
			if strings.Contains(line, "invalid") {
				reply("550 5.1.1 no such user")
				continue
			}
			s.mu.Lock()
			s.rec.rcpt = append(s.rec.rcpt, angleAddr(line))
			s.mu.Unlock()
			reply("250 OK")

		case "DATA":
			reply("354 End data with <CR><LF>.<CR><LF>")
			msg, err := readDotStuffed(r)
			if err != nil {
				return err
			}
			s.mu.Lock()
			s.rec.message = msg
			s.mu.Unlock()
			reply("250 OK")

		case "QUIT":
			reply("221 Bye")
			return nil

		default:
			reply("250 OK")
		}
	}
}

// angleAddr pulls the address out of a protocol line, from the angle brackets
// when present and bare otherwise.
func angleAddr(line string) string {
	if i := strings.IndexByte(line, '<'); i >= 0 {
		if j := strings.IndexByte(line[i:], '>'); j >= 0 {
			return line[i+1 : i+j]
		}
	}
	if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// readDotStuffed collects message lines until the terminating dot, dropping the
// dot-stuffing escape.
func readDotStuffed(r *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			return strings.TrimSuffix(sb.String(), "\r\n"), nil
		}
		if strings.HasPrefix(trimmed, "..") {
			trimmed = trimmed[1:]
		}
		sb.WriteString(trimmed)
		sb.WriteString("\r\n")
	}
}

// selfSignedCert mints a loopback certificate and the pool that trusts it.
func selfSignedCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}, pool
}

func TestSMTPMailerDeliversAnOTPEmail(t *testing.T) {
	m, received := startFakeSMTP(t)

	if err := m.SendOTP(context.Background(), "dev@example.com", "481920"); err != nil {
		t.Fatalf("SendOTP() = %v, want nil", err)
	}

	rec := received()
	if rec.from != "sender@example.com" {
		t.Errorf("MAIL FROM = %q, want sender@example.com", rec.from)
	}
	if len(rec.rcpt) != 1 || rec.rcpt[0] != "dev@example.com" {
		t.Errorf("RCPT TO = %v, want [dev@example.com]", rec.rcpt)
	}

	// The exact contract the frontend relies on: the code, HTML, and headers
	// that make a mail client render it as HTML rather than show raw markup.
	for _, want := range []string{
		"From: sender@example.com",
		"To: dev@example.com",
		"Subject: Your Verification Code",
		"Content-Type: text/html; charset=UTF-8",
		"481920",
	} {
		if !strings.Contains(rec.message, want) {
			t.Errorf("message missing %q\n---\n%s", want, rec.message)
		}
	}
}

// TestSMTPMailerUpgradesToSTARTTLSBeforeAuth covers the path every real
// submission server takes — and the one the plain fake server never offers.
//
// Without this the encryption branch had no coverage at all: a bug that skipped
// STARTTLS would pass every other test, and the app password would travel to
// Gmail in clear text.
func TestSMTPMailerUpgradesToSTARTTLSBeforeAuth(t *testing.T) {
	m, received := startTLSSMTP(t)

	if err := m.SendOTP(context.Background(), "dev@example.com", "481920"); err != nil {
		t.Fatalf("SendOTP() = %v, want nil", err)
	}

	rec := received()
	if !rec.startTLS {
		t.Fatal("server never saw STARTTLS; the client did not upgrade")
	}
	if !rec.authOnTLS {
		t.Error("AUTH was sent before the upgrade, which puts the app password in clear text")
	}
	if !strings.Contains(rec.message, "481920") {
		t.Errorf("message did not arrive over the encrypted channel:\n%s", rec.message)
	}
}

func TestSMTPMailerSendInviteEmbedsTheConfiguredAppBaseURL(t *testing.T) {
	m, received := startFakeSMTP(t)

	if err := m.SendInvite(context.Background(), "dev@example.com", "tok-abc"); err != nil {
		t.Fatalf("SendInvite() = %v, want nil", err)
	}

	msg := received().message
	want := "https://app.example.com/accept-invite#token=tok-abc"
	if !strings.Contains(msg, want) {
		t.Errorf("message missing invite link %q\n---\n%s", want, msg)
	}
	// The fragment must survive into the href, otherwise the token is dropped
	// and the acceptance page has nothing to redeem.
	if !strings.Contains(msg, `href="https://app.example.com/accept-invite#token=tok-abc"`) {
		t.Errorf("invite URL not in an href attribute\n---\n%s", msg)
	}
}

func TestSMTPMailerDeliversAPasswordResetCode(t *testing.T) {
	m, received := startFakeSMTP(t)

	if err := m.SendPasswordReset(context.Background(), "dev@example.com", "771122"); err != nil {
		t.Fatalf("SendPasswordReset() = %v, want nil", err)
	}

	if !strings.Contains(received().message, "771122") {
		t.Errorf("message missing the reset code")
	}
}

func TestSMTPMailerSendsNothingWhenTheContextIsAlreadyCancelled(t *testing.T) {
	m, received := startFakeSMTP(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.SendOTP(ctx, "dev@example.com", "123456")
	if err == nil {
		t.Fatal("SendOTP() = nil, want an error for a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("SendOTP() = %v, want it to wrap context.Canceled", err)
	}
	if rec := received(); rec.message != "" {
		t.Errorf("cancelled request still delivered a message:\n%s", rec.message)
	}
}

func TestBuildMessageTerminatesEveryHeaderWithCRLF(t *testing.T) {
	// A bare LF in a header is a protocol violation: some servers reject it
	// outright, and others fold the rest of the message into that header.
	msg := buildMessage("a@example.com", "b@example.com", "Hi", "<p>line one</p>")

	if strings.Contains(msg, "\r\r") {
		t.Errorf("message contains a doubled carriage return:\n%q", msg)
	}
	for _, line := range strings.Split(msg, "\r\n") {
		if strings.Contains(line, "\n") {
			t.Errorf("bare LF survived in line %q", line)
		}
	}
}

func TestBuildMessageNormalizesPreexistingCRLF(t *testing.T) {
	msg := buildMessage("a@example.com", "b@example.com", "Hi", "one\r\ntwo")

	if strings.Contains(msg, "\r\r\n") {
		t.Errorf("existing CRLF was doubled:\n%q", msg)
	}
	if !strings.Contains(msg, "one\r\ntwo") {
		t.Errorf("expected a single CRLF between lines, got:\n%q", msg)
	}
}

func TestSMTPMailerDefaultsToPort587WhenUnset(t *testing.T) {
	// 587 is the submission port that negotiates STARTTLS. Leaving it unset
	// has to mean that, not an empty port string — net.JoinHostPort with an
	// empty port would dial "host:" and fail on every send.
	m := NewSMTPMailer("smtp.gmail.com", "", "u", "p", "f@example.com", "https://x.example")
	if m.port != "587" {
		t.Errorf("port = %q, want 587", m.port)
	}
}

func TestSMTPMailerReportsARefusedRecipient(t *testing.T) {
	// A bad recipient must surface as an error, not as a silently dropped
	// code that signup already told someone to look for.
	m, _ := startFakeSMTP(t)

	err := m.SendOTP(context.Background(), "nobody@invalid", "481920")
	if err == nil {
		t.Fatal("SendOTP() = nil, want an error for a rejected recipient")
	}
	if !strings.Contains(err.Error(), "SMTP") && !strings.Contains(err.Error(), "smtp") {
		t.Errorf("SendOTP() = %v, want the failure attributed to SMTP", err)
	}
}
