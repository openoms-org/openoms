package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	inpostsdk "github.com/openoms-org/openoms/packages/inpost-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("inpost", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewInPostProvider(credentials, settings)
	})
}

// InPostCredentials is the JSON structure stored in encrypted integration credentials.
type InPostCredentials struct {
	APIToken       string `json:"api_token"`
	OrganizationID string `json:"organization_id"`
	Sandbox        bool   `json:"sandbox,omitempty"`
}

// InPostProvider implements integration.CarrierProvider for InPost ShipX.
type InPostProvider struct {
	client *inpostsdk.Client
	logger *slog.Logger
}

// NewInPostProvider creates an InPost CarrierProvider from encrypted credentials.
func NewInPostProvider(credentials json.RawMessage, settings json.RawMessage) (*InPostProvider, error) {
	var creds InPostCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("inpost: parse credentials: %w", err)
	}

	var opts []inpostsdk.Option
	if creds.Sandbox {
		opts = append(opts, inpostsdk.WithSandbox())
	}

	client := inpostsdk.NewClient(creds.APIToken, creds.OrganizationID, opts...)

	return &InPostProvider{
		client: client,
		logger: slog.Default().With("provider", "inpost"),
	}, nil
}

func (p *InPostProvider) ProviderName() string { return "inpost" }

func (p *InPostProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	// Determine service type
	svcType := inpostsdk.ServiceCourierStandard
	if req.TargetPoint != "" {
		svcType = inpostsdk.ServiceLockerStandard
	} else if req.ServiceType != "" {
		svcType = inpostsdk.ServiceType(req.ServiceType)
	}

	// Build parcel
	parcel := inpostsdk.Parcel{
		Weight: inpostsdk.Weight{Amount: req.Parcel.WeightKg, Unit: "kg"},
	}
	if req.Parcel.SizeCode != "" {
		parcel.Template = inpostsdk.ParcelTemplate(req.Parcel.SizeCode)
	}
	if req.Parcel.WidthCm > 0 || req.Parcel.HeightCm > 0 || req.Parcel.DepthCm > 0 {
		parcel.Dimensions = &inpostsdk.Dimensions{
			Width:  req.Parcel.WidthCm * 10, // cm → mm
			Height: req.Parcel.HeightCm * 10,
			Length: req.Parcel.DepthCm * 10,
		}
	}

	inpostReq := &inpostsdk.CreateShipmentRequest{
		Receiver: inpostsdk.Receiver{
			Name:  req.Receiver.Name,
			Phone: req.Receiver.Phone,
			Email: req.Receiver.Email,
		},
		Parcels:   []inpostsdk.Parcel{parcel},
		Service:   svcType,
		Reference: req.Reference,
	}

	// Set target point for locker or address for courier
	if req.TargetPoint != "" {
		inpostReq.CustomAttributes = &inpostsdk.CustomAttributes{
			TargetPoint: req.TargetPoint,
		}
	} else {
		inpostReq.Receiver.Address = &inpostsdk.Address{
			Street:      req.Receiver.Street,
			City:        req.Receiver.City,
			PostCode:    req.Receiver.PostalCode,
			CountryCode: req.Receiver.Country,
		}
	}

	shipment, err := p.client.Shipments.Create(ctx, inpostReq)
	if err != nil {
		return nil, fmt.Errorf("inpost: create shipment: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     strconv.FormatInt(shipment.ID, 10),
		TrackingNumber: shipment.TrackingNumber,
		Status:         shipment.Status,
	}, nil
}

func (p *InPostProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	id, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("inpost: invalid shipment ID %q: %w", externalID, err)
	}

	labelFmt := inpostsdk.LabelPDF
	switch format {
	case "zpl":
		labelFmt = inpostsdk.LabelZPL
	case "epl":
		labelFmt = inpostsdk.LabelEPL
	}

	return p.client.Labels.Get(ctx, id, labelFmt)
}

func (p *InPostProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	// TODO: InPost SDK does not have a tracking endpoint yet.
	return nil, nil
}

func (p *InPostProvider) CancelShipment(ctx context.Context, externalID string) error {
	return fmt.Errorf("inpost: cancel shipment not implemented")
}

func (p *InPostProvider) MapStatus(carrierStatus string) (string, bool) {
	return inpostsdk.MapStatus(carrierStatus)
}

func (p *InPostProvider) SupportsPickupPoints() bool { return true }

func (p *InPostProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	resp, err := p.client.Points.Search(ctx, query, inpostsdk.PointTypeParcelLocker, 10)
	if err != nil {
		return nil, fmt.Errorf("inpost: search points: %w", err)
	}

	points := make([]integration.PickupPoint, 0, len(resp.Items))
	for _, pt := range resp.Items {
		pp := integration.PickupPoint{
			ID:        pt.Name,
			Name:      pt.Name,
			Latitude:  pt.Location.Latitude,
			Longitude: pt.Location.Longitude,
		}
		if pt.AddressDetails != nil {
			pp.Street = pt.AddressDetails.Street
			if pt.AddressDetails.BuildingNumber != "" {
				pp.Street += " " + pt.AddressDetails.BuildingNumber
			}
			pp.City = pt.AddressDetails.City
			pp.PostalCode = pt.AddressDetails.PostCode
		}
		if len(pt.Type) > 0 {
			pp.Type = pt.Type[0]
		}
		points = append(points, pp)
	}

	return points, nil
}
