package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	dpdsdk "github.com/openoms-org/openoms/packages/dpd-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
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
func NewDPDProvider(credentials json.RawMessage, _ json.RawMessage) (*DPDProvider, error) {
	var creds DPDCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("dpd: parse credentials: %w", err)
	}

	var opts []dpdsdk.Option
	if creds.Sandbox {
		opts = append(opts, dpdsdk.WithSandbox())
	}

	opts = append(opts, dpdsdk.WithHTTPClient(netutil.SafeHTTPClient(30*time.Second)))
	client := dpdsdk.NewClient(creds.Login, creds.Password, creds.MasterFid, opts...)

	return &DPDProvider{
		client: client,
		logger: slog.Default().With("provider", "dpd"),
	}, nil
}

// ProviderName returns the carrier provider identifier.
func (p *DPDProvider) ProviderName() string { return "dpd" }

// CreateShipment creates a DPD shipment and returns the response with tracking info.
func (p *DPDProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcType, err := mapDPDServiceType(req.ServiceType)
	if err != nil {
		return nil, err
	}

	if svcType == "PICKUP" && req.TargetPoint == "" {
		return nil, fmt.Errorf("dpd: target point is required for dpd_pickup service type")
	}

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
		Reference:   req.Reference,
		ServiceType: svcType,
	}

	if req.TargetPoint != "" {
		dpdReq.TargetPoint = req.TargetPoint
	}

	if req.Shipper != nil {
		dpdReq.Sender = &dpdsdk.Address{
			Name:        req.Shipper.Name,
			Phone:       req.Shipper.Phone,
			Email:       req.Shipper.Email,
			Street:      req.Shipper.Street,
			City:        req.Shipper.City,
			PostalCode:  req.Shipper.PostalCode,
			CountryCode: req.Shipper.Country,
		}
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

	// Use waybill as ExternalID — label generation requires the waybill number.
	return &integration.CarrierShipmentResponse{
		ExternalID:     resp.Waybill,
		TrackingNumber: resp.Waybill,
		Status:         resp.Status,
	}, nil
}

// GetLabel downloads the shipping label for the given DPD shipment.
func (p *DPDProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	return p.client.Shipments.GetLabel(ctx, externalID)
}

// GetTracking returns tracking events for the given DPD shipment.
func (p *DPDProvider) GetTracking(_ context.Context, _ string) ([]integration.TrackingEvent, error) {
	return nil, fmt.Errorf("dpd: tracking not available via DPD REST API")
}

// CancelShipment cancels a DPD shipment by its external ID.
func (p *DPDProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

// MapStatus maps a DPD carrier status to the internal shipment status.
func (p *DPDProvider) MapStatus(carrierStatus string) (string, bool) {
	return dpdsdk.MapStatus(carrierStatus)
}

// GetRates reports that DPD rate shopping is not implemented.
func (p *DPDProvider) GetRates(_ context.Context, _ integration.RateRequest) ([]integration.Rate, error) {
	return nil, fmt.Errorf("dpd: %w", integration.ErrCarrierRatesNotImplemented)
}

// SupportsPickupPoints returns false as DPD does not support pickup point search via REST API.
func (p *DPDProvider) SupportsPickupPoints() bool { return false }

// SearchPickupPoints is not supported by DPD REST API and always returns an error.
func (p *DPDProvider) SearchPickupPoints(_ context.Context, _ string) ([]integration.PickupPoint, error) {
	return nil, fmt.Errorf("dpd: pickup point search not yet implemented")
}

func mapDPDServiceType(serviceType string) (string, error) {
	switch serviceType {
	case "dpd_classic", "":
		return "STANDARD", nil
	case "dpd_pickup":
		return "PICKUP", nil
	default:
		return "", fmt.Errorf("dpd: unknown service type %q — valid types: dpd_classic, dpd_pickup", serviceType)
	}
}
