// Package httpserver constructs the Echo router and global middleware pipeline.
package httpserver

import (
	"log/slog"
	"net/http"

	"server/internal/auth/handler"
	authmw "server/internal/auth/middleware"
	"server/internal/auth/repository"
	"server/internal/auth/token"
	"server/internal/platform/apiresponse"

	_ "server/docs"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Handlers bundles every domain's HTTP handler so we don't have a giant param list.
type Handlers struct {
	Auth *handler.AuthHandler
}

// NewRouter constructs the global HTTP pipeline with middlewares and routes.
//
// trustedProxyCIDRs declares which reverse proxies may be believed when they
// set X-Forwarded-For. See buildIPExtractor for why the rate limiter depends
// on it.
//
// authRateLimitPerMinute is requests per minute per resolved client IP across
// the unauthenticated auth routes. It arrives from configuration rather than a
// literal because the right number is environment-specific: a backend where
// every client resolves to the same proxy address needs a higher limit, and a
// production deployment does not.
func NewRouter(
	h Handlers,
	repo repository.AuthRepository,
	jwtIssuer *token.JWTIssuer,
	allowedOrigins,
	trustedProxyCIDRs []string,
	authRateLimitPerMinute int,
) *echo.Echo {
	e := echo.New()

	// Every error reply must be the envelope, including the ones Echo answers
	// on its own: an unknown route (404), a wrong method (405), an oversized
	// body from BodyLimit (413), and any panic caught by Recover (500).
	// apiresponse.ErrorHandler is what turns those into the envelope; without
	// it they come back as Echo's default {"message": "..."} and the client
	// would have to handle a second shape forever.
	e.HTTPErrorHandler = apiresponse.ErrorHandler

	// Resolve the true client IP before any middleware that keys on it runs.
	e.IPExtractor = buildIPExtractor(trustedProxyCIDRs)

	// State the trust model in the boot log. Which mode is active decides
	// whether the rate limiter buckets by the real caller or by one shared
	// proxy address, and nothing else in the logs distinguishes the two: a
	// deployment with a wrong TRUSTED_PROXY_CIDRS behaves exactly like a
	// correct one right up until somebody is locked out by everyone else.
	// The value itself is logged so a fix needs no archaeology.
	if len(trustedProxyCIDRs) == 0 {
		slog.Info("client IP resolution: direct connection address; X-Forwarded-For is ignored")
	} else {
		slog.Info("client IP resolution: X-Forwarded-For, trusting only the configured proxies",
			"cidr_count", len(trustedProxyCIDRs), "cidrs", trustedProxyCIDRs)
	}

	// Global middlewares
	e.Use(middleware.RequestID())
	// RequestLogger, not Logger: the latter is deprecated (SA1019) and the
	// former is the maintained equivalent.
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Every request body in this API is a small JSON object (the largest is a
	// password reset, well under 1KB). Without a ceiling, c.Bind streams the
	// body into a json.Decoder with nothing stopping it, so a single
	// unauthenticated request with a multi-gigabyte body is enough to OOM the
	// process. Registered after Recover so an oversized body is a clean 413
	// rather than a panic.
	e.Use(middleware.BodyLimit("16K"))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Interactive OpenAPI / Swagger UI documentation endpoint
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Health check (Public)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Rate limiter for the authentication endpoints.
	//
	// Applied to more than just login: /verify, /reset-password and
	// /forgot-password are the unauthenticated endpoints where an attacker can
	// make the server burn CPU or send mail at will. /reset-password in
	// particular runs a bcrypt hash at the configured cost (~260ms at cost 12)
	// before it ever consults the OTP table, so an unthrottled caller gets a
	// free CPU-burn primitive. The attempt counters in the service layer are
	// the second layer of defence; this is the first.
	//
	// Keyed through clientIPKey so the limiter sees the same resolved client IP
	// as the rest of the pipeline. Keying off r.RemoteAddr directly (the
	// deprecated httprate.LimitByIP) would put every client behind a load
	// balancer in one shared bucket, so five attempts from anyone would lock
	// out every user.
	//
	// The per-minute number comes from configuration because the correct value
	// depends on how well client IPs are being resolved. Behind a proxy whose
	// CIDRs are not trusted, every caller shares one bucket and a low limit
	// locks out the entire user base — which is a config mistake, and the
	// reason the trust mode is logged above rather than left implicit.
	loginRateLimit := authRateLimiter(e.IPExtractor, authRateLimitPerMinute)

	// Auth Public Routes
	authGroup := e.Group("/auth")
	authGroup.POST("/signup", h.Auth.Signup, loginRateLimit)
	authGroup.POST("/verify", h.Auth.Verify, loginRateLimit)
	authGroup.POST("/resend-otp", h.Auth.ResendOTP, loginRateLimit)
	authGroup.POST("/login", h.Auth.Login, loginRateLimit)
	authGroup.POST("/refresh", h.Auth.Refresh, loginRateLimit)
	authGroup.POST("/logout", h.Auth.Logout, loginRateLimit)
	authGroup.POST("/forgot-password", h.Auth.ForgotPassword, loginRateLimit)
	authGroup.POST("/reset-password", h.Auth.ResetPassword, loginRateLimit)
	// Stays under /auth and stays unauthenticated: the Swagger contract
	// documents GET /auth/check-username, and the check is part of the signup
	// flow. Moving it would change a published API path, and putting it behind
	// RequireAuth would break the registration screen it exists to serve.
	authGroup.GET("/check-username", h.Auth.CheckUsername, loginRateLimit)

	// Protected API Routes
	v1 := e.Group("/v1")
	v1.Use(authmw.RequireAuth(jwtIssuer), authmw.LoadUser(repo))

	// User/Profile Protected Routes
	userGroup := v1.Group("/users")
	userGroup.GET("/me", h.Auth.GetProfile)
	userGroup.PATCH("/me", h.Auth.UpdateProfile)
	userGroup.POST("/me/password", h.Auth.ChangePassword)
	userGroup.DELETE("/me", h.Auth.DeleteProfile)

	return e
}
