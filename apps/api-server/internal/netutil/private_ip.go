// Package netutil provides shared network utility functions.
package netutil

import (
	"net"
	"net/url"
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
