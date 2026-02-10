package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dpdsdk "github.com/openoms-org/openoms/packages/dpd-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("dpd", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewDPDProvider(credentials, settings)
	})
}

// DPDCredentials is the JSON structure stored in encrypted integration credentials.
type DPDCredentials struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	MasterFid string `json:"master_fid"`
	Sandbox   bool   `json:"sandbox,omitempty"`
}

// DPDProvider implements integration.CarrierProvider for DPD Poland.
type DPDProvider struct {
	client *dpdsdk.Client
	logger *slog.Logger
}

// NewDPDProvider creates a DPD CarrierProvider from encrypted credentials.
func NewDPDProvider(credentials json.RawMessage, settings json.RawMessage) (*DPDProvider, error) {
	var creds DPDCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("dpd: parse credentials: %w", err)
	}

	var opts []dpdsdk.Option
	if creds.Sandbox {
		opts = append(opts, dpdsdk.WithSandbox())
	}

	client := dpdsdk.NewClient(creds.Login, creds.Password, creds.MasterFid, opts...)

	return &DPDProvider{
		client: client,
		logger: slog.Default().With("provider", "dpd"),
	}, nil
}

func (p *DPDProvider) ProviderName() string { return "dpd" }

func (p *DPDProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	dpdReq := &dpdsdk.CreateParcelRequest{
		Receiver: dpdsdk.Address{
			Name:        req.Receiver.Name,
			Email:       req.Receiver.Email,
			Phone:       req.Receiver.Phone,
			Street:      req.Receiver.Street,
			City:        req.Receiver.City,
			PostalCode:  req.Receiver.PostalCode,
			CountryCode: req.Receiver.Country,
		},
		Parcels: []dpdsdk.ParcelSpec{
			{
				Weight: req.Parcel.WeightKg,
				SizeX:  req.Parcel.WidthCm,
				SizeY:  req.Parcel.HeightCm,
				SizeZ:  req.Parcel.DepthCm,
			},
		},
		Reference: req.Reference,
	}

	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		dpdReq.Services = &dpdsdk.Services{
			COD: &dpdsdk.COD{
				Amount:   req.CODAmount,
				Currency: currency,
			},
		}
	}

	if req.InsuredValue > 0 {
		if dpdReq.Services == nil {
			dpdReq.Services = &dpdsdk.Services{}
		}
		dpdReq.Services.DeclaredValue = &dpdsdk.Money{
			Amount:   req.InsuredValue,
			Currency: "PLN",
		}
	}

	resp, err := p.client.Shipments.Create(ctx, dpdReq)
	if err != nil {
		return nil, fmt.Errorf("dpd: create shipment: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     resp.ParcelID,
		TrackingNumber: resp.Waybill,
		Status:         resp.Status,
	}, nil
}

func (p *DPDProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

func (p *DPDProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("dpd: get tracking: %w", err)
	}

	events := make([]integration.TrackingEvent, 0, len(resp.Events))
	for _, ev := range resp.Events {
		events = append(events, integration.TrackingEvent{
			Status:    ev.Status,
			Location:  ev.Location,
			Timestamp: ev.DateTime,
			Details:   ev.Description,
		})
	}

	return events, nil
}

func (p *DPDProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *DPDProvider) MapStatus(carrierStatus string) (string, bool) {
	return dpdsdk.MapStatus(carrierStatus)
}

func (p *DPDProvider) SupportsPickupPoints() bool { return true }

func (p *DPDProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	// DPD Pickup points — requires separate API endpoint.
	// TODO: implement when DPD pickup points API is available.
	return nil, nil
}
