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
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
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
