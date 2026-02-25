package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDropshipTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		// Valid transitions from "pending"
		{"pending to sent", "pending", "sent", true},
		{"pending to cancelled", "pending", "cancelled", true},

		// Valid transitions from "sent"
		{"sent to confirmed", "sent", "confirmed", true},
		{"sent to shipped", "sent", "shipped", true},
		{"sent to cancelled", "sent", "cancelled", true},

		// Valid transitions from "confirmed"
		{"confirmed to shipped", "confirmed", "shipped", true},
		{"confirmed to cancelled", "confirmed", "cancelled", true},

		// Valid transitions from "shipped"
		{"shipped to delivered", "shipped", "delivered", true},
		{"shipped to cancelled", "shipped", "cancelled", true},

		// Terminal states — no transitions allowed
		{"delivered is terminal", "delivered", "pending", false},
		{"delivered to sent", "delivered", "sent", false},
		{"delivered to confirmed", "delivered", "confirmed", false},
		{"delivered to shipped", "delivered", "shipped", false},
		{"delivered to cancelled", "delivered", "cancelled", false},
		{"delivered to delivered", "delivered", "delivered", false},

		{"cancelled is terminal", "cancelled", "pending", false},
		{"cancelled to sent", "cancelled", "sent", false},
		{"cancelled to confirmed", "cancelled", "confirmed", false},
		{"cancelled to shipped", "cancelled", "shipped", false},
		{"cancelled to delivered", "cancelled", "delivered", false},
		{"cancelled to cancelled", "cancelled", "cancelled", false},

		// Invalid transitions — skipping states
		{"pending to confirmed skips sent", "pending", "confirmed", false},
		{"pending to shipped skips multiple", "pending", "shipped", false},
		{"pending to delivered skips all", "pending", "delivered", false},
		{"sent to delivered skips shipped", "sent", "delivered", false},
		{"confirmed to delivered skips shipped", "confirmed", "delivered", false},

		// Backward transitions
		{"sent to pending backward", "sent", "pending", false},
		{"confirmed to sent backward", "confirmed", "sent", false},
		{"confirmed to pending backward", "confirmed", "pending", false},
		{"shipped to sent backward", "shipped", "sent", false},
		{"shipped to confirmed backward", "shipped", "confirmed", false},
		{"shipped to pending backward", "shipped", "pending", false},

		// Self transitions
		{"pending to pending", "pending", "pending", false},
		{"sent to sent", "sent", "sent", false},
		{"confirmed to confirmed", "confirmed", "confirmed", false},
		{"shipped to shipped", "shipped", "shipped", false},

		// Empty strings
		{"empty from", "", "sent", false},
		{"empty to", "pending", "", false},
		{"both empty", "", "", false},

		// Unknown states
		{"unknown from state", "unknown", "sent", false},
		{"unknown to state", "pending", "unknown", false},
		{"both unknown", "foo", "bar", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidDropshipTransition(tt.from, tt.to))
		})
	}
}
