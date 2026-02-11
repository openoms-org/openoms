package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	fedexsdk "github.com/openoms-org/openoms/packages/fedex-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

func init() {
	integration.RegisterCarrierProvider("fedex", func(credentials json.RawMessage, settings json.RawMessage) (integration.CarrierProvider, error) {
		return NewFedExProvider(credentials, settings)
	})
}

// FedExCredentials is the JSON structure stored in encrypted integration credentials.
type FedExCredentials struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	AccountNumber string `json:"account_number"`
	Sandbox       bool   `json:"sandbox,omitempty"`
}

// FedExProvider implements integration.CarrierProvider for FedEx.
type FedExProvider struct {
	client *fedexsdk.Client
	logger *slog.Logger
}

// NewFedExProvider creates a FedEx CarrierProvider from encrypted credentials.
func NewFedExProvider(credentials json.RawMessage, settings json.RawMessage) (*FedExProvider, error) {
	var creds FedExCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("fedex: parse credentials: %w", err)
	}

	var opts []fedexsdk.Option
	if creds.Sandbox {
		opts = append(opts, fedexsdk.WithSandbox())
	}

	client := fedexsdk.NewClient(creds.ClientID, creds.ClientSecret, creds.AccountNumber, opts...)

	return &FedExProvider{
		client: client,
		logger: slog.Default().With("provider", "fedex"),
	}, nil
}

func (p *FedExProvider) ProviderName() string { return "fedex" }

func (p *FedExProvider) CreateShipment(ctx context.Context, req integration.CarrierShipmentRequest) (*integration.CarrierShipmentResponse, error) {
	svcType := req.ServiceType
	if svcType == "" {
		svcType = "FEDEX_INTERNATIONAL_PRIORITY"
	}

	fedexReq := &fedexsdk.ShipmentRequest{
		AccountNumber: fedexsdk.AccountNumber{Value: p.client.AccountNumber()},
		RequestedShipment: fedexsdk.RequestedShipment{
			ServiceType:   svcType,
			PackagingType: "YOUR_PACKAGING",
			Recipients: []fedexsdk.Party{
				{
					Contact: fedexsdk.Contact{
						PersonName:   req.Receiver.Name,
						PhoneNumber:  req.Receiver.Phone,
						EmailAddress: req.Receiver.Email,
					},
					Address: fedexsdk.Address{
						StreetLines: []string{req.Receiver.Street},
						City:        req.Receiver.City,
						PostalCode:  req.Receiver.PostalCode,
						CountryCode: req.Receiver.Country,
					},
				},
			},
			RequestedPackageLineItems: []fedexsdk.PackageLineItem{
				{
					Weight: fedexsdk.Weight{
						Units: "KG",
						Value: req.Parcel.WeightKg,
					},
				},
			},
			LabelSpecification: fedexsdk.LabelSpecification{
				LabelFormatType: "COMMON2D",
				ImageType:       "PDF",
				LabelStockType:  "PAPER_4X6",
			},
		},
	}

	// Add dimensions if provided
	if req.Parcel.WidthCm > 0 || req.Parcel.HeightCm > 0 || req.Parcel.DepthCm > 0 {
		fedexReq.RequestedShipment.RequestedPackageLineItems[0].Dimensions = &fedexsdk.Dimensions{
			Units:  "CM",
			Length: req.Parcel.DepthCm,
			Width:  req.Parcel.WidthCm,
			Height: req.Parcel.HeightCm,
		}
	}

	// Add COD if specified
	if req.CODAmount > 0 {
		currency := req.CODCurrency
		if currency == "" {
			currency = "PLN"
		}
		fedexReq.RequestedShipment.ShipmentSpecialServices = &fedexsdk.ShipmentSpecialServices{
			SpecialServiceTypes: []string{"COD"},
			CodDetail: &fedexsdk.CODDetail{
				CodCollectionAmount: fedexsdk.Money{
					Amount:   req.CODAmount,
					Currency: currency,
				},
			},
		}
	}

	// Add reference if specified
	if req.Reference != "" {
		fedexReq.RequestedShipment.CustomerReferences = []fedexsdk.CustomerReference{
			{
				CustomerReferenceType: "CUSTOMER_REFERENCE",
				Value:                 req.Reference,
			},
		}
	}

	resp, err := p.client.Shipments.Create(ctx, fedexReq)
	if err != nil {
		return nil, fmt.Errorf("fedex: create shipment: %w", err)
	}

	result := &integration.CarrierShipmentResponse{
		Status: "OC", // Shipment information sent
	}

	if len(resp.Output.TransactionShipments) > 0 {
		ts := resp.Output.TransactionShipments[0]
		result.ExternalID = ts.MasterTrackingNumber
		result.TrackingNumber = ts.MasterTrackingNumber

		// Extract label URL from piece responses
		if len(ts.PieceResponses) > 0 {
			for _, doc := range ts.PieceResponses[0].PackageDocuments {
				if doc.URL != "" {
					result.LabelURL = doc.URL
					break
				}
				if doc.EncodedLabel != "" {
					result.LabelURL = "data:application/pdf;base64," + doc.EncodedLabel
					break
				}
			}
		}
	}

	return result, nil
}

func (p *FedExProvider) GetLabel(ctx context.Context, externalID string, format string) ([]byte, error) {
	data, err := p.client.Shipments.GetLabel(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("fedex: get label: %w", err)
	}
	return data, nil
}

func (p *FedExProvider) GetTracking(ctx context.Context, trackingNumber string) ([]integration.TrackingEvent, error) {
	resp, err := p.client.Shipments.GetTracking(ctx, trackingNumber)
	if err != nil {
		return nil, fmt.Errorf("fedex: get tracking: %w", err)
	}

	var events []integration.TrackingEvent
	for _, result := range resp.Output.CompleteTrackResults {
		for _, tr := range result.TrackResults {
			for _, scan := range tr.ScanEvents {
				ts, _ := time.Parse("2006-01-02T15:04:05-07:00", scan.Date)
				location := scan.ScanLocation.City
				if scan.ScanLocation.CountryCode != "" && location != "" {
					location += ", " + scan.ScanLocation.CountryCode
				}
				events = append(events, integration.TrackingEvent{
					Status:    scan.EventType,
					Location:  location,
					Timestamp: ts,
					Details:   scan.EventDescription,
				})
			}
		}
	}

	return events, nil
}

func (p *FedExProvider) CancelShipment(ctx context.Context, externalID string) error {
	return p.client.Shipments.Cancel(ctx, externalID)
}

func (p *FedExProvider) MapStatus(carrierStatus string) (string, bool) {
	return fedexsdk.MapStatus(carrierStatus)
}

func (p *FedExProvider) SupportsPickupPoints() bool { return false }

func (p *FedExProvider) SearchPickupPoints(ctx context.Context, query string) ([]integration.PickupPoint, error) {
	return nil, nil
}
