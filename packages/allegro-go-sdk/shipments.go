package allegro

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
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

// UnmarshalJSON accepts official postalCode and the older zipCode key.
func (a *ShipmentAddress) UnmarshalJSON(data []byte) error {
	type alias ShipmentAddress
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*a = ShipmentAddress(raw)
	if a.PostalCode == "" {
		a.PostalCode = a.ZipCode
	}
	return nil
}

// MarshalJSON sends official postalCode and omits zipCode.
func (a ShipmentAddress) MarshalJSON() ([]byte, error) {
	postal := a.PostalCode
	if postal == "" {
		postal = a.ZipCode
	}
	type out struct {
		Name         string `json:"name,omitempty"`
		Company      string `json:"company,omitempty"`
		Street       string `json:"street"`
		StreetNumber string `json:"streetNumber,omitempty"`
		City         string `json:"city"`
		PostalCode   string `json:"postalCode,omitempty"`
		State        string `json:"state,omitempty"`
		CountryCode  string `json:"countryCode"`
		Phone        string `json:"phone,omitempty"`
		Email        string `json:"email,omitempty"`
		Point        string `json:"point,omitempty"`
	}
	return json.Marshal(out{
		Name:         a.Name,
		Company:      a.Company,
		Street:       a.Street,
		StreetNumber: a.StreetNumber,
		City:         a.City,
		PostalCode:   postal,
		State:        a.State,
		CountryCode:  a.CountryCode,
		Phone:        a.Phone,
		Email:        a.Email,
		Point:        a.Point,
	})
}

// UnmarshalJSON accepts a JSON number or a numeric string (official weight sample).
func (d *Dimension) UnmarshalJSON(data []byte) error {
	var raw struct {
		Value json.RawMessage `json:"value"`
		Unit  string          `json:"unit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.Unit = raw.Unit
	if len(raw.Value) == 0 || string(raw.Value) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.Value, &d.Value); err == nil {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw.Value, &s); err != nil {
		return err
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	d.Value = v
	return nil
}

const (
	createCommandPollAttempts = 20
	createCommandPollInterval = 200 * time.Millisecond
)

// ListDeliveryServices retrieves available delivery services for shipment management.
func (s *ShipmentManagementService) ListDeliveryServices(ctx context.Context) ([]DeliveryService, error) {
	var result DeliveryServiceList
	if err := s.client.do(ctx, "GET", "/shipment-management/delivery-services", nil, &result); err != nil {
		return nil, err
	}
	return result.DeliveryServices, nil
}

// GetDeliveryProposals returns the official prefilled create-commands body for an order.
func (s *ShipmentManagementService) GetDeliveryProposals(ctx context.Context, orderID string) (*DeliveryProposals, error) {
	var result DeliveryProposals
	path := fmt.Sprintf("/shipment-management/delivery-proposals/%s", orderID)
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateShipment creates a managed shipment and waits for the async command shipmentId.
func (s *ShipmentManagementService) CreateShipment(ctx context.Context, cmd CreateShipmentCommand) (*CreateShipmentResponse, error) {
	var result CreateShipmentResponse
	if err := s.client.do(ctx, "POST", "/shipment-management/shipments/create-commands", cmd, &result); err != nil {
		return nil, err
	}
	if result.ShipmentID != "" {
		return &result, nil
	}
	commandID := result.CommandID
	if commandID == "" {
		commandID = cmd.CommandID
	}
	if commandID == "" {
		return &result, nil
	}
	return s.waitForCreateCommand(ctx, commandID)
}

func (s *ShipmentManagementService) waitForCreateCommand(ctx context.Context, commandID string) (*CreateShipmentResponse, error) {
	path := fmt.Sprintf("/shipment-management/shipments/create-commands/%s", commandID)
	var last ShipmentCommandStatus
	for attempt := range createCommandPollAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(createCommandPollInterval):
			}
		}
		if err := s.client.do(ctx, "GET", path, nil, &last); err != nil {
			return nil, err
		}
		switch strings.ToUpper(last.Status) {
		case "SUCCESS":
			return &CreateShipmentResponse{
				CommandID:  last.CommandID,
				ShipmentID: last.ShipmentID,
				Status:     last.Status,
			}, nil
		case "ERROR", "ERROR_LIMIT_EXCEEDED", "CANCELED", "CANCELLED":
			msg := last.Status
			if len(last.Errors) > 0 {
				if last.Errors[0].UserMessage != "" {
					msg = last.Errors[0].UserMessage
				} else if last.Errors[0].Message != "" {
					msg = last.Errors[0].Message
				}
			}
			return nil, fmt.Errorf("allegro: create-commands %s: %s", last.Status, msg)
		}
		if last.ShipmentID != "" {
			return &CreateShipmentResponse{
				CommandID:  last.CommandID,
				ShipmentID: last.ShipmentID,
				Status:     last.Status,
			}, nil
		}
	}
	return nil, fmt.Errorf("allegro: create-commands %s timed out waiting for shipmentId", commandID)
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
