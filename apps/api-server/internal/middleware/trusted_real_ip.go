package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// TrustedRealIP updates r.RemoteAddr from forwarding headers only when the
// immediate peer is an explicitly trusted proxy. With no trusted proxies it is
// a no-op, so spoofed X-Forwarded-For headers cannot affect rate-limit keys.
func TrustedRealIP(trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(trustedProxies) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			peerIP, ok := remoteAddrIP(r.RemoteAddr)
			if !ok || !isTrustedProxy(peerIP, trustedProxies) {
				next.ServeHTTP(w, r)
				return
			}

			if clientIP, ok := forwardedClientIP(r.Header.Get("X-Forwarded-For"), trustedProxies); ok {
				r.RemoteAddr = clientIP.String()
				next.ServeHTTP(w, r)
				return
			}

			if clientIP, ok := headerIP(r.Header.Get("X-Real-IP")); ok {
				r.RemoteAddr = clientIP.String()
			}

			next.ServeHTTP(w, r)
		})
	}
}

func remoteAddrIP(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return headerIP(host)
}

func forwardedClientIP(value string, trustedProxies []netip.Prefix) (netip.Addr, bool) {
	chain := make([]netip.Addr, 0, 4)
	for part := range strings.SplitSeq(value, ",") {
		if ip, ok := headerIP(part); ok {
			chain = append(chain, ip)
		}
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if !isTrustedProxy(chain[i], trustedProxies) {
			return chain[i], true
		}
	}
	return netip.Addr{}, false
}

func headerIP(value string) (netip.Addr, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return netip.Addr{}, false
	}
	ip, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

func isTrustedProxy(ip netip.Addr, trustedProxies []netip.Prefix) bool {
	ip = ip.Unmap()
	for _, trusted := range trustedProxies {
		if trusted.Contains(ip) {
			return true
		}
	}
	return false
}
