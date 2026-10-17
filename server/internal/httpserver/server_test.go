package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// req builds a request whose RemoteAddr and X-Forwarded-For are set
// independently, so a test can assert which one the extractor believed.
func req(remoteAddr, xff string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	r.RemoteAddr = remoteAddr
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

// TestBuildIPExtractorIgnoresXFFWithNoTrustedProxies is the security-critical
// case: with no proxy configured, a client-supplied X-Forwarded-For must have
// no effect. Trusting it unconditionally would let any caller choose its own
// rate-limit bucket by rewriting the header.
func TestBuildIPExtractorIgnoresXFFWithNoTrustedProxies(t *testing.T) {
	extractor := buildIPExtractor(nil)

	r := req("203.0.113.9:1234", "1.2.3.4")

	if got := extractor(r); got != "203.0.113.9" {
		t.Fatalf("extractor() = %q, want %q from RemoteAddr; a spoofed X-Forwarded-For was trusted", got, "203.0.113.9")
	}
}

// TestBuildIPExtractorWalksXFFPastTrustedProxies covers the deployed case: a
// request arrives from a trusted load balancer carrying the real client IP in
// X-Forwarded-For.
func TestBuildIPExtractorWalksXFFPastTrustedProxies(t *testing.T) {
	extractor := buildIPExtractor([]string{"10.0.0.0/8"})

	r := req("10.1.2.3:5678", "203.0.113.9")

	if got := extractor(r); got != "203.0.113.9" {
		t.Fatalf("extractor() = %q, want the X-Forwarded-For client IP %q", got, "203.0.113.9")
	}
}

// TestBuildIPExtractorRejectsSpoofedChainFromUntrustedPeer confirms the trust
// range is load-bearing. An X-Forwarded-For that arrives from an address
// outside the trusted ranges is ignored even when the header is populated,
// because otherwise any client could forge a chain and rotate its apparent IP.
func TestBuildIPExtractorRejectsSpoofedChainFromUntrustedPeer(t *testing.T) {
	extractor := buildIPExtractor([]string{"10.0.0.0/8"})

	// Peer is not a trusted proxy, so the header must not be believed.
	r := req("198.51.100.7:9000", "1.2.3.4")

	if got := extractor(r); got != "198.51.100.7" {
		t.Fatalf("extractor() = %q, want the peer's own address %q; an untrusted X-Forwarded-For was believed", got, "198.51.100.7")
	}
}

// TestBuildIPExtractorFallsBackWhenEveryCIDRIsInvalid documents the
// fail-closed behavior for a malformed TRUSTED_PROXY_CIDRS: the server still
// boots, in the mode that cannot be spoofed.
func TestBuildIPExtractorFallsBackWhenEveryCIDRIsInvalid(t *testing.T) {
	extractor := buildIPExtractor([]string{"not-a-cidr", "10.0.0.0/99"})

	r := req("203.0.113.9:1234", "1.2.3.4")

	if got := extractor(r); got != "203.0.113.9" {
		t.Fatalf("extractor() = %q, want RemoteAddr when no CIDR parsed", got)
	}
}

func TestBuildIPExtractorSkipsInvalidCIDRAndKeepsValidOnes(t *testing.T) {
	extractor := buildIPExtractor([]string{"garbage", "10.0.0.0/8"})

	r := req("10.1.2.3:5678", "203.0.113.9")

	if got := extractor(r); got != "203.0.113.9" {
		t.Fatalf("extractor() = %q, want the XFF client IP; one valid CIDR should have been kept", got)
	}
}

// TestClientIPKeyIsStableForIPv6InSamePrefix verifies IPv6 bucketing. A client
// with a delegated /64 can hold many addresses; keying on the full address
// would hand it a fresh rate-limit bucket for each one.
func TestClientIPKeyIsStableForSameIPv6Prefix(t *testing.T) {
	extractor := buildIPExtractor(nil)
	key := clientIPKey(extractor)

	first, err := key(req("[2001:db8:1:2::1]:443", ""))
	if err != nil {
		t.Fatalf("key() = %v", err)
	}
	second, err := key(req("[2001:db8:1:2::9]:443", ""))
	if err != nil {
		t.Fatalf("key() = %v", err)
	}

	if first != second {
		t.Fatalf("keys differ within one /64: %q vs %q; IPv6 rotation escapes the rate limit", first, second)
	}
}

// TestClientIPKeyDistinguishesDifferentIPv6Prefixes confirms the /64 bucketing
// is not over-collapsing: two genuinely different clients must not share a
// bucket, or one attacker could exhaust everyone else's allowance.
func TestClientIPKeyDistinguishesDifferentIPv6Prefixes(t *testing.T) {
	extractor := buildIPExtractor(nil)
	key := clientIPKey(extractor)

	first, _ := key(req("[2001:db8:1:2::1]:443", ""))
	second, _ := key(req("[2001:db8:9:9::1]:443", ""))

	if first == second {
		t.Fatalf("keys collided across different /64s: both %q", first)
	}
}

func TestClientIPKeyDistinguishesIPv4Addresses(t *testing.T) {
	extractor := buildIPExtractor(nil)
	key := clientIPKey(extractor)

	first, _ := key(req("203.0.113.1:80", ""))
	second, _ := key(req("203.0.113.2:80", ""))

	if first == second {
		t.Fatalf("keys collided for distinct IPv4 addresses: both %q", first)
	}
}

// TestClientIPKeyDoesNotCollapseOnUnresolvableValue guards the degenerate
// case: a value that is not an IP at all must not be merged with a real one.
func TestClientIPKeyDoesNotCollapseOnUnresolvableValue(t *testing.T) {
	extractor := buildIPExtractor(nil)
	key := clientIPKey(extractor)

	real, _ := key(req("203.0.113.1:80", ""))
	garbage, _ := key(req("not-an-ip:80", ""))

	if real == garbage {
		t.Fatalf("a non-IP value collided with a real address: both %q", real)
	}
	if garbage == "not-an-ip" {
		t.Fatal("unresolvable value was not namespaced; it must be distinguishable from a real IP")
	}
}
