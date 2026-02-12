package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

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

	// Set custom attributes (target point and/or sending method)
	if req.TargetPoint != "" || req.SendingMethod != "" {
		inpostReq.CustomAttributes = &inpostsdk.CustomAttributes{}
		if req.TargetPoint != "" {
			inpostReq.CustomAttributes.TargetPoint = req.TargetPoint
		}
		if req.SendingMethod != "" {
			inpostReq.CustomAttributes.SendingMethod = req.SendingMethod
		}
	}

	// Set receiver address for courier services
	if req.TargetPoint == "" {
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

	// InPost generates offers asynchronously. Poll until offers are available, then buy.
	shipmentID := shipment.ID
	var offerID int64
	for attempt := 0; attempt < 10; attempt++ {
		if len(shipment.Offers) > 0 {
			offerID = shipment.Offers[0].ID
			break
		}
		// If status is already "confirmed", no need to buy
		if shipment.Status == "confirmed" {
			break
		}
		time.Sleep(time.Duration(500+attempt*500) * time.Millisecond)
		polled, err := p.client.Shipments.Get(ctx, shipmentID)
		if err != nil {
			p.logger.Warn("inpost: poll shipment failed", "id", shipmentID, "error", err)
			break
		}
		shipment = polled
	}

	// Check for payment failures in transactions
	for _, tx := range shipment.Transactions {
		if tx.Status == "failure" {
			return nil, fmt.Errorf("inpost: opłacenie przesyłki nie powiodło się — sprawdź rozliczenia konta InPost (ID przesyłki InPost: %d)", shipmentID)
		}
	}

	if shipment.Status != "confirmed" && offerID > 0 {
		bought, err := p.client.Shipments.Buy(ctx, shipmentID, offerID)
		if err != nil {
			p.logger.Warn("inpost: buy failed", "id", shipmentID, "offer_id", offerID, "error", err)
		} else {
			shipment = bought
			// Check transactions again after buy
			for _, tx := range shipment.Transactions {
				if tx.Status == "failure" {
					return nil, fmt.Errorf("inpost: opłacenie przesyłki nie powiodło się — sprawdź rozliczenia konta InPost (ID przesyłki InPost: %d)", shipmentID)
				}
			}
		}
	} else if shipment.Status != "confirmed" && offerID == 0 {
		p.logger.Warn("inpost: no offers available after polling", "id", shipmentID, "status", shipment.Status)
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
	resp, err := p.client.Tracking.Get(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("inpost get tracking: %w", err)
	}

	events := make([]integration.TrackingEvent, 0, len(resp.TrackingDetails))
	for _, td := range resp.TrackingDetails {
		ts, _ := time.Parse(time.RFC3339, td.Datetime)
		events = append(events, integration.TrackingEvent{
			Status:    td.Status,
			Location:  td.Agency,
			Timestamp: ts,
			Details:   td.OriginStatus,
		})
	}
	return events, nil
}

func (p *InPostProvider) CancelShipment(ctx context.Context, externalID string) error {
	return fmt.Errorf("inpost: cancel shipment not implemented")
}

func (p *InPostProvider) CreateDispatchOrder(ctx context.Context, shipmentExternalIDs []int64, address integration.DispatchOrderAddress, contact integration.DispatchOrderContact) (int64, error) {
	req := &inpostsdk.CreateDispatchOrderRequest{
		Shipments: shipmentExternalIDs,
		Address: &inpostsdk.DispatchOrderAddress{
			Street:         address.Street,
			BuildingNumber: address.BuildingNumber,
			City:           address.City,
			PostCode:       address.PostCode,
			CountryCode:    address.CountryCode,
		},
		Name:    contact.Name,
		Phone:   contact.Phone,
		Email:   contact.Email,
		Comment: contact.Comment,
	}
	order, err := p.client.DispatchOrders.Create(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("inpost: create dispatch order: %w", err)
	}
	return order.ID, nil
}

func (p *InPostProvider) MapStatus(carrierStatus string) (string, bool) {
	return inpostsdk.MapStatus(carrierStatus)
}

func (p *InPostProvider) SupportsPickupPoints() bool { return true }

func (p *InPostProvider) GetRates(_ context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	// InPost does not expose a real-time rate API.
	// Use hardcoded Polish domestic pricing tiers (net prices approximation).
	var rates []integration.Rate

	w := req.Weight
	width := req.Width
	height := req.Height
	length := req.Length

	// Paczkomat sizing:
	//   A: max 8 kg, fits in 38x64x8 cm
	//   B: max 25 kg, fits in 38x64x19 cm
	//   C: max 25 kg, fits in 41x38x64 cm (largest)
	fitsA := w <= 8 && width <= 38 && height <= 8 && length <= 64
	fitsB := w <= 25 && width <= 38 && height <= 19 && length <= 64
	fitsC := w <= 25 && width <= 41 && height <= 38 && length <= 64

	// Only return paczkomat rates for domestic PL shipments
	domestic := (req.FromCountry == "" || req.FromCountry == "PL") &&
		(req.ToCountry == "" || req.ToCountry == "PL")

	if domestic {
		if fitsA {
			price := 12.99
			if req.COD > 0 {
				price += 3.50
			}
			rates = append(rates, integration.Rate{
				CarrierName:   "InPost",
				CarrierCode:   "inpost",
				ServiceName:   "Paczkomat A (mała)",
				Price:         price,
				Currency:      "PLN",
				EstimatedDays: 2,
				PickupPoint:   true,
			})
		}
		if fitsB {
			price := 13.99
			if req.COD > 0 {
				price += 3.50
			}
			rates = append(rates, integration.Rate{
				CarrierName:   "InPost",
				CarrierCode:   "inpost",
				ServiceName:   "Paczkomat B (średnia)",
				Price:         price,
				Currency:      "PLN",
				EstimatedDays: 2,
				PickupPoint:   true,
			})
		}
		if fitsC {
			price := 15.49
			if req.COD > 0 {
				price += 3.50
			}
			rates = append(rates, integration.Rate{
				CarrierName:   "InPost",
				CarrierCode:   "inpost",
				ServiceName:   "Paczkomat C (duża)",
				Price:         price,
				Currency:      "PLN",
				EstimatedDays: 2,
				PickupPoint:   true,
			})
		}

		// Courier rate (up to 25 kg for standard)
		if w <= 25 {
			price := 16.99
			if w > 10 {
				price = 19.99
			}
			if req.COD > 0 {
				price += 4.00
			}
			rates = append(rates, integration.Rate{
				CarrierName:   "InPost",
				CarrierCode:   "inpost",
				ServiceName:   "Kurier Standard",
				Price:         price,
				Currency:      "PLN",
				EstimatedDays: 1,
				PickupPoint:   false,
			})
		}
	}

	return rates, nil
}

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
