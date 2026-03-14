// Package carriers provides implementations of the carrier shipping provider interface.
package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	dhlsdk "github.com/openoms-org/openoms/packages/dhl-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("dhl", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewDHLProvider(credentials, settings)
	})
}

// DHLCredentials is the JSON structure stored in encrypted integration credentials.
type DHLCredentials struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AccountNumber string `json:"account_number"`
	Sandbox       bool   `json:"sandbox,omitempty"`
}

// DHLProvider implements integration.CarrierProvider for DHL Parcel Poland.
type DHLProvider struct {
	client *dhlsdk.Client
	logger *slog.Logger
}

// NewDHLProvider creates a DHL CarrierProvider from encrypted credentials.
func NewDHLProvider(credentials json.RawMessage, _ json.RawMessage) (*DHLProvider, error) {
	var creds DHLCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("dhl: parse credentials: %w", err)
	}

	if creds.Sandbox {
		return nil, fmt.Errorf("dhl: DHL24 WebAPI2 does not support sandbox mode — remove the sandbox flag from credentials or use DHL test account numbers")
	}

	client := dhlsdk.NewClient(creds.Username, creds.Password, creds.AccountNumber)

	return &DHLProvider{
		client: client,
		logger: slog.Default().With("provider", "dhl"),
	}, nil
}

// ProviderName returns the carrier provider identifier.
func (p *DHLProvider) ProviderName() string { return "dhl" }

// CreateShipment creates a DHL shipment and returns the response with tracking info.
func (p *DHLProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcType, err := mapDHLServiceType(req.ServiceType)
	if err != nil {
		return nil, err
	}

	recvStreet, recvHouseNo := splitStreetHouseNo(req.Receiver.Street)

	dhlReq := &dhlsdk.CreateShipmentRequest{
		ShipperAccount: p.client.AccountNumber(),
		Receiver: dhlsdk.Receiver{
			Name:       req.Receiver.Name,
			Email:      req.Receiver.Email,
			Phone:      req.Receiver.Phone,
			Street:     recvStreet,
			HouseNo:    recvHouseNo,
			City:       req.Receiver.City,
			PostalCode: req.Receiver.PostalCode,
			Country:    req.Receiver.Country,
		},
		Piece: dhlsdk.Piece{
			Weight: req.Parcel.WeightKg,
			Width:  req.Parcel.WidthCm,
			Height: req.Parcel.HeightCm,
			Length: req.Parcel.DepthCm,
		},
		ServiceType: svcType,
		Reference:   req.Reference,
	}

	// Map shipper address — required by DHL24 SOAP API.
	if req.Shipper == nil {
		return nil, fmt.Errorf("dhl: shipper address is required (configure warehouse address or company settings)")
	}
	shipStreet, shipHouseNo := splitStreetHouseNo(req.Shipper.Street)
	dhlReq.Shipper = dhlsdk.Shipper{
		Name:       req.Shipper.Name,
		Email:      req.Shipper.Email,
		Phone:      req.Shipper.Phone,
		Street:     shipStreet,
		HouseNo:    shipHouseNo,
		City:       req.Shipper.City,
		PostalCode: req.Shipper.PostalCode,
		Country:    req.Shipper.Country,
	}

	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		dhlReq.COD = &dhlsdk.COD{
			Amount:   req.CODAmount,
			Currency: currency,
		}
	}

	if req.InsuredValue > 0 {
		dhlReq.Insurance = &dhlsdk.Money{
			Amount:   req.InsuredValue,
			Currency: "PLN",
		}
	}

	resp, err := p.client.Shipments.Create(ctx, dhlReq)
	if err != nil {
		return nil, fmt.Errorf("dhl: create shipment: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     resp.ShipmentID,
		TrackingNumber: resp.TrackingNumber,
		Status:         resp.Status,
		LabelURL:       resp.LabelURL,
	}, nil
}

// mapDHLServiceType translates frontend service type names to DHL24 SOAP serviceType codes.
// Codes sourced from DHL24 WebAPI2 (dhl24.com.pl/webapi2):
//   - AH: Domestic parcel (standard DHL Parcel)
//   - DR: Domestic courier (DHL Courier / next-day delivery)
//
// Returns error for unknown service types to prevent sending arbitrary strings to DHL24 SOAP API.
func mapDHLServiceType(serviceType string) (string, error) {
	switch serviceType {
	case "dhl_parcel", "AH":
		return "AH", nil
	case "dhl_courier", "DR":
		return "DR", nil
	case "":
		return "AH", nil
	default:
		return "", fmt.Errorf("dhl: unknown service type %q — valid types: dhl_parcel, dhl_courier", serviceType)
	}
}

// splitStreetHouseNo splits a Polish address string into street name and house number.
// DHL24 SOAP API requires separate <street> and <houseNo> fields.
// Examples:
//
//	"Marszalkowska 10"         → ("Marszalkowska", "10")
//	"ul. Krakowska 15A"        → ("ul. Krakowska", "15A")
//	"Aleje Jerozolimskie 100/2" → ("Aleje Jerozolimskie", "100/2")
//	"Krakowska"                → ("Krakowska", "")
func splitStreetHouseNo(address string) (street, houseNo string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", ""
	}

	// Find the last space-separated token that starts with a digit.
	// Polish addresses put the house number at the end.
	parts := strings.Fields(address)
	if len(parts) <= 1 {
		return address, ""
	}

	last := parts[len(parts)-1]
	if len(last) > 0 && unicode.IsDigit(rune(last[0])) {
		return strings.Join(parts[:len(parts)-1], " "), last
	}

	return address, ""
}

// GetLabel downloads the shipping label for the given DHL shipment.
func (p *DHLProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

// GetTracking returns tracking events for the given DHL shipment.
func (p *DHLProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("dhl: get tracking: %w", err)
	}

	events := make([]integration.TrackingEvent, 0, len(resp.Events))
	for _, ev := range resp.Events {
		events = append(events, integration.TrackingEvent{
			Status:    ev.Status,
			Location:  ev.Location,
			Timestamp: ev.Timestamp,
			Details:   ev.Details,
		})
	}

	return events, nil
}

// CancelShipment cancels a DHL shipment by its external ID.
func (p *DHLProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

// MapStatus maps a DHL carrier status to the internal shipment status.
func (p *DHLProvider) MapStatus(carrierStatus string) (string, bool) {
	return dhlsdk.MapStatus(carrierStatus)
}

// GetRates returns estimated shipping rates for DHL.
func (p *DHLProvider) GetRates(_ context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	// DHL Poland domestic rate estimation based on weight tiers.
	// Real integration would use the DHL Rating API.
	domestic := (req.FromCountry == "" || req.FromCountry == "PL") &&
		(req.ToCountry == "" || req.ToCountry == "PL")

	if !domestic {
		// International rates are not hardcoded — return empty for now.
		return nil, nil
	}

	w := req.Weight
	var rates []integration.Rate

	if w <= 1 {
		price := 12.50
		if req.COD > 0 {
			price += 5.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "DHL",
			CarrierCode:   "dhl",
			ServiceName:   "DHL Parcel do 1 kg",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}
	if w <= 5 {
		price := 14.50
		if req.COD > 0 {
			price += 5.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "DHL",
			CarrierCode:   "dhl",
			ServiceName:   "DHL Parcel do 5 kg",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}
	if w <= 10 {
		price := 16.50
		if req.COD > 0 {
			price += 5.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "DHL",
			CarrierCode:   "dhl",
			ServiceName:   "DHL Parcel do 10 kg",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}
	if w <= 20 {
		price := 18.50
		if req.COD > 0 {
			price += 5.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "DHL",
			CarrierCode:   "dhl",
			ServiceName:   "DHL Parcel do 20 kg",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}
	if w <= 31.5 {
		price := 21.00
		if req.COD > 0 {
			price += 5.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "DHL",
			CarrierCode:   "dhl",
			ServiceName:   "DHL Parcel do 31.5 kg",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}

	return rates, nil
}

// SupportsPickupPoints reports that DHL does not support pickup point delivery.
func (p *DHLProvider) SupportsPickupPoints() bool { return false }

// SearchPickupPoints is not supported by DHL.
func (p *DHLProvider) SearchPickupPoints(_ context.Context, _ string) ([]integration.PickupPoint, error) {
	return nil, fmt.Errorf("dhl: pickup point search not supported")
}
