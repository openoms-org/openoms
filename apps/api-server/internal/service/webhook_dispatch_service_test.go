package service

import (
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/stretchr/testify/assert"
)

func TestMatchesEvent(t *testing.T) {
	tests := []struct {
		name      string
		events    []string
		eventType string
		want      bool
	}{
		{"exact match", []string{"order.created"}, "order.created", true},
		{"wildcard", []string{"*"}, "order.created", true},
		{"no match", []string{"order.updated"}, "order.created", false},
		{"empty events", []string{}, "order.created", false},
		{"multiple events match", []string{"order.created", "order.updated"}, "order.updated", true},
		{"multiple events no match", []string{"order.created", "order.updated"}, "shipment.created", false},
		{"wildcard among others", []string{"order.created", "*"}, "anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesEvent(tt.events, tt.eventType))
		})
	}
}

func TestIsPrivateURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"empty url", "", true},
		{"unparseable url", "://bad", true},
		{"localhost", "http://localhost/webhook", true},
		{"loopback", "http://127.0.0.1/webhook", true},
		{"private 10.x", "http://10.0.0.1/webhook", true},
		{"private 192.168.x", "http://192.168.1.1/webhook", true},
		{"private 172.16.x", "http://172.16.0.1/webhook", true},
		{"public url", "https://example.com/webhook", false},
		{"no hostname", "http:///path", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutil.IsPrivateURL(tt.url)
			assert.Equal(t, tt.want, result)
		})
	}
}
