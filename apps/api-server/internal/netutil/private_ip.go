// Package netutil provides shared network utility functions.
package netutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// privateCIDRs holds parsed private/internal IP ranges, initialized once at package init.
var privateCIDRs []*net.IPNet

func init() {
	privateRanges := []string{
		// RFC 1918 — private networks
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		// Loopback & link-local
		"127.0.0.0/8",
		"169.254.0.0/16",
		// Special-purpose
		"0.0.0.0/8",          // RFC 1122 — "this" network
		"100.64.0.0/10",      // RFC 6598 — Shared/CGN
		"192.0.0.0/24",       // RFC 6890 — IETF Protocol Assignments
		"192.0.2.0/24",       // RFC 5737 — TEST-NET-1
		"198.18.0.0/15",      // RFC 2544 — Benchmarking
		"198.51.100.0/24",    // RFC 5737 — TEST-NET-2
		"203.0.113.0/24",     // RFC 5737 — TEST-NET-3
		"224.0.0.0/4",        // RFC 5771 — Multicast
		"240.0.0.0/4",        // RFC 1112 — Reserved
		"255.255.255.255/32", // Limited Broadcast
		// IPv6 special-purpose
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"2001:db8::/32", // RFC 3849 — Documentation
		"::/128",        // IPv6 unspecified address
		"ff00::/8",      // IPv6 multicast
	}
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		privateCIDRs = append(privateCIDRs, ipNet)
	}
}

// IsPrivateIP checks whether the given IP address belongs to a private/internal range.
func IsPrivateIP(ip net.IP) bool {
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// NoPrivateDialer returns a DialContext function that refuses to connect to private IP addresses.
// This prevents SSRF TOCTOU attacks by checking the resolved IP at connect time (atomically).
func NoPrivateDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}

		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed: %w", err)
		}

		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip != nil && IsPrivateIP(ip) {
				return nil, fmt.Errorf("connection to private IP %s rejected", ipStr)
			}
		}

		// Connect to the first resolved IP to avoid TOCTOU
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
	}
}

// SafeHTTPClient returns an HTTP client that refuses to connect to private IP addresses.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: NoPrivateDialer(),
		},
	}
}

// IsPrivateURL checks whether a URL resolves to a private/internal IP address.
func IsPrivateURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return true // reject unparseable URLs
	}

	hostname := u.Hostname()
	if hostname == "" {
		return true
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return true // reject unresolvable hostnames
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if IsPrivateIP(ip) {
			return true
		}
	}

	return false
}
