package carriers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapApaczkaStatus(t *testing.T) {
	tests := []struct {
		input   string
		wantOMS string
		wantOK  bool
	}{
		// pending group
		{"NEW", "pending", true},
		{"CREATED", "pending", true},
		{"WAITING", "pending", true},
		// in_transit group
		{"SENT", "in_transit", true},
		{"IN_TRANSIT", "in_transit", true},
		{"PICKED_UP", "in_transit", true},
		// out_for_delivery
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		// delivered
		{"DELIVERED", "delivered", true},
		// returned group
		{"RETURNED", "returned", true},
		{"RETURN", "returned", true},
		// cancelled group
		{"CANCELLED", "cancelled", true},
		{"CANCELED", "cancelled", true},
		// case-insensitive + whitespace
		{"new", "pending", true},
		{" Delivered ", "delivered", true},
		// unknown
		{"UNKNOWN_STATUS", "", false},
		{"", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := mapApaczkaStatus(tc.input)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantOMS, got)
		})
	}
}
