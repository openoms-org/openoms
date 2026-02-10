package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
func NewDHLProvider(credentials json.RawMessage, settings json.RawMessage) (*DHLProvider, error) {
	var creds DHLCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("dhl: parse credentials: %w", err)
	}

	var opts []dhlsdk.Option
	if creds.Sandbox {
		opts = append(opts, dhlsdk.WithSandbox())
	}

	client := dhlsdk.NewClient(creds.Username, creds.Password, creds.AccountNumber, opts...)

	return &DHLProvider{
		client: client,
		logger: slog.Default().With("provider", "dhl"),
	}, nil
}

func (p *DHLProvider) ProviderName() string { return "dhl" }

func (p *DHLProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcType := req.ServiceType
	if svcType == "" {
		svcType = "AH" // DHL Parcel domestic
	}

	dhlReq := &dhlsdk.CreateShipmentRequest{
		ShipperAccount: p.client.AccountNumber(),
		Receiver: dhlsdk.Receiver{
			Name:       req.Receiver.Name,
			Email:      req.Receiver.Email,
			Phone:      req.Receiver.Phone,
			Street:     req.Receiver.Street,
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

func (p *DHLProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

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

func (p *DHLProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *DHLProvider) MapStatus(carrierStatus string) (string, bool) {
	return dhlsdk.MapStatus(carrierStatus)
}

func (p *DHLProvider) SupportsPickupPoints() bool { return false }

func (p *DHLProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	return nil, nil
}
