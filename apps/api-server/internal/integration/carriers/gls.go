package carriers

import (
	"context"
	"encoding/base64"
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
	Username string `json:"username"`
	Password string `json:"password"`
	Sandbox  bool   `json:"sandbox,omitempty"`
}

// GLSProvider implements integration.CarrierProvider for GLS Poland.
type GLSProvider struct {
	client    *glssdk.Client
	logger    *slog.Logger
	labelData map[string][]byte // externalID → decoded PDF label from CreateShipment
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

	client := glssdk.NewClient(creds.Username, creds.Password, opts...)

	return &GLSProvider{
		client: client,
		logger: slog.Default().With("provider", "gls"),
	}, nil
}

func (p *GLSProvider) ProviderName() string { return "gls" }

func (p *GLSProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	product, err := mapGLSServiceType(req.ServiceType)
	if err != nil {
		return nil, err
	}

	// NOTE: GLS uses ContactID (a pre-registered shipper reference) instead of
	// inline shipper address fields like DHL/DPD. The req.Shipper address is
	// intentionally not mapped — the default GLS account shipper is used.
	// To use a specific ContactID, configure it via GLS integration settings.
	glsReq := &glssdk.CreateParcelRequest{
		Product: product,
		Consignee: glssdk.Consignee{
			Address: glssdk.ConsigneeAddress{
				Name1:       req.Receiver.Name,
				Street:      req.Receiver.Street,
				City:        req.Receiver.City,
				ZIPCode:     req.Receiver.PostalCode,
				CountryCode: req.Receiver.Country,
				Phone:       req.Receiver.Phone,
				EMail:       req.Receiver.Email,
			},
		},
		ShippingUnit: []glssdk.ShipmentUnit{
			{
				Weight: req.Parcel.WeightKg,
				Width:  req.Parcel.WidthCm,
				Height: req.Parcel.HeightCm,
				Depth:  req.Parcel.DepthCm,
			},
		},
		PrintingOptions: &glssdk.PrintingOptions{
			ReturnLabels: glssdk.LabelOptions{
				TemplateSet: "NONE",
				LabelFormat: "PDF",
			},
		},
	}

	if req.Reference != "" {
		glsReq.References = []string{req.Reference}
	}

	var services []glssdk.Service
	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		services = append(services, glssdk.Service{
			ServiceName: "service_cash",
			Cash: &glssdk.CashService{
				Amount:   req.CODAmount,
				Currency: currency,
				Reason:   "COD",
			},
		})
	}

	if req.InsuredValue > 0 {
		services = append(services, glssdk.Service{
			ServiceName: "service_addonliability",
			AddOnLiability: &glssdk.AddOnLiabilityService{
				ParcelContent: "goods",
				Currency:      "PLN",
				Amount:        req.InsuredValue,
			},
		})
	}

	if len(services) > 0 {
		glsReq.Service = &glssdk.ServiceSection{
			Service: services,
		}
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

	// GLS returns labels inline in the create response (no separate label API).
	// Decode and cache the label so GetLabel can return it for this provider instance.
	if externalID != "" && len(resp.PrintData) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(resp.PrintData[0])
		if err == nil && len(decoded) > 0 {
			if p.labelData == nil {
				p.labelData = make(map[string][]byte)
			}
			p.labelData[externalID] = decoded
		}
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     externalID,
		TrackingNumber: trackingNumber,
		Status:         "PREADVICE",
	}, nil
}

// GetLabel returns the label for a shipment created in this session. GLS returns
// labels inline during CreateShipment (CreatedShipment.PrintData[].Data as base64)
// and has no separate label retrieval endpoint. The decoded label is cached by
// CreateShipment and returned here by externalID.
func (p *GLSProvider) GetLabel(_ context.Context, externalID string, _ string) ([]byte, error) {
	if data, ok := p.labelData[externalID]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("gls: labels are embedded in the create shipment response — use PrintData from CreateShipment")
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

func (p *GLSProvider) GetRates(_ context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	// TODO: Implement real GLS rate API integration.
	domestic := (req.FromCountry == "" || req.FromCountry == "PL") &&
		(req.ToCountry == "" || req.ToCountry == "PL")
	if !domestic {
		return nil, nil
	}

	w := req.Weight
	var rates []integration.Rate

	if w <= 31.5 {
		price := 13.99
		if w > 10 {
			price = 16.99
		}
		if w > 20 {
			price = 19.99
		}
		if req.COD > 0 {
			price += 4.00
		}
		rates = append(rates, integration.Rate{
			CarrierName:   "GLS",
			CarrierCode:   "gls",
			ServiceName:   "GLS Business Parcel",
			Price:         price,
			Currency:      "PLN",
			EstimatedDays: 2,
			PickupPoint:   false,
		})
	}

	return rates, nil
}

// SupportsPickupPoints returns false as GLS pickup point search API is not yet integrated.
func (p *GLSProvider) SupportsPickupPoints() bool { return false }

// SearchPickupPoints is not yet implemented for GLS.
func (p *GLSProvider) SearchPickupPoints(_ context.Context, _ string) ([]integration.PickupPoint, error) {
	return nil, fmt.Errorf("gls: pickup point search not yet implemented")
}

// mapGLSServiceType maps OMS service types to GLS product codes.
// GLS ShipIT API accepts: PARCEL, EXPRESS_10, EXPRESS_12.
func mapGLSServiceType(serviceType string) (string, error) {
	switch serviceType {
	case "standard", "":
		return "PARCEL", nil
	case "express_10":
		return "EXPRESS_10", nil
	case "express_12":
		return "EXPRESS_12", nil
	default:
		return "", fmt.Errorf("gls: unknown service type %q — valid types: standard, express_10, express_12", serviceType)
	}
}
