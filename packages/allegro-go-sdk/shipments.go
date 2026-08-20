package allegro

import (
	"context"
	"encoding/json"
	"fmt"
)

// UnmarshalJSON accepts Allegro's official DeliveryServicesDto.services key
// and the older deliveryServices envelope as a fallback.
func (l *DeliveryServiceList) UnmarshalJSON(data []byte) error {
	var raw struct {
		Services         []DeliveryService `json:"services"`
		DeliveryServices []DeliveryService `json:"deliveryServices"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Services != nil {
		l.DeliveryServices = raw.Services
		return nil
	}
	l.DeliveryServices = raw.DeliveryServices
	return nil
}

// UnmarshalJSON accepts a bare string id or official DeliveryServiceIdDto
// ({deliveryMethodId, credentialsId}).
func (s *DeliveryService) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID        json.RawMessage `json:"id"`
		Name      string          `json:"name"`
		CarrierID string          `json:"carrierId"`
		Owner     string          `json:"owner"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	id, err := unmarshalDeliveryServiceID(raw.ID)
	if err != nil {
		return err
	}
	s.ID = id
	s.Name = raw.Name
	s.CarrierID = raw.CarrierID
	s.Owner = raw.Owner
	return nil
}

func unmarshalDeliveryServiceID(data json.RawMessage) (string, error) {
	if len(data) == 0 || string(data) == "null" {
		return "", nil
	}
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		return id, nil
	}
	var obj struct {
		DeliveryMethodID string `json:"deliveryMethodId"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", err
	}
	return obj.DeliveryMethodID, nil
}

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
	return s.client.doRaw(ctx, "POST", "/shipment-management/label", body, "application/pdf")
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
	return s.client.doRaw(ctx, "POST", "/shipment-management/protocol", body, "application/pdf")
}
