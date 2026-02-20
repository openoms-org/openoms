package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCookieDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"subdomain returns dot-prefixed root", "https://app.openoms.org", ".openoms.org"},
		{"deep subdomain returns dot-prefixed root", "https://admin.app.openoms.org", ".openoms.org"},
		{"root domain returns dot-prefixed", "https://openoms.org", ".openoms.org"},
		{"localhost returns empty", "http://localhost:3000", ""},
		{"127.0.0.1 returns empty", "http://127.0.0.1:8080", ""},
		{"IPv6 loopback returns empty", "http://[::1]:3000", ""},
		{"invalid URL returns empty", "://not-a-url", ""},
		{"empty string returns empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractCookieDomain(tt.input))
		})
	}
}
