package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	apaczka "github.com/openoms-org/openoms/packages/apaczka-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

func init() {
	integration.RegisterCarrierProvider("apaczka", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewApaczkaProvider(credentials, settings)
	})
}

// ApaczkaCredentials is the JSON structure stored in encrypted integration credentials.
type ApaczkaCredentials struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// apaczkaSettings holds optional provider settings.
type apaczkaSettings struct {
	BaseURL        string `json:"base_url,omitempty"`
	CODBankAccount string `json:"cod_bank_account,omitempty"`
}

// ApaczkaProvider implements integration.CarrierProvider for the Apaczka broker.
type ApaczkaProvider struct {
	client   *apaczka.Client
	settings apaczkaSettings
	logger   *slog.Logger
}

// NewApaczkaProvider creates an Apaczka CarrierProvider from encrypted credentials
// and optional settings.
func NewApaczkaProvider(credentials json.RawMessage, settings json.RawMessage) (*ApaczkaProvider, error) {
	var creds ApaczkaCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("apaczka: parse credentials: %w", err)
	}
	if creds.AppID == "" || creds.AppSecret == "" {
		return nil, fmt.Errorf("apaczka: credentials must include app_id and app_secret")
	}

	var cfg apaczkaSettings
	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("apaczka: parse settings: %w", err)
		}
	}

	opts := []apaczka.Option{
		apaczka.WithHTTPClient(netutil.SafeHTTPClient(30 * time.Second)),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, apaczka.WithBaseURL(cfg.BaseURL))
	}

	client := apaczka.NewClient(creds.AppID, creds.AppSecret, opts...)

	return &ApaczkaProvider{
		client:   client,
		settings: cfg,
		logger:   slog.Default().With("provider", "apaczka"),
	}, nil
}

// ProviderName returns the carrier provider identifier.
func (p *ApaczkaProvider) ProviderName() string { return "apaczka" }

// GetRates returns shipping rates for all Apaczka broker services that have
// a price for the given shipment dimensions.
//
// Implementation: fetches the service catalog via GetServiceStructure, then
// issues a single GetValuation call with the full dimensions. The returned
// price_table maps service_id → price row; we emit one Rate per matched service.
//
// Note: If live testing shows order_valuation needs a per-service_id call,
// switch to a bounded concurrent per-service loop.
func (p *ApaczkaProvider) GetRates(ctx context.Context, req integration.RateRequest) ([]integration.Rate, error) {
	svcResp, err := p.client.GetServiceStructure(ctx)
	if err != nil {
		return nil, fmt.Errorf("apaczka: get service structure: %w", err)
	}

	valReq := &apaczka.ValuationRequest{
		Order: apaczka.OrderInput{
			Address: apaczka.AddressPair{
				Sender: apaczka.Address{
					PostalCode:  req.FromPostalCode,
					CountryCode: defaultStr(req.FromCountry, "PL"),
				},
				Receiver: apaczka.Address{
					PostalCode:  req.ToPostalCode,
					CountryCode: defaultStr(req.ToCountry, "PL"),
				},
			},
			Shipment: []apaczka.ShipmentItem{
				{
					// req.Length maps to the same axis as CarrierParcel.DepthCm used in
					// CreateShipment (Dimension1), so valuation and order dimension
					// mappings stay intentionally consistent.
					Dimension1:       int(req.Length),
					Dimension2:       int(req.Width),
					Dimension3:       int(req.Height),
					Weight:           req.Weight,
					ShipmentTypeCode: "PACZKA",
				},
			},
		},
	}

	valResp, err := p.client.GetValuation(ctx, valReq)
	if err != nil {
		return nil, fmt.Errorf("apaczka: get valuation: %w", err)
	}

	// Build a lookup map: service_id → Service metadata.
	svcByID := make(map[string]apaczka.Service, len(svcResp.Services))
	for _, svc := range svcResp.Services {
		svcByID[svc.ServiceID] = svc
	}

	rates := make([]integration.Rate, 0, len(valResp.PriceTable))
	for svcID, priceRow := range valResp.PriceTable {
		svc, ok := svcByID[svcID]
		if !ok {
			// Service returned in price_table but not in service_structure — skip.
			continue
		}

		grosze, err := strconv.ParseInt(priceRow.PriceGross, 10, 64)
		if err != nil {
			p.logger.Warn("apaczka: skipping service with unparseable price_gross",
				"service_id", svcID, "price_gross", priceRow.PriceGross)
			continue
		}

		rates = append(rates, integration.Rate{
			CarrierName: svc.Name,
			CarrierCode: svc.Supplier,
			// ServiceName stores the service_id so it round-trips as CreateShipment's ServiceType.
			ServiceName: svc.ServiceID,
			Price:       float64(grosze) / 100.0,
			Currency:    "PLN",
			PickupPoint: svc.DoorToDoor == "0",
			IsEstimate:  false,
		})
	}

	return rates, nil
}

// CreateShipment creates a new shipment order via the Apaczka broker.
// req.ServiceType must be set to the service_id (e.g. as returned by GetRates).
func (p *ApaczkaProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	if req.ServiceType == "" {
		return nil, fmt.Errorf("apaczka: service_type (service_id) is required")
	}

	receiver := apaczka.Address{
		CountryCode: defaultStr(req.Receiver.Country, "PL"),
		Name:        req.Receiver.Name,
		Line1:       req.Receiver.Street,
		PostalCode:  req.Receiver.PostalCode,
		City:        req.Receiver.City,
		Email:       req.Receiver.Email,
		Phone:       req.Receiver.Phone,
	}

	var sender apaczka.Address
	if req.Shipper != nil {
		sender = apaczka.Address{
			CountryCode: defaultStr(req.Shipper.Country, "PL"),
			Name:        req.Shipper.Name,
			Line1:       req.Shipper.Street,
			PostalCode:  req.Shipper.PostalCode,
			City:        req.Shipper.City,
			Email:       req.Shipper.Email,
			Phone:       req.Shipper.Phone,
		}
	}

	orderInput := apaczka.OrderInput{
		ServiceID: req.ServiceType,
		Address: apaczka.AddressPair{
			Sender:   sender,
			Receiver: receiver,
		},
		Shipment: []apaczka.ShipmentItem{
			{
				Dimension1:       int(req.Parcel.DepthCm),
				Dimension2:       int(req.Parcel.WidthCm),
				Dimension3:       int(req.Parcel.HeightCm),
				Weight:           req.Parcel.WeightKg,
				ShipmentTypeCode: "PACZKA",
			},
		},
		Pickup: &apaczka.Pickup{
			Type: "SELF",
		},
	}

	if req.CODAmount > 0 {
		if p.settings.CODBankAccount == "" {
			p.logger.Warn("apaczka: COD requested but cod_bank_account setting is empty — order may be rejected by API")
		}
		orderInput.COD = &apaczka.COD{
			Amount:      int(math.Round(req.CODAmount * 100)), // PLN → grosze
			Currency:    defaultStr(req.CODCurrency, "PLN"),
			BankAccount: p.settings.CODBankAccount,
			Country:     "PL",
		}
	}

	createReq := &apaczka.CreateOrderRequest{Order: orderInput}
	order, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("apaczka: create order: %w", err)
	}

	return &integration.CarrierShipmentResponse{
		ExternalID:     order.ID,
		TrackingNumber: order.WaybillNumber,
		Status:         order.Status,
	}, nil
}

// GetLabel downloads the shipping label (PDF) for the given Apaczka order ID.
// The format parameter is ignored — the API always returns PDF.
func (p *ApaczkaProvider) GetLabel(ctx context.Context, externalID string, _ string) ([]byte, error) {
	data, err := p.client.GetWaybill(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("apaczka: get waybill: %w", err)
	}
	return data, nil
}

// GetTracking returns tracking events for the given waybill number.
func (p *ApaczkaProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("apaczka: get tracking: %w", err)
	}

	events := make([]integration.TrackingEvent, 0, len(resp.Tracking))
	for _, e := range resp.Tracking {
		var ts time.Time
		if e.UpdatedAt != "" {
			ts, _ = time.Parse(time.RFC3339, e.UpdatedAt)
		}
		events = append(events, integration.TrackingEvent{
			Status:    e.Status,
			Location:  e.Place,
			Timestamp: ts,
			Details:   e.Description,
		})
	}
	return events, nil
}

// CancelShipment cancels the Apaczka order with the given external ID.
func (p *ApaczkaProvider) CancelShipment(ctx context.Context, externalID string) error {
	if err := p.client.CancelOrder(ctx, externalID); err != nil {
		return fmt.Errorf("apaczka: cancel order: %w", err)
	}
	return nil
}

// MapStatus maps an Apaczka carrier status to the internal OMS shipment status.
func (p *ApaczkaProvider) MapStatus(carrierStatus string) (string, bool) {
	return mapApaczkaStatus(carrierStatus)
}

// SupportsPickupPoints reports that Apaczka supports pickup-point delivery.
func (p *ApaczkaProvider) SupportsPickupPoints() bool { return true }

// SearchPickupPoints returns pickup points available through the Apaczka broker.
//
// Note: The Apaczka broker point search is carrier-type-scoped (e.g. INPOST, UPS,
// POCZTA) and a country code — the free-text query string is not used by this API.
// This first cut returns InPost/PL points. A richer carrier-type mapping based on
// the query is a follow-up improvement.
func (p *ApaczkaProvider) SearchPickupPoints(ctx context.Context, _ string) ([]integration.PickupPoint, error) {
	pts, err := p.client.SearchPoints(ctx, "INPOST", "PL", "")
	if err != nil {
		return nil, fmt.Errorf("apaczka: search points: %w", err)
	}

	points := make([]integration.PickupPoint, 0, len(pts))
	for _, pt := range pts {
		points = append(points, integration.PickupPoint{
			ID:         pt.ID,
			Name:       pt.Name,
			Street:     pt.Line1,
			City:       pt.City,
			PostalCode: pt.PostalCode,
			Latitude:   pt.Lat,
			Longitude:  pt.Lng,
		})
	}
	return points, nil
}

// defaultStr returns val if non-empty, otherwise returns def.
func defaultStr(val, def string) string {
	if val != "" {
		return val
	}
	return def
}
