package httpserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"server/internal/platform/apiresponse"

	"github.com/go-chi/httprate"
	"github.com/labstack/echo/v4"
)

// authRateLimiter builds the middleware that caps the unauthenticated auth
// routes.
//
// extractor is threaded in rather than derived here so the limiter buckets on
// exactly the client IP the rest of the pipeline resolved. Deriving it a second
// time would be a chance for the two to disagree, and a disagreement is
// invisible until one user locks out everyone else.
func authRateLimiter(extractor echo.IPExtractor, perMinute int) echo.MiddlewareFunc {
	return echo.WrapMiddleware(
		httprate.LimitBy(perMinute, time.Minute, clientIPKey(extractor),
			// Without a handler of our own, httprate answers over-limit
			// requests with a plain-text "Too Many Requests" body — the one
			// 429 that would not be the envelope.
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "60")
				apiresponse.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please slow down.")
			}),
		),
	)
}

// clientIPKey adapts an Echo IPExtractor to the key function httprate expects.
//
// The extractor is threaded through rather than re-derived so the rate limiter
// buckets on exactly the same client IP the rest of the request pipeline sees,
// whether the deployment is direct or proxied.
func clientIPKey(extractor echo.IPExtractor) httprate.KeyFunc {
	return func(r *http.Request) (string, error) {
		ip := extractor(r)
		// Bucket IPv6 by /64 rather than by full address. A client with a
		// delegated prefix can rotate addresses inside it, and keying on the
		// full address would hand it a fresh rate-limit bucket for free.
		//
		// CanonicalizeIP rewrites IPv6 to its /64 prefix and returns IPv4
		// unchanged, so a rewritten key means the input was a real IPv6
		// address. An unchanged key is either a genuine IPv4 address, which
		// is already the right bucket, or a value that is not an IP at all —
		// which is prefixed so an unresolvable client cannot be conflated
		// with a real one.
		canonical := httprate.CanonicalizeIP(ip)
		if canonical != ip || net.ParseIP(ip) != nil {
			return canonical, nil
		}
		return fmt.Sprintf("unresolvable:%s", ip), nil
	}
}
