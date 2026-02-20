// Package btp implements the SupplierProvider interface for BTP.pro wholesaler platform.
package btp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	btpsdk "github.com/openoms-org/openoms/packages/btp-go-sdk"
)

func init() {
	integration.RegisterSupplierProvider("btp", func(credentials, settings json.RawMessage) (integration.SupplierProvider, error) {
		return NewProvider(credentials, settings)
	})
}

// Credentials holds the authentication credentials for the BTP API.
type Credentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	BaseURL      string `json:"base_url,omitempty"`
	CatalogueURL string `json:"catalogue_url,omitempty"`
}

// Provider implements integration.SupplierProvider for BTP.pro wholesaler platform.
type Provider struct {
	client       *btpsdk.Client
	catalogueURL string
	logger       *slog.Logger
}

// NewProvider creates a new BTP supplier provider from encrypted credentials and settings.
func NewProvider(credentials, _ json.RawMessage) (*Provider, error) {
	var creds Credentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("btp: unmarshal credentials: %w", err)
	}
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("btp: username and password are required")
	}

	var opts []btpsdk.Option
	if creds.BaseURL != "" {
		opts = append(opts, btpsdk.WithBaseURL(creds.BaseURL))
	}

	client := btpsdk.NewClient(creds.Username, creds.Password, opts...)

	return &Provider{
		client:       client,
		catalogueURL: creds.CatalogueURL,
		logger:       slog.Default().With("provider", "btp"),
	}, nil
}

// ProviderName returns the provider identifier.
func (p *Provider) ProviderName() string { return "btp" }

// FetchProducts retrieves the full product catalogue from BTP.
// If a catalogue_url is configured, it parses the XML feed (with images, descriptions, etc.).
// Otherwise it falls back to the REST API.
func (p *Provider) FetchProducts(ctx context.Context) ([]integration.SupplierProduct, error) {
	if p.catalogueURL != "" {
		return p.fetchProductsFromXML(ctx)
	}
	return p.fetchProductsFromAPI(ctx)
}

// MapCatalogueProducts converts BTP SDK catalogue products to normalized SupplierProducts.
// Exported for reuse by the XML feed format sync path.
func MapCatalogueProducts(items []btpsdk.CatalogueProduct) []integration.SupplierProduct {
	products := make([]integration.SupplierProduct, 0, len(items))
	for _, item := range items {
		sp := integration.SupplierProduct{
			ExternalID:  item.SupplierItemCode,
			Name:        item.ItemDescription,
			EAN:         normalizeEAN(item.EAN),
			SKU:         item.ManufacturerItemCode,
			Price:       item.UnitNetPrice,
			RetailPrice: item.UnitRetailPrice,
			Brand:       item.BrandName,
			Weight:      item.Weight,
			Attributes:  map[string]string{},
		}

		// Description: prefer long description, fall back to short
		if item.LongItemDescription != "" {
			sp.Description = item.LongItemDescription
		} else if item.ItemDescription != "" {
			sp.Description = item.ItemDescription
		}

		// Category: "PrimaryProductGroup > ProductGroup"
		switch {
		case item.PrimaryProductGroup != "" && item.ProductGroup != "":
			sp.Category = item.PrimaryProductGroup + " > " + item.ProductGroup
		case item.PrimaryProductGroup != "":
			sp.Category = item.PrimaryProductGroup
		case item.ProductGroup != "":
			sp.Category = item.ProductGroup
		}

		// Images
		if len(item.Pictures) > 0 {
			sp.ImageURL = item.Pictures[0]
			sp.Images = item.Pictures
		}

		// Extra attributes
		if item.TaxRate > 0 {
			sp.Attributes["tax_rate"] = fmt.Sprintf("%.1f", item.TaxRate)
		}
		if item.CustomsTariffNumber != "" {
			sp.Attributes["cn_code"] = item.CustomsTariffNumber
		}
		if item.Guarantee > 0 {
			sp.Attributes["guarantee_months"] = fmt.Sprintf("%d", item.Guarantee)
		}
		maps.Copy(sp.Attributes, item.Attributes)

		products = append(products, sp)
	}
	return products
}

// fetchProductsFromXML parses the BTP XML catalogue feed for full product data.
func (p *Provider) fetchProductsFromXML(ctx context.Context) ([]integration.SupplierProduct, error) {
	httpClient := netutil.SafeHTTPClient(60 * time.Second)

	items, err := btpsdk.ParseCatalogueURL(ctx, p.catalogueURL, httpClient)
	if err != nil {
		return nil, fmt.Errorf("btp: fetch XML catalogue: %w", err)
	}

	products := MapCatalogueProducts(items)
	p.logger.Info("fetched product catalogue from XML", "count", len(products))
	return products, nil
}

// fetchProductsFromAPI retrieves products from the BTP REST API (no images/descriptions).
func (p *Provider) fetchProductsFromAPI(ctx context.Context) ([]integration.SupplierProduct, error) {
	catalogue, err := p.client.Catalogue.GetCatalogue(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("btp: fetch catalogue: %w", err)
	}

	products := make([]integration.SupplierProduct, 0, len(catalogue.Lines))
	for _, line := range catalogue.Lines {
		sp := integration.SupplierProduct{
			ExternalID:  line.ItemID,
			Name:        line.ItemName,
			Description: line.ItemDescription,
			EAN:         normalizeEAN(line.EAN),
			SKU:         line.ManufacturerItemCode,
			Price:       line.UnitNetPrice,
			RetailPrice: line.UnitRetailPrice,
			Attributes:  map[string]string{},
		}

		if len(line.CategoryNodes) > 0 {
			names := make([]string, len(line.CategoryNodes))
			for i, cn := range line.CategoryNodes {
				names[i] = cn.CategoryNodeName
			}
			sp.Category = strings.Join(names, " > ")
		}

		if len(line.BrandNodes) > 0 {
			sp.Brand = line.BrandNodes[0].BrandNodeName
		}

		if line.UnitOfMeasure != "" {
			sp.Attributes["unit_of_measure"] = line.UnitOfMeasure
		}

		// Derive stock from availability if present
		if line.Availability != nil {
			totalStock := 0
			for _, s := range line.Availability.StockSchedules {
				totalStock += int(s.Quantity)
			}
			sp.StockQuantity = totalStock
		}

		products = append(products, sp)
	}

	p.logger.Info("fetched product catalogue", "count", len(products))
	return products, nil
}

// FetchInventory retrieves current inventory levels and prices from BTP.
// Recommended refresh: 4-24 times per day.
func (p *Provider) FetchInventory(ctx context.Context) ([]integration.SupplierProduct, error) {
	report, err := p.client.Inventory.GetReport(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("btp: fetch inventory: %w", err)
	}

	products := make([]integration.SupplierProduct, 0, len(report.Lines))
	for _, line := range report.Lines {
		stock := 0
		if line.QuantityOnHand != nil {
			stock = int(*line.QuantityOnHand)
		} else if line.ActualStock != nil {
			stock = int(*line.ActualStock)
		}

		sp := integration.SupplierProduct{
			ExternalID:    line.ItemID,
			Name:          line.ItemName,
			EAN:           normalizeEAN(line.EAN),
			SKU:           line.ManufacturerItemCode,
			Price:         line.UnitNetPrice,
			RetailPrice:   line.UnitRetailPrice,
			StockQuantity: stock,
		}

		products = append(products, sp)
	}

	p.logger.Info("fetched inventory report", "count", len(products))
	return products, nil
}

// CreateOrder places an order with the BTP wholesaler.
func (p *Provider) CreateOrder(ctx context.Context, req integration.SupplierOrderRequest) (*integration.SupplierOrderResult, error) {
	btpReq := &btpsdk.OrderRequest{
		ClientOrderNumber: req.ClientOrderNumber,
		DeliveryPointID:   req.DeliveryPointID,
		CarrierID:         req.CarrierID,
		Note:              req.Note,
		IsTesting:         req.IsTesting,
	}

	for _, line := range req.Lines {
		btpLine := btpsdk.OrderRequestLine{
			BarCode:  line.EAN,
			ItemID:   line.ItemID,
			Quantity: line.Quantity,
		}
		btpReq.Lines = append(btpReq.Lines, btpLine)
	}

	resp, err := p.client.Orders.Create(ctx, btpReq)
	if err != nil {
		return nil, fmt.Errorf("btp: create order: %w", err)
	}

	p.logger.Info("order created", "order_id", resp.OrderID, "order_number", resp.OrderNumber)

	return &integration.SupplierOrderResult{
		ExternalOrderID: resp.OrderID,
		OrderNumber:     resp.OrderNumber,
		OrderDate:       resp.OrderDate,
	}, nil
}

// normalizeEAN returns empty string for placeholder EANs.
func normalizeEAN(ean string) string {
	if ean == "1111111111111" {
		return ""
	}
	return ean
}
