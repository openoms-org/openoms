package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustPrefix(t *testing.T, raw string) netip.Prefix {
	t.Helper()
	prefix, err := netip.ParsePrefix(raw)
	require.NoError(t, err)
	return prefix
}

func captureRemoteAddr(t *testing.T, trusted []netip.Prefix, remoteAddr string, headers map[string]string) string {
	t.Helper()
	var got string
	handler := middleware.TrustedRealIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	handler.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestTrustedRealIP_IgnoresForwardedHeadersWhenNoTrustedProxiesConfigured(t *testing.T) {
	got := captureRemoteAddr(t, nil, "203.0.113.10:12345", map[string]string{
		"X-Forwarded-For": "198.51.100.20",
		"X-Real-IP":       "198.51.100.21",
	})

	assert.Equal(t, "203.0.113.10:12345", got)
}

func TestTrustedRealIP_IgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

	got := captureRemoteAddr(t, trusted, "203.0.113.10:12345", map[string]string{
		"X-Forwarded-For": "198.51.100.20",
		"X-Real-IP":       "198.51.100.21",
	})

	assert.Equal(t, "203.0.113.10:12345", got)
}

func TestTrustedRealIP_UsesXForwardedForFromTrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

	got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
		"X-Forwarded-For": "198.51.100.20, 10.42.0.25",
		"X-Real-IP":       "198.51.100.21",
	})

	assert.Equal(t, "198.51.100.20", got)
}

func TestTrustedRealIP_UsesXRealIPFallbackFromTrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

	got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
		"X-Real-IP": "198.51.100.21",
	})

	assert.Equal(t, "198.51.100.21", got)
}

func TestTrustedRealIP_IgnoresInvalidForwardedHeaders(t *testing.T) {
	trusted := []netip.Prefix{mustPrefix(t, "10.42.0.0/16")}

	got := captureRemoteAddr(t, trusted, "10.42.0.25:5678", map[string]string{
		"X-Forwarded-For": "not-an-ip",
		"X-Real-IP":       "also-not-an-ip",
	})

	assert.Equal(t, "10.42.0.25:5678", got)
}

func TestTrustedRealIP_SupportsIPv6TrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{mustPrefix(t, "fd00::/8")}

	got := captureRemoteAddr(t, trusted, "[fd00::10]:5678", map[string]string{
		"X-Forwarded-For": "2001:db8::5",
	})

	assert.Equal(t, "2001:db8::5", got)
}
