package allegro

import (
	"context"
	"fmt"
)

// ShipmentManagementService handles communication with the "Wysyłam z Allegro"
// shipment management endpoints.
type ShipmentManagementService struct {
	client *Client
}

// ListDeliveryServices retrieves available delivery services for shipment management.
func (s *ShipmentManagementService) ListDeliveryServices(ctx context.Context) ([]DeliveryService, error) {
	var result DeliveryServiceList
	if err := s.client.do(ctx, "GET", "/shipment-management/delivery-services", nil, &result); err != nil {
		return nil, err
	}
	return result.DeliveryServices, nil
}

// CreateShipment creates a new managed shipment via Allegro's shipment management.
func (s *ShipmentManagementService) CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (*CreateShipmentResponse, error) {
	var result CreateShipmentResponse
	if err := s.client.do(ctx, "POST", "/shipment-management/shipments/create-commands", cmd, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetShipment retrieves a managed shipment by ID.
func (s *ShipmentManagementService) GetShipment(ctx context.Context, shipmentID string) (*ManagedShipment, error) {
	var result ManagedShipment
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/shipment-management/shipments/%s", shipmentID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLabel generates a shipping label PDF for the given shipment IDs.
// Returns raw PDF bytes.
func (s *ShipmentManagementService) GetLabel(ctx context.Context, shipmentIDs []string) ([]byte, error) {
	body := map[string]any{
		"shipmentIds": shipmentIDs,
	}
	return s.client.doRaw(ctx, "POST", "/shipment-management/label", body)
}

// CancelShipment cancels one or more managed shipments.
func (s *ShipmentManagementService) CancelShipment(ctx context.Context, shipmentIDs []string) error {
	body := map[string]any{
		"shipmentIds": shipmentIDs,
	}
	return s.client.do(ctx, "POST", "/shipment-management/shipments/cancel-commands", body, nil)
}

// GetPickupProposals retrieves available pickup proposals for managed shipments.
func (s *ShipmentManagementService) GetPickupProposals(ctx context.Context, req PickupProposalRequest) ([]PickupProposal, error) {
	var result PickupProposalList
	if err := s.client.do(ctx, "POST", "/shipment-management/pickup-proposals", req, &result); err != nil {
		return nil, err
	}
	return result.Proposals, nil
}

// SchedulePickup schedules a courier pickup for managed shipments.
func (s *ShipmentManagementService) SchedulePickup(ctx context.Context, cmd SchedulePickupCommand) error {
	return s.client.do(ctx, "POST", "/shipment-management/pickups/create-commands", cmd, nil)
}

// GenerateProtocol generates a dispatch protocol PDF for the given shipment IDs.
// Returns raw PDF bytes.
func (s *ShipmentManagementService) GenerateProtocol(ctx context.Context, shipmentIDs []string) ([]byte, error) {
	body := map[string]any{
		"shipmentIds": shipmentIDs,
	}
	return s.client.doRaw(ctx, "POST", "/shipment-management/protocol", body)
}
