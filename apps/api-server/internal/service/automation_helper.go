package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
)

// FireAutomationEvent is a shared helper that fires an automation event if the
// automation service is available. This deduplicates the identical pattern used
// by OrderService, ShipmentService, ProductService, and ReturnService.
func FireAutomationEvent(automationSvc *AutomationService, tenantID uuid.UUID, entityType, eventType string, entityID uuid.UUID, data map[string]any) {
	if automationSvc != nil {
		automationSvc.ProcessEvent(context.Background(), automation.Event{
			Type:       eventType,
			TenantID:   tenantID,
			EntityType: entityType,
			EntityID:   entityID,
			Data:       data,
		})
	}
}
