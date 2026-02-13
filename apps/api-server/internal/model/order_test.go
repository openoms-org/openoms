package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrderRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateOrderRequest
		wantErr string
	}{
		{
			name:    "missing customer_name",
			req:     CreateOrderRequest{CustomerName: ""},
			wantErr: "customer_name is required",
		},
		{
			name:    "whitespace-only customer_name",
			req:     CreateOrderRequest{CustomerName: "   "},
			wantErr: "customer_name is required",
		},
		{
			name:    "negative total_amount",
			req:     CreateOrderRequest{CustomerName: "Jan", TotalAmount: -10},
			wantErr: "total_amount must be non-negative",
		},
		{
			name: "valid with defaults",
			req:  CreateOrderRequest{CustomerName: "Jan Kowalski", TotalAmount: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				// Check defaults are set
				assert.Equal(t, "manual", tt.req.Source)
				assert.Equal(t, "PLN", tt.req.Currency)
			}
		})
	}
}

func TestUpdateOrderRequest_Validate(t *testing.T) {
	t.Run("empty update", func(t *testing.T) {
		req := UpdateOrderRequest{}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field must be provided")
	})

	t.Run("negative total_amount", func(t *testing.T) {
		neg := -5.0
		req := UpdateOrderRequest{TotalAmount: &neg}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "total_amount must be non-negative")
	})

	t.Run("valid partial update", func(t *testing.T) {
		name := "Updated Name"
		req := UpdateOrderRequest{CustomerName: &name}
		require.NoError(t, req.Validate())
	})
}

func TestStatusTransitionRequest_Validate(t *testing.T) {
	t.Run("empty status", func(t *testing.T) {
		req := StatusTransitionRequest{Status: ""}
		require.Error(t, req.Validate())
	})

	t.Run("whitespace status", func(t *testing.T) {
		req := StatusTransitionRequest{Status: "   "}
		require.Error(t, req.Validate())
	})

	t.Run("valid status", func(t *testing.T) {
		req := StatusTransitionRequest{Status: "confirmed"}
		require.NoError(t, req.Validate())
	})
}

func TestBulkStatusTransitionRequest_Validate(t *testing.T) {
	t.Run("empty order_ids", func(t *testing.T) {
		req := BulkStatusTransitionRequest{Status: "confirmed"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one order_id")
	})

	t.Run("too many order_ids", func(t *testing.T) {
		ids := make([]uuid.UUID, 101)
		req := BulkStatusTransitionRequest{OrderIDs: ids, Status: "confirmed"}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maximum 100")
	})

	t.Run("empty status", func(t *testing.T) {
		ids := []uuid.UUID{uuid.New()}
		req := BulkStatusTransitionRequest{OrderIDs: ids, Status: ""}
		err := req.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status is required")
	})
}

func TestOrderStatusConfig_IsValidStatus(t *testing.T) {
	cfg := DefaultOrderStatusConfig()

	assert.True(t, cfg.IsValidStatus("new"))
	assert.True(t, cfg.IsValidStatus("confirmed"))
	assert.True(t, cfg.IsValidStatus("delivered"))
	assert.True(t, cfg.IsValidStatus("refunded"))
	assert.False(t, cfg.IsValidStatus("bogus"))
	assert.False(t, cfg.IsValidStatus(""))
}

func TestOrderStatusConfig_CanTransition(t *testing.T) {
	cfg := DefaultOrderStatusConfig()

	// Valid transitions
	assert.True(t, cfg.CanTransition("new", "confirmed"))
	assert.True(t, cfg.CanTransition("new", "cancelled"))
	assert.True(t, cfg.CanTransition("new", "on_hold"))
	assert.True(t, cfg.CanTransition("confirmed", "processing"))
	assert.True(t, cfg.CanTransition("shipped", "in_transit"))
	assert.True(t, cfg.CanTransition("delivered", "completed"))
	assert.True(t, cfg.CanTransition("completed", "refunded"))

	// Invalid transitions
	assert.False(t, cfg.CanTransition("new", "delivered"))
	assert.False(t, cfg.CanTransition("new", "shipped"))
	assert.False(t, cfg.CanTransition("completed", "new"))
	assert.False(t, cfg.CanTransition("refunded", "new"))
	assert.False(t, cfg.CanTransition("refunded", "confirmed"))

	// Unknown source status
	assert.False(t, cfg.CanTransition("nonexistent", "confirmed"))
}

func TestOrderStatusConfig_GetStatusDef(t *testing.T) {
	cfg := DefaultOrderStatusConfig()

	def := cfg.GetStatusDef("new")
	require.NotNil(t, def)
	assert.Equal(t, "Now", def.Label)
	assert.Equal(t, "blue", def.Color)

	assert.Nil(t, cfg.GetStatusDef("nonexistent"))
}
