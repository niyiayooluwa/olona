// @title           Olona API
// @version         1.0
// @description     Backend for Olona — Auth and User Management
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT access token.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authconfig "server/internal/auth/config"
	authhandler "server/internal/auth/handler"
	"server/internal/auth/model"
	"server/internal/auth/repository"
	"server/internal/auth/service"
	"server/internal/auth/utils"
	"server/internal/config"
	db "server/internal/db/generated"
	"server/internal/httpserver"
	"server/internal/mailer"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Configuration.
	//
	// Every environment-derived value is resolved and validated here, before
	// anything is constructed. Load reports all problems at once and returns an
	// error for anything missing or unsafe, so a misconfigured deploy dies at
	// boot with an actionable message instead of half-starting.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Bound connection setup so a black-holed database host fails fast rather
	// than hanging the boot indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Database Connection
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// A lazy pool can start cleanly and only fail on the first real request, by
	// which point the server is already accepting traffic. Ping now so an
	// unreachable database is a boot failure.
	if err := pool.Ping(ctx); err != nil {
		slog.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// 3. Initialize Dependencies
	queries := db.New(pool)

	// Create the Auth Repository.
	//
	// WithTxSource is not optional. Without it the repository has no
	// transaction source and WithTx returns ErrTransactionsUnsupported, which
	// takes down every caller that needs atomicity — currently OTP
	// verification and password reset. Both would answer 500 on every
	// request, and the OTP brute-force attempt limit would never advance.
	authRepo := repository.NewAuthRepository(queries, repository.WithTxSource(pool))

	// Create JWT Issuer. cfg.JWTSecret is guaranteed non-empty and at least 32
	// bytes by this point; there is deliberately no fallback value, because a
	// known signing key lets an attacker mint a valid token for any user ID
	// without touching the database.
	jwtIssuer := utils.NewJWTIssuer(cfg.JWTSecret)

	// Initialize Mailer.
	//
	// Transport priority: SMTP first, then Resend, then the no-op fallback.
	// SMTP is checked first because it is explicit — setting SMTP_HOST says
	// "use this account", and having a RESEND_API_KEY left over from an
	// earlier attempt must not silently win.
	//
	// The fallback is deliberately loud. It is a real footgun: a no-op mailer
	// accepts signups, writes the OTP to the log and returns 201, so nobody
	// receives a verification code and nothing in the response says why. What
	// matters is that this state is impossible to mistake for a healthy one —
	// anyone tailing the boot log, or reading a support issue where the user
	// "never got the email", sees it immediately.
	var mailService model.Mailer
	switch {
	case cfg.SMTPConfigured:
		mailService = mailer.NewSMTPMailer(
			cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername,
			cfg.SMTPPassword, cfg.SMTPFrom, cfg.AppBaseURL)
		slog.Info("Initialized SMTP Mailer", "host", cfg.SMTPHost, "port", cfg.SMTPPort, "from", cfg.SMTPFrom)
	case cfg.MailConfigured:
		mailService = mailer.NewResendMailer(cfg.ResendAPIKey, cfg.ResendFromAddress, cfg.AppBaseURL)
		slog.Info("Initialized Resend Mailer", "from", cfg.ResendFromAddress)
	default:
		mailService = mailer.NewNoOpMailer()
		slog.Warn("RESEND_API_KEY is not set: falling back to the no-op mailer. "+
			"Emails will NOT be delivered — verification codes, password resets and "+
			"invitations are written to the log instead. This is expected for local "+
			"development, and a misconfiguration anywhere else.",
			"hint", "set SMTP_HOST (or RESEND_API_KEY) to send real mail")
	}

	// Initialize Auth Config.
	authCfg := authconfig.DefaultConfig()
	if err := authCfg.Validate(); err != nil {
		slog.Error("Invalid auth configuration", "error", err)
		os.Exit(1)
	}

	// 4. Initialize Services
	authServices := service.NewServices(authRepo, authCfg, jwtIssuer, mailService)

	// 5. Initialize Handlers
	authHandler := authhandler.NewAuthHandler(authServices)
	handlers := httpserver.Handlers{
		Auth: authHandler,
	}

	// 6. Build HTTP Router
	router := httpserver.NewRouter(
		handlers,
		authRepo,
		jwtIssuer,
		cfg.AllowedOrigins,
		cfg.TrustedProxyCIDRs,
		cfg.AuthRateLimitPerMinute,
	)

	// 7. Start HTTP Server with Graceful Shutdown
	port := cfg.Port

	// Timeouts are not tuning knobs, they are the only thing standing between
	// this process and a Slowloris attack. Go's zero value means "no limit",
	// so omitting them lets a handful of connections that dribble one byte of
	// headers per minute hold every accept-loop slot forever — the Recover
	// middleware never engages because no request ever completes.
	//
	// ReadHeaderTimeout is the one that matters most; the rest are defense in
	// depth against slow bodies and slow readers.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server startup failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited cleanly")
}
