package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	glssdk "github.com/openoms-org/openoms/packages/gls-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("gls", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewGLSProvider(credentials, settings)
	})
}

// GLSCredentials is the JSON structure stored in encrypted integration credentials.
type GLSCredentials struct {
	APIKey  string `json:"api_key"`
	Sandbox bool   `json:"sandbox,omitempty"`
}

// GLSProvider implements integration.CarrierProvider for GLS Poland.
type GLSProvider struct {
	client *glssdk.Client
	logger *slog.Logger
}

// NewGLSProvider creates a GLS CarrierProvider from encrypted credentials.
func NewGLSProvider(credentials json.RawMessage, settings json.RawMessage) (*GLSProvider, error) {
	var creds GLSCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("gls: parse credentials: %w", err)
	}

	var opts []glssdk.Option
	if creds.Sandbox {
		opts = append(opts, glssdk.WithSandbox())
	}

	client := glssdk.NewClient(creds.APIKey, opts...)

	return &GLSProvider{
		client: client,
		logger: slog.Default().With("provider", "gls"),
	}, nil
}

func (p *GLSProvider) ProviderName() string { return "gls" }

func (p *GLSProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	glsReq := &glssdk.CreateParcelRequest{
		Consignee: glssdk.Party{
			Name:        req.Receiver.Name,
			Email:       req.Receiver.Email,
			Phone:       req.Receiver.Phone,
			Street:      req.Receiver.Street,
			City:        req.Receiver.City,
			ZipCode:     req.Receiver.PostalCode,
			CountryCode: req.Receiver.Country,
		},
		Parcels: []glssdk.Parcel{
			{
				Weight: req.Parcel.WeightKg,
				Width:  req.Parcel.WidthCm,
				Height: req.Parcel.HeightCm,
				Length: req.Parcel.DepthCm,
			},
		},
		Reference: req.Reference,
	}

	if req.CODAmount > 0 {
		glsReq.Services = append(glsReq.Services, "COD")
	}

	if req.InsuredValue > 0 {
		glsReq.Services = append(glsReq.Services, "INS")
	}

	resp, err := p.client.Shipments.Create(ctx, glsReq)
	if err != nil {
		return nil, fmt.Errorf("gls: create shipment: %w", err)
	}

	var externalID, trackingNumber string
	if len(resp.ParcelIDs) > 0 {
		externalID = resp.ParcelIDs[0]
	}
	if len(resp.TrackIDs) > 0 {
		trackingNumber = resp.TrackIDs[0]
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     externalID,
		TrackingNumber: trackingNumber,
		Status:         "PREADVICE",
	}, nil
}

func (p *GLSProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

func (p *GLSProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("gls: get tracking: %w", err)
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

func (p *GLSProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *GLSProvider) MapStatus(carrierStatus string) (string, bool) {
	return glssdk.MapStatus(carrierStatus)
}

func (p *GLSProvider) SupportsPickupPoints() bool { return true }

func (p *GLSProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	// GLS Szybka Paczka pickup points — requires separate API endpoint.
	// TODO: implement when GLS pickup points API is available.
	return nil, nil
}
