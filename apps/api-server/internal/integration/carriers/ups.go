package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	upssdk "github.com/openoms-org/openoms/packages/ups-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
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
func NewUPSProvider(credentials json.RawMessage, _ json.RawMessage) (*UPSProvider, error) {
	var creds UPSCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("ups: parse credentials: %w", err)
	}

	var opts []upssdk.Option
	if creds.Sandbox {
		opts = append(opts, upssdk.WithSandbox())
	}

	opts = append(opts, upssdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)))
	client := upssdk.NewClient(creds.ClientID, creds.ClientSecret, opts...)

	return &UPSProvider{
		client: client,
		logger: slog.Default().With("provider", "ups"),
	}, nil
}

// ProviderName returns the carrier provider identifier.
func (p *UPSProvider) ProviderName() string { return "ups" }

// CreateShipment creates a UPS shipment and returns the response with tracking info.
func (p *UPSProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcCode := req.ServiceType
	if svcCode == "" {
		svcCode = "11" // UPS Standard
	}

	upsReq := &upssdk.ShipmentRequest{
		ShipTo: upssdk.Party{
			Name: req.Receiver.Name,
			Address: upssdk.Address{
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

// GetLabel downloads the shipping label for the given UPS shipment.
func (p *UPSProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	data, err := p.client.Shipments.GetLabel(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("ups: get label: %w", err)
	}
	return data, nil
}

// GetTracking returns tracking events for the given UPS shipment.
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

// CancelShipment cancels a UPS shipment by its external ID.
func (p *UPSProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

// MapStatus maps a UPS carrier status to the internal shipment status.
func (p *UPSProvider) MapStatus(carrierStatus string) (string, bool) {
	return upssdk.MapStatus(carrierStatus)
}

// GetRates returns estimated shipping rates for UPS.
func (p *UPSProvider) GetRates(_ context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	// TODO: Implement real UPS Rating API integration.
	domestic := (req.FromCountry == "" || req.FromCountry == "PL") &&
		(req.ToCountry == "" || req.ToCountry == "PL")
	if !domestic {
		return nil, nil
	}

	w := req.Weight
	var rates []integration.Rate

	if w <= 30 {
		price := 22.00
		if w > 10 {
			price = 28.00
		}
		if w > 20 {
			price = 35.00
		}
		if req.COD > 0 {
			price += 6.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "UPS",
			CarrierCode:   "ups",
			ServiceName:   "UPS Standard",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 3,
			PickupPoint:   false,
			IsEstimate:    true,
		})
	}

	return rates, nil
}

// SupportsPickupPoints reports that UPS does not support pickup point delivery.
func (p *UPSProvider) SupportsPickupPoints() bool { return false }

// SearchPickupPoints is not supported by UPS.
func (p *UPSProvider) SearchPickupPoints(_ context.Context, _ string) ([]integration.PickupPoint, error) {
	return nil, fmt.Errorf("ups: pickup point search not supported")
}
