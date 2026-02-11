package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	orlensdk "github.com/openoms-org/openoms/packages/orlen-paczka-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("orlen_paczka", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewOrlenPaczkaProvider(credentials, settings)
	})
}

// OrlenPaczkaCredentials is the JSON structure stored in encrypted integration credentials.
type OrlenPaczkaCredentials struct {
	APIKey    string `json:"api_key"`
	PartnerID string `json:"partner_id"`
	Sandbox   bool   `json:"sandbox,omitempty"`
}

// OrlenPaczkaProvider implements integration.CarrierProvider for Orlen Paczka.
type OrlenPaczkaProvider struct {
	client *orlensdk.Client
	logger *slog.Logger
}

// NewOrlenPaczkaProvider creates an Orlen Paczka CarrierProvider from encrypted credentials.
func NewOrlenPaczkaProvider(credentials json.RawMessage, settings json.RawMessage) (*OrlenPaczkaProvider, error) {
	var creds OrlenPaczkaCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("orlen_paczka: parse credentials: %w", err)
	}

	var opts []orlensdk.Option
	if creds.Sandbox {
		opts = append(opts, orlensdk.WithSandbox())
	}

	client := orlensdk.NewClient(creds.APIKey, creds.PartnerID, opts...)

	return &OrlenPaczkaProvider{
		client: client,
		logger: slog.Default().With("provider", "orlen_paczka"),
	}, nil
}

func (p *OrlenPaczkaProvider) ProviderName() string { return "orlen_paczka" }

func (p *OrlenPaczkaProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	orlenReq := &orlensdk.CreateShipmentRequest{
		Receiver: orlensdk.Receiver{
			Name:  req.Receiver.Name,
			Email: req.Receiver.Email,
			Phone: req.Receiver.Phone,
		},
		Parcel: orlensdk.Parcel{
			SizeCode: req.Parcel.SizeCode,
			Weight:   req.Parcel.WeightKg,
			Width:    req.Parcel.WidthCm,
			Height:   req.Parcel.HeightCm,
			Length:   req.Parcel.DepthCm,
		},
		TargetPoint: req.TargetPoint,
		Reference:   req.Reference,
	}

	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		orlenReq.COD = &orlensdk.COD{
			Amount:   req.CODAmount,
			Currency: currency,
		}
	}

	if req.InsuredValue > 0 {
		orlenReq.Insurance = &orlensdk.Money{
			Amount:   req.InsuredValue,
			Currency: "PLN",
		}
	}

	resp, err := p.client.Shipments.Create(ctx, orlenReq)
	if err != nil {
		return nil, fmt.Errorf("orlen_paczka: create shipment: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     resp.ShipmentID,
		TrackingNumber: resp.TrackingNumber,
		Status:         resp.Status,
		LabelURL:       resp.LabelURL,
	}, nil
}

func (p *OrlenPaczkaProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

func (p *OrlenPaczkaProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("orlen_paczka: get tracking: %w", err)
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

func (p *OrlenPaczkaProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *OrlenPaczkaProvider) MapStatus(carrierStatus string) (string, bool) {
	return orlensdk.MapStatus(carrierStatus)
}

func (p *OrlenPaczkaProvider) SupportsPickupPoints() bool { return true }

func (p *OrlenPaczkaProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	resp, err := p.client.Points.SearchPoints(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("orlen_paczka: search points: %w", err)
	}

	points := make([]integration.PickupPoint, 0, len(resp.Points))
	for _, pt := range resp.Points {
		points = append(points, integration.PickupPoint{
			ID:         pt.ID,
			Name:       pt.Name,
			Street:     pt.Street,
			City:       pt.City,
			PostalCode: pt.PostalCode,
			Latitude:   pt.Latitude,
			Longitude:  pt.Longitude,
			Type:       pt.Type,
		})
	}

	return points, nil
}
