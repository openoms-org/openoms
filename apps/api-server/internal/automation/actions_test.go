package automation

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestDefaultActionExecutor_ActivateListing_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No listing deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "activate_listing"}, Event{
		Type:       "product.stock_restored",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "activate_listing with nil deps should not error")
}

func TestDefaultActionExecutor_ActivateListing_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetListingActivatorDeps(&ListingActivatorDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "activate_listing"}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "activate_listing with order entity should error")
	assert.Contains(t, err.Error(), "only supports product entities")
}

func TestDefaultActionExecutor_DeactivateListing_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No listing deactivator deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "deactivate_listing"}, Event{
		Type:       "product.out_of_stock",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "deactivate_listing with nil deps should not error")
}

func TestDefaultActionExecutor_DeactivateListing_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetListingDeactivatorDeps(&ListingDeactivatorDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "deactivate_listing"}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "deactivate_listing with order entity should error")
	assert.Contains(t, err.Error(), "only supports product entities")
}

func TestDefaultActionExecutor_UnknownAction(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "nonexistent"}, Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestDefaultActionExecutor_SetStatus_NilTransitioner(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "set_status",
		Params: map[string]any{"status": "confirmed"},
	}, Event{
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	// With nil transitioner, should return nil (skip gracefully)
	assert.NoError(t, err)
}

func TestDefaultActionExecutor_WebhookMissingURL(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "webhook",
		Params: map[string]any{},
	}, Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing url")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No marketplace message deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "send_marketplace_message with nil deps should not error")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_NilPool(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// Deps set but Pool is nil — should skip gracefully.
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "send_marketplace_message with nil Pool should skip gracefully")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// Need a non-nil Pool to pass the deps check, but it won't actually be used
	// because the entity type check comes first.
	executor.pool = &pgxpool.Pool{} // Set the field directly (package-internal test)
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "product.stock_restored",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "send_marketplace_message with product entity should error")
	assert.Contains(t, err.Error(), "only supports order entities")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_MissingTemplateID(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'template_id'")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_InvalidTemplateID(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": "not-a-uuid"},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template_id")
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
	}{
		{
			name:     "simple substitution",
			template: "Hello {{customer_name}}, your order {{order_id}} is ready.",
			vars:     map[string]string{"customer_name": "Jan", "order_id": "ORD-123"},
			expected: "Hello Jan, your order ORD-123 is ready.",
		},
		{
			name:     "no variables",
			template: "Hello, your order is ready.",
			vars:     map[string]string{"customer_name": "Jan"},
			expected: "Hello, your order is ready.",
		},
		{
			name:     "missing variable stays",
			template: "Hello {{customer_name}}, tracking: {{tracking_number}}.",
			vars:     map[string]string{"customer_name": "Jan"},
			expected: "Hello Jan, tracking: {{tracking_number}}.",
		},
		{
			name:     "empty vars map",
			template: "Hello {{customer_name}}!",
			vars:     map[string]string{},
			expected: "Hello {{customer_name}}!",
		},
		{
			name:     "multiple occurrences",
			template: "{{name}} ordered. Dear {{name}}, thank you.",
			vars:     map[string]string{"name": "Anna"},
			expected: "Anna ordered. Dear Anna, thank you.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVariables(tt.template, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}
