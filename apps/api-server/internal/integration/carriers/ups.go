package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	upssdk "github.com/openoms-org/openoms/packages/ups-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("ups", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewUPSProvider(credentials, settings)
	})
}

// UPSCredentials is the JSON structure stored in encrypted integration credentials.
type UPSCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Sandbox      bool   `json:"sandbox,omitempty"`
}

// UPSProvider implements integration.CarrierProvider for UPS.
type UPSProvider struct {
	client *upssdk.Client
	logger *slog.Logger
}

// NewUPSProvider creates a UPS CarrierProvider from encrypted credentials.
func NewUPSProvider(credentials json.RawMessage, settings json.RawMessage) (*UPSProvider, error) {
	var creds UPSCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("ups: parse credentials: %w", err)
	}

	var opts []upssdk.Option
	if creds.Sandbox {
		opts = append(opts, upssdk.WithSandbox())
	}

	client := upssdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)

	return &UPSProvider{
		client: client,
		logger: slog.Default().With("provider", "ups"),
	}, nil
}

func (p *UPSProvider) ProviderName() string { return "ups" }

func (p *UPSProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcCode := req.ServiceType
	if svcCode == "" {
		svcCode = "11" // UPS Standard
	}

	upsReq := &upssdk.ShipmentRequest{
		ShipTo: upssdk.Party{
			Name: req.Receiver.Name,
			Address: upssdk.UPSAddress{
				AddressLine: []string{req.Receiver.Street},
				City:        req.Receiver.City,
				PostalCode:  req.Receiver.PostalCode,
				CountryCode: req.Receiver.Country,
			},
		},
		Service: upssdk.ServiceCode{
			Code: svcCode,
		},
		Package: []upssdk.PackageSpec{
			{
				PackagingType: upssdk.Code{Code: "02"}, // Customer Supplied Package
				Dimensions: upssdk.Dims{
					UnitOfMeasurement: upssdk.Code{Code: "CM"},
					Length:            fmt.Sprintf("%.0f", req.Parcel.DepthCm),
					Width:             fmt.Sprintf("%.0f", req.Parcel.WidthCm),
					Height:            fmt.Sprintf("%.0f", req.Parcel.HeightCm),
				},
				PackageWeight: upssdk.PkgWeight{
					UnitOfMeasurement: upssdk.Code{Code: "KGS"},
					Weight:            fmt.Sprintf("%.1f", req.Parcel.WeightKg),
				},
			},
		},
	}

	if req.Receiver.Phone != "" {
		upsReq.ShipTo.Phone = &upssdk.Phone{Number: req.Receiver.Phone}
	}

	if req.Reference != "" {
		upsReq.Reference = &upssdk.Reference{Value: req.Reference}
	}

	resp, err := p.client.Shipments.Create(ctx, upsReq)
	if err != nil {
		return nil, fmt.Errorf("ups: create shipment: %w", err)
	}

	result := &integration.CarrierShipmentResponse{
		ExternalID:     resp.ShipmentID,
		TrackingNumber: resp.TrackingNumber,
		Status:         "M", // Manifest
	}

	// If label image was returned inline, provide it as a label URL (data URI)
	if resp.LabelImage != "" {
		result.LabelURL = "data:application/pdf;base64," + resp.LabelImage
	}

	return result, nil
}

func (p *UPSProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	data, err := p.client.Shipments.GetLabel(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("ups: get label: %w", err)
	}
	return data, nil
}

func (p *UPSProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("ups: get tracking: %w", err)
	}

	events := make([]integration.TrackingEvent, 0, len(resp.Events))
	for _, ev := range resp.Events {
		events = append(events, integration.TrackingEvent{
			Status:    ev.Status,
			Location:  ev.Location,
			Timestamp: ev.Timestamp,
			Details:   ev.Description,
		})
	}

	return events, nil
}

func (p *UPSProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *UPSProvider) MapStatus(carrierStatus string) (string, bool) {
	return upssdk.MapStatus(carrierStatus)
}

func (p *UPSProvider) SupportsPickupPoints() bool { return false }

func (p *UPSProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	return nil, nil
}
