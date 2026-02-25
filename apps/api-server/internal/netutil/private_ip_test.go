package netutil

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// === RFC 1918 — 10.0.0.0/8 ===
		{name: "10.0.0.0 (start of 10/8)", ip: "10.0.0.0", expected: true},
		{name: "10.0.0.1 (typical private)", ip: "10.0.0.1", expected: true},
		{name: "10.255.255.255 (end of 10/8)", ip: "10.255.255.255", expected: true},
		{name: "10.128.64.32 (mid range 10/8)", ip: "10.128.64.32", expected: true},

		// === RFC 1918 — 172.16.0.0/12 ===
		{name: "172.16.0.0 (start of 172.16/12)", ip: "172.16.0.0", expected: true},
		{name: "172.16.0.1 (typical private 172)", ip: "172.16.0.1", expected: true},
		{name: "172.31.255.255 (end of 172.16/12)", ip: "172.31.255.255", expected: true},
		{name: "172.20.10.5 (mid range 172.16/12)", ip: "172.20.10.5", expected: true},
		{name: "172.15.255.255 (just below 172.16/12)", ip: "172.15.255.255", expected: false},
		{name: "172.32.0.0 (just above 172.16/12)", ip: "172.32.0.0", expected: false},

		// === RFC 1918 — 192.168.0.0/16 ===
		{name: "192.168.0.0 (start of 192.168/16)", ip: "192.168.0.0", expected: true},
		{name: "192.168.0.1 (common router)", ip: "192.168.0.1", expected: true},
		{name: "192.168.1.1 (common LAN)", ip: "192.168.1.1", expected: true},
		{name: "192.168.255.255 (end of 192.168/16)", ip: "192.168.255.255", expected: true},

		// === Loopback — 127.0.0.0/8 ===
		{name: "127.0.0.0 (start of loopback)", ip: "127.0.0.0", expected: true},
		{name: "127.0.0.1 (localhost)", ip: "127.0.0.1", expected: true},
		{name: "127.255.255.255 (end of loopback)", ip: "127.255.255.255", expected: true},
		{name: "127.0.0.2 (alternate loopback)", ip: "127.0.0.2", expected: true},

		// === Link-local — 169.254.0.0/16 ===
		{name: "169.254.0.1 (link-local)", ip: "169.254.0.1", expected: true},
		{name: "169.254.169.254 (AWS metadata)", ip: "169.254.169.254", expected: true},

		// === Special-purpose ranges ===
		{name: "0.0.0.0 (RFC 1122 this network)", ip: "0.0.0.0", expected: true},
		{name: "100.64.0.1 (RFC 6598 CGN/shared)", ip: "100.64.0.1", expected: true},
		{name: "100.127.255.255 (end of CGN)", ip: "100.127.255.255", expected: true},
		{name: "192.0.0.1 (IETF protocol assignments)", ip: "192.0.0.1", expected: true},
		{name: "192.0.2.1 (TEST-NET-1)", ip: "192.0.2.1", expected: true},
		{name: "198.18.0.1 (benchmarking)", ip: "198.18.0.1", expected: true},
		{name: "198.51.100.1 (TEST-NET-2)", ip: "198.51.100.1", expected: true},
		{name: "203.0.113.1 (TEST-NET-3)", ip: "203.0.113.1", expected: true},
		{name: "224.0.0.1 (multicast)", ip: "224.0.0.1", expected: true},
		{name: "239.255.255.255 (end of multicast)", ip: "239.255.255.255", expected: true},
		{name: "240.0.0.1 (reserved)", ip: "240.0.0.1", expected: true},
		{name: "255.255.255.255 (broadcast)", ip: "255.255.255.255", expected: true},

		// === IPv6 loopback ===
		{name: "::1 (IPv6 loopback)", ip: "::1", expected: true},

		// === IPv6 unique local (fc00::/7) ===
		{name: "fc00::1 (IPv6 unique local)", ip: "fc00::1", expected: true},
		{name: "fd00::1 (IPv6 unique local fd)", ip: "fd00::1", expected: true},
		{name: "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff (end of ULA)", ip: "fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", expected: true},

		// === IPv6 link-local (fe80::/10) ===
		{name: "fe80::1 (IPv6 link-local)", ip: "fe80::1", expected: true},
		{name: "fe80::abcd:ef01:2345:6789", ip: "fe80::abcd:ef01:2345:6789", expected: true},
		{name: "febf::1 (end of fe80::/10)", ip: "febf::1", expected: true},

		// === IPv6 documentation (2001:db8::/32) ===
		{name: "2001:db8::1 (documentation)", ip: "2001:db8::1", expected: true},

		// === IPv6 unspecified ===
		{name: ":: (IPv6 unspecified)", ip: "::", expected: true},

		// === IPv6 multicast (ff00::/8) ===
		{name: "ff02::1 (IPv6 multicast all-nodes)", ip: "ff02::1", expected: true},
		{name: "ff05::2 (IPv6 site-local multicast)", ip: "ff05::2", expected: true},

		// === IPv4-mapped IPv6 addresses ===
		{name: "::ffff:127.0.0.1 (mapped loopback)", ip: "::ffff:127.0.0.1", expected: true},
		{name: "::ffff:10.0.0.1 (mapped private 10)", ip: "::ffff:10.0.0.1", expected: true},
		{name: "::ffff:192.168.1.1 (mapped private 192.168)", ip: "::ffff:192.168.1.1", expected: true},
		{name: "::ffff:172.16.0.1 (mapped private 172.16)", ip: "::ffff:172.16.0.1", expected: true},
		{name: "::ffff:8.8.8.8 (mapped public)", ip: "::ffff:8.8.8.8", expected: false},

		// === Public IPs (should return false) ===
		{name: "8.8.8.8 (Google DNS)", ip: "8.8.8.8", expected: false},
		{name: "1.1.1.1 (Cloudflare DNS)", ip: "1.1.1.1", expected: false},
		{name: "104.16.132.229 (Cloudflare)", ip: "104.16.132.229", expected: false},
		{name: "142.250.74.110 (Google)", ip: "142.250.74.110", expected: false},
		{name: "93.184.216.34 (example.com)", ip: "93.184.216.34", expected: false},
		{name: "54.239.28.85 (AWS public)", ip: "54.239.28.85", expected: false},
		{name: "2606:4700::6810:84e5 (Cloudflare IPv6)", ip: "2606:4700::6810:84e5", expected: false},
		{name: "2001:4860:4860::8888 (Google DNS IPv6)", ip: "2001:4860:4860::8888", expected: false},

		// === Edge cases around boundaries ===
		{name: "100.63.255.255 (just below CGN)", ip: "100.63.255.255", expected: false},
		{name: "100.128.0.0 (just above CGN)", ip: "100.128.0.0", expected: false},
		{name: "11.0.0.0 (just above 10/8)", ip: "11.0.0.0", expected: false},
		{name: "9.255.255.255 (just below 10/8)", ip: "9.255.255.255", expected: false},
		{name: "126.255.255.255 (just below 127/8)", ip: "126.255.255.255", expected: false},
		{name: "128.0.0.0 (just above 127/8)", ip: "128.0.0.0", expected: false},
		{name: "192.167.255.255 (just below 192.168/16)", ip: "192.167.255.255", expected: false},
		{name: "192.169.0.0 (just above 192.168/16)", ip: "192.169.0.0", expected: false},
		{name: "fec0::1 (above fe80::/10, not in fc00::/7)", ip: "fec0::1", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.NotNil(t, ip, "failed to parse IP: %s", tt.ip)
			result := IsPrivateIP(ip)
			assert.Equal(t, tt.expected, result, "IsPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
		})
	}
}

func TestIsPrivateIP_NilIP(t *testing.T) {
	// A nil IP should not match any CIDR and return false.
	assert.False(t, IsPrivateIP(nil))
}

func TestIsPrivateURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Private/internal — should return true
		{name: "localhost URL", url: "http://localhost/path", expected: true},
		{name: "127.0.0.1 URL", url: "http://127.0.0.1:8080/api", expected: true},
		{name: "10.x URL", url: "http://10.0.0.5/admin", expected: true},
		{name: "192.168 URL", url: "https://192.168.1.1/config", expected: true},
		{name: "IPv6 loopback URL", url: "http://[::1]:3000/", expected: true},

		// Invalid/unparseable — should return true (fail closed)
		{name: "empty string", url: "", expected: true},
		{name: "no hostname", url: "http:///path", expected: true},
		{name: "unresolvable host", url: "http://this-host-does-not-exist-openoms-test.invalid/x", expected: true},

		// Public URLs — should return false
		{name: "google.com", url: "https://www.google.com/", expected: false},
		{name: "cloudflare DNS", url: "http://1.1.1.1/cdn-cgi", expected: false},
		{name: "example.com with path", url: "https://example.com/path?q=1", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrivateURL(tt.url)
			assert.Equal(t, tt.expected, result, "IsPrivateURL(%q) = %v, want %v", tt.url, result, tt.expected)
		})
	}
}

func TestSafeHTTPClient(t *testing.T) {
	t.Run("returns non-nil client", func(t *testing.T) {
		client := SafeHTTPClient(10 * time.Second)
		assert.NotNil(t, client)
	})

	t.Run("respects provided timeout", func(t *testing.T) {
		timeout := 15 * time.Second
		client := SafeHTTPClient(timeout)
		assert.Equal(t, timeout, client.Timeout)
	})

	t.Run("has custom transport with SSRF protection", func(t *testing.T) {
		client := SafeHTTPClient(5 * time.Second)
		assert.NotNil(t, client.Transport, "transport should be set for SSRF protection")
		transport, ok := client.Transport.(*http.Transport)
		assert.True(t, ok, "transport should be *http.Transport")
		assert.NotNil(t, transport.DialContext, "DialContext should be set to NoPrivateDialer")
	})

	t.Run("zero timeout is accepted", func(t *testing.T) {
		client := SafeHTTPClient(0)
		assert.NotNil(t, client)
		assert.Equal(t, time.Duration(0), client.Timeout)
	})
}

func TestNoPrivateDialer(t *testing.T) {
	t.Run("returns non-nil function", func(t *testing.T) {
		dialer := NoPrivateDialer()
		assert.NotNil(t, dialer)
	})
}

func TestPrivateCIDRs_Initialized(t *testing.T) {
	// Verify that the init function populated privateCIDRs with the expected number of ranges.
	// 15 IPv4 ranges + 6 IPv6 ranges = 21 total
	assert.NotEmpty(t, privateCIDRs, "privateCIDRs should be populated by init()")
	assert.Equal(t, 21, len(privateCIDRs), "expected 21 CIDR ranges in privateCIDRs")
}
