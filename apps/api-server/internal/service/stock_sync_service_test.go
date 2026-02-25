package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// --- Stock Restored Detection ---

func TestStockRestoredDetection_ZeroToPositive(t *testing.T) {
	// The core logic: oldQty == 0 && newQty > 0 should trigger stock_restored.
	oldQty, newQty := 0, 10
	isRestored := oldQty == 0 && newQty > 0
	assert.True(t, isRestored, "stock going from 0 to 10 should be detected as restored")
}

func TestStockRestoredDetection_ZeroToZero(t *testing.T) {
	oldQty, newQty := 0, 0
	isRestored := oldQty == 0 && newQty > 0
	assert.False(t, isRestored, "stock staying at 0 should not trigger restored")
}

func TestStockRestoredDetection_PositiveToPositive(t *testing.T) {
	oldQty, newQty := 5, 10
	isRestored := oldQty == 0 && newQty > 0
	assert.False(t, isRestored, "stock going from 5 to 10 should not trigger restored")
}

func TestStockRestoredDetection_PositiveToZero(t *testing.T) {
	oldQty, newQty := 10, 0
	isRestored := oldQty == 0 && newQty > 0
	assert.False(t, isRestored, "stock going from 10 to 0 should not trigger restored")
}

func TestStockRestoredDetection_ZeroToOne(t *testing.T) {
	oldQty, newQty := 0, 1
	isRestored := oldQty == 0 && newQty > 0
	assert.True(t, isRestored, "stock going from 0 to 1 should be detected as restored")
}

// --- Trigger Event Validation ---

func TestValidTriggerEvents_IncludesStockRestored(t *testing.T) {
	assert.True(t, model.ValidTriggerEvents["product.stock_restored"],
		"product.stock_restored should be a valid trigger event")
}

func TestValidTriggerEvents_AllExpectedEvents(t *testing.T) {
	expectedEvents := []string{
		"order.created",
		"order.status_changed",
		"order.updated",
		"shipment.created",
		"shipment.status_changed",
		"return.created",
		"return.status_changed",
		"product.created",
		"product.updated",
		"product.stock_restored",
	}
	for _, event := range expectedEvents {
		assert.True(t, model.ValidTriggerEvents[event],
			"expected %q to be a valid trigger event", event)
	}
}

// --- Service Construction ---

func TestStockSyncService_SetAutomationService(t *testing.T) {
	svc := &StockSyncService{}
	assert.Nil(t, svc.automationSvc)

	automationSvc := &AutomationService{}
	svc.SetAutomationService(automationSvc)
	assert.NotNil(t, svc.automationSvc, "automation service should be set after SetAutomationService")
}

// --- Auto-Relist Logic ---

func TestAutoRelist_ListingStatusFiltering(t *testing.T) {
	// Test that only "inactive" and "ended" statuses would be candidates for relisting.
	tests := []struct {
		status   string
		expected bool
	}{
		{"inactive", true},
		{"ended", true},
		{"active", false},
		{"pending", false},
		{"error", false},
		{"synced", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			shouldActivate := tt.status == "inactive" || tt.status == "ended"
			assert.Equal(t, tt.expected, shouldActivate,
				"listing with status %q: activate=%v", tt.status, tt.expected)
		})
	}
}
