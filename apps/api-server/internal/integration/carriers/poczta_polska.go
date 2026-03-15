package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pocztasdk "github.com/openoms-org/openoms/packages/poczta-polska-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("poczta_polska", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewPocztaPolskaProvider(credentials, settings)
	})
}

// PocztaPolskaCredentials is the JSON structure stored in encrypted integration credentials.
type PocztaPolskaCredentials struct {
	APIKey    string `json:"api_key"`
	PartnerID string `json:"partner_id"`
	Sandbox   bool   `json:"sandbox,omitempty"`
}

// PocztaPolskaProvider implements integration.CarrierProvider for Poczta Polska eNadawca.
type PocztaPolskaProvider struct {
	client *pocztasdk.Client
	logger *slog.Logger
}

// NewPocztaPolskaProvider creates a Poczta Polska CarrierProvider from encrypted credentials.
func NewPocztaPolskaProvider(credentials json.RawMessage, _ json.RawMessage) (*PocztaPolskaProvider, error) {
	var creds PocztaPolskaCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("poczta_polska: parse credentials: %w", err)
	}

	var opts []pocztasdk.Option
	if creds.Sandbox {
		opts = append(opts, pocztasdk.WithSandbox())
	}

	client := pocztasdk.NewClient(creds.APIKey, creds.PartnerID, opts...)

	return &PocztaPolskaProvider{
		client: client,
		logger: slog.Default().With("provider", "poczta_polska"),
	}, nil
}

// ProviderName returns the carrier provider identifier.
func (p *PocztaPolskaProvider) ProviderName() string { return "poczta_polska" }

// CreateShipment creates a Poczta Polska shipment and returns the response with tracking info.
func (p *PocztaPolskaProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcType := req.ServiceType
	if svcType == "" {
		svcType = "POCZTEX_KURIER_48" // Pocztex courier 48h
	}

	ppReq := &pocztasdk.CreateShipmentRequest{
		ServiceType: svcType,
		Receiver: pocztasdk.Receiver{
			Name:       req.Receiver.Name,
			Email:      req.Receiver.Email,
			Phone:      req.Receiver.Phone,
			Street:     req.Receiver.Street,
			City:       req.Receiver.City,
			PostalCode: req.Receiver.PostalCode,
			Country:    req.Receiver.Country,
		},
		Parcel: pocztasdk.Parcel{
			Weight: req.Parcel.WeightKg,
			Width:  req.Parcel.WidthCm,
			Height: req.Parcel.HeightCm,
			Length: req.Parcel.DepthCm,
		},
		Reference: req.Reference,
	}

	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		ppReq.COD = &pocztasdk.COD{
			Amount:   req.CODAmount,
			Currency: currency,
		}
	}

	if req.InsuredValue > 0 {
		ppReq.Insurance = &pocztasdk.Money{
			Amount:   req.InsuredValue,
			Currency: "PLN",
		}
	}

	resp, err := p.client.Shipments.Create(ctx, ppReq)
	if err != nil {
		return nil, fmt.Errorf("poczta_polska: create shipment: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     resp.ShipmentID,
		TrackingNumber: resp.TrackingNumber,
		Status:         resp.Status,
		LabelURL:       resp.LabelURL,
	}, nil
}

// GetLabel downloads the shipping label for the given Poczta Polska shipment.
func (p *PocztaPolskaProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

// GetTracking returns tracking events for the given Poczta Polska shipment.
func (p *PocztaPolskaProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("poczta_polska: get tracking: %w", err)
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

// CancelShipment cancels a Poczta Polska shipment by its external ID.
func (p *PocztaPolskaProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

// MapStatus maps a Poczta Polska carrier status to the internal shipment status.
func (p *PocztaPolskaProvider) MapStatus(carrierStatus string) (string, bool) {
	return pocztasdk.MapStatus(carrierStatus)
}

// GetRates returns estimated shipping rates for Poczta Polska.
func (p *PocztaPolskaProvider) GetRates(_ context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	// TODO: Implement real Poczta Polska eNadawca rate API integration.
	domestic := (req.FromCountry == "" || req.FromCountry == "PL") &&
		(req.ToCountry == "" || req.ToCountry == "PL")
	if !domestic {
		return nil, nil
	}

	w := req.Weight
	var rates []integration.Rate

	if w <= 20 {
		price := 14.00
		if w > 5 {
			price = 16.50
		}
		if w > 10 {
			price = 19.00
		}
		if req.COD > 0 {
			price += 5.50
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "Poczta Polska",
			CarrierCode:   "poczta_polska",
			ServiceName:   "Pocztex Kurier 48",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}

	return rates, nil
}

// SupportsPickupPoints reports that Poczta Polska does not support pickup point delivery.
func (p *PocztaPolskaProvider) SupportsPickupPoints() bool { return false }

// SearchPickupPoints is not supported by Poczta Polska.
func (p *PocztaPolskaProvider) SearchPickupPoints(_ context.Context, _ string) ([]integration.PickupPoint, error) {
	return nil, fmt.Errorf("poczta_polska: pickup point search not supported")
}
