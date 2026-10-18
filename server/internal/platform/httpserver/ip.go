package httpserver

import (
	"log/slog"
	"net"
	"strings"

	"github.com/labstack/echo/v4"
)

// buildIPExtractor returns the Echo IPExtractor matching the deployment's trust
// model.
//
//   - No trusted proxies configured -> ExtractIPDirect: uses r.RemoteAddr and
//     ignores X-Forwarded-For entirely. Correct for a server reachable
//     directly on the internet, and the only safe default when the proxy chain
//     is unknown.
//   - Trusted proxies configured -> ExtractIPFromXFFHeader with one TrustIPRange
//     per CIDR. Echo walks X-Forwarded-For right-to-left, discarding hops that
//     fall inside a trusted range, and returns the first untrusted address.
//
// Trusting the whole X-Forwarded-For header without a range check would let any
// client set its own apparent IP and walk straight through the rate limiter, so
// the CIDR list is what makes header-based resolution safe. Adding or removing a
// proxy is a config change, never a code change.
func buildIPExtractor(cidrs []string) echo.IPExtractor {
	if len(cidrs) == 0 {
		return echo.ExtractIPDirect()
	}

	opts := make([]echo.TrustOption, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			// A malformed entry is a configuration mistake worth reporting, but
			// skipping it is safer than refusing to boot: the server comes up in
			// direct mode, which cannot be spoofed. Silently trusting a garbage
			// range would be the dangerous outcome.
			slog.Warn("TRUSTED_PROXY_CIDRS: skipping invalid CIDR", "cidr", cidr, "error", err)
			continue
		}
		opts = append(opts, echo.TrustIPRange(network))
	}

	// Every entry was malformed, so there is nothing to trust.
	if len(opts) == 0 {
		return echo.ExtractIPDirect()
	}
	return echo.ExtractIPFromXFFHeader(opts...)
}
