package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/internal/platform/apiresponse"

	"github.com/labstack/echo/v4"
)

// serveWithLimit runs n requests from the given address through a bare Echo
// carrying only the auth rate limiter, and returns the status codes.
func serveWithLimit(t *testing.T, remoteAddr string, perMinute, n int) []int {
	t.Helper()

	e := echo.New()
	e.IPExtractor = buildIPExtractor(nil)
	e.Use(authRateLimiter(e.IPExtractor, perMinute))
	e.Any("/auth/login", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	codes := make([]int, 0, n)
	for i := 0; i < n; i++ {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req(remoteAddr, ""))
		codes = append(codes, rec.Code)
	}
	return codes
}

// TestAuthRateLimiterAllowsTheConfiguredNumberThenRejects covers the whole point
// of making the limit configurable: the number the deployment asks for is the
// number the limiter actually enforces.
//
// The literal 5 used to live in NewRouter, so there was nothing to test — the
// limit and its implementation could only be changed together, and neither was
// reachable from a test. Now a wrong wiring is a failing test rather than
// something discovered when a user is locked out.
func TestAuthRateLimiterAllowsTheConfiguredNumberThenRejects(t *testing.T) {
	const perMinute = 3

	codes := serveWithLimit(t, "203.0.113.10:5000", perMinute, perMinute+2)

	for i := 0; i < perMinute; i++ {
		if codes[i] != http.StatusOK {
			t.Errorf("request %d = %d, want 200 within the limit", i+1, codes[i])
		}
	}
	for i := perMinute; i < len(codes); i++ {
		if codes[i] != http.StatusTooManyRequests {
			t.Errorf("request %d = %d, want 429 past the limit", i+1, codes[i])
		}
	}
}

func TestAuthRateLimiterHonoursARaisedLimit(t *testing.T) {
	// The deployment-side case: a backend where every client resolves to the
	// same proxy address needs a limit high enough for a handful of people to
	// share one bucket. If raising the setting did not actually raise the limit,
	// the deployment would lock users out at the old number.
	const perMinute = 60

	codes := serveWithLimit(t, "203.0.113.10:5000", perMinute, 6)

	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("request %d = %d, want 200 at a limit of %d", i+1, code, perMinute)
		}
	}
}

func TestAuthRateLimiterKeepsSeparateBucketsPerResolvedClient(t *testing.T) {
	// One person exhausting their bucket must not lock out anyone else, which
	// only holds if the limiter keys on the resolved client IP.
	limiter := authRateLimiter(buildIPExtractor(nil), 1)

	call := func(remoteAddr string) int {
		e := echo.New()
		e.IPExtractor = buildIPExtractor(nil)
		e.Use(limiter)
		e.Any("/auth/login", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req(remoteAddr, ""))
		return rec.Code
	}

	if got := call("198.51.100.7:1000"); got != http.StatusOK {
		t.Fatalf("first client, first request = %d, want 200", got)
	}
	if got := call("198.51.100.7:1000"); got != http.StatusTooManyRequests {
		t.Fatalf("first client, second request = %d, want 429", got)
	}
	if got := call("198.51.100.8:1000"); got != http.StatusOK {
		t.Fatalf("second client, first request = %d, want 200 — one client must not lock out another", got)
	}
}

// TestAuthRateLimiterEmitsEnvelope covers the one 429 the handlers do not
// produce: httprate answers over-limit requests itself, and without a limit
// handler its body is plain-text "Too Many Requests" rather than the envelope.
func TestAuthRateLimiterEmitsEnvelope(t *testing.T) {
	e := echo.New()
	e.Use(authRateLimiter(echo.ExtractIPDirect(), 1))
	e.POST("/limited", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })

	first := httptest.NewRecorder()
	e.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/limited", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want 204 (body: %s)", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	e.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/limited", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429 (body: %s)", second.Code, second.Body.String())
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("a 429 must carry Retry-After")
	}
	var got apiresponse.ErrorEnvelope
	if err := json.Unmarshal(second.Body.Bytes(), &got); err != nil {
		t.Fatalf("429 body is not the envelope: %v (body: %s)", err, second.Body.String())
	}
	if got.Error.Code != "rate_limited" {
		t.Fatalf("code = %q, want rate_limited", got.Error.Code)
	}
}
