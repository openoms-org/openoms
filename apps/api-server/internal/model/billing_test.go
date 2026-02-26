package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckoutSessionRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CheckoutSessionRequest
		wantErr string
	}{
		{
			name:    "valid month",
			req:     CheckoutSessionRequest{PlanID: "standard", Interval: "month"},
			wantErr: "",
		},
		{
			name:    "valid year",
			req:     CheckoutSessionRequest{PlanID: "pro", Interval: "year"},
			wantErr: "",
		},
		{
			name:    "missing plan_id",
			req:     CheckoutSessionRequest{PlanID: "", Interval: "month"},
			wantErr: "plan_id is required",
		},
		{
			name:    "invalid interval",
			req:     CheckoutSessionRequest{PlanID: "standard", Interval: "weekly"},
			wantErr: "interval must be 'month' or 'year'",
		},
		{
			name:    "empty interval",
			req:     CheckoutSessionRequest{PlanID: "standard", Interval: ""},
			wantErr: "interval must be 'month' or 'year'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
