package service

import (
	"testing"

	"github.com/google/uuid"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
	"github.com/stretchr/testify/assert"
)

func TestAllegroImportService_New(t *testing.T) {
	svc := NewAllegroImportService(nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.logger)
}

func TestMapAllegroOfferToProduct(t *testing.T) {
	tenantID := uuid.New()
	offer := allegrosdk.Offer{
		ID:   "offer-123",
		Name: "Test Product",
		SellingMode: &allegrosdk.OfferSellingMode{
			Price: allegrosdk.Amount{
				Amount:   "99.99",
				Currency: "PLN",
			},
			Format: "BUY_NOW",
		},
		Stock: &allegrosdk.OfferStock{
			Available: 42,
			Unit:      "UNIT",
		},
		PrimaryImage: &allegrosdk.OfferImage{
			URL: "https://example.com/image.jpg",
		},
		External: &allegrosdk.OfferExternal{
			ID: "SKU-ABC-123",
		},
	}

	product := mapAllegroOfferToProduct(offer, tenantID)

	assert.Equal(t, tenantID, product.TenantID)
	assert.Equal(t, "Test Product", product.Name)
	assert.Equal(t, "allegro", product.Source)
	assert.NotEqual(t, uuid.Nil, product.ID)

	// ExternalID should be the seller's SKU, not the Allegro offer ID.
	assert.NotNil(t, product.ExternalID)
	assert.Equal(t, "SKU-ABC-123", *product.ExternalID)

	assert.InDelta(t, 99.99, product.Price, 0.001)
	assert.Equal(t, 42, product.StockQuantity)

	assert.NotNil(t, product.ImageURL)
	assert.Equal(t, "https://example.com/image.jpg", *product.ImageURL)

	assert.NotNil(t, product.SKU)
	assert.Equal(t, "SKU-ABC-123", *product.SKU)
}

func TestMapAllegroOfferToProduct_NilFields(t *testing.T) {
	tenantID := uuid.New()
	offer := allegrosdk.Offer{
		ID:   "offer-456",
		Name: "Minimal Offer",
		// SellingMode, Stock, PrimaryImage, External all nil
	}

	product := mapAllegroOfferToProduct(offer, tenantID)

	assert.Equal(t, tenantID, product.TenantID)
	assert.Equal(t, "Minimal Offer", product.Name)
	assert.Equal(t, "allegro", product.Source)
	assert.NotEqual(t, uuid.Nil, product.ID)

	// No External.ID — ExternalID should be nil.
	assert.Nil(t, product.ExternalID)

	assert.Equal(t, float64(0), product.Price)
	assert.Equal(t, 0, product.StockQuantity)
	assert.Nil(t, product.ImageURL)
	assert.Nil(t, product.SKU)
}

func TestMapAllegroOfferToProduct_WithSKU(t *testing.T) {
	tenantID := uuid.New()
	offer := allegrosdk.Offer{
		ID:   "offer-789",
		Name: "SKU Offer",
		External: &allegrosdk.OfferExternal{
			ID: "MY-SKU-001",
		},
	}

	product := mapAllegroOfferToProduct(offer, tenantID)

	assert.NotNil(t, product.SKU)
	assert.Equal(t, "MY-SKU-001", *product.SKU)
	assert.NotNil(t, product.ExternalID)
	assert.Equal(t, "MY-SKU-001", *product.ExternalID)
	assert.Equal(t, "allegro", product.Source)
}

func TestMapAllegroOfferToProduct_EmptyExternalID(t *testing.T) {
	tenantID := uuid.New()
	offer := allegrosdk.Offer{
		ID:   "offer-empty",
		Name: "No External ID",
		External: &allegrosdk.OfferExternal{
			ID: "",
		},
	}

	product := mapAllegroOfferToProduct(offer, tenantID)

	// External is present but ID is empty — SKU and ExternalID should not be set.
	assert.Nil(t, product.SKU)
	assert.Nil(t, product.ExternalID)
}

func TestMapAllegroOfferToProduct_InvalidPrice(t *testing.T) {
	tenantID := uuid.New()
	offer := allegrosdk.Offer{
		ID:   "offer-bad-price",
		Name: "Bad Price Offer",
		SellingMode: &allegrosdk.OfferSellingMode{
			Price: allegrosdk.Amount{
				Amount:   "not-a-number",
				Currency: "PLN",
			},
		},
	}

	product := mapAllegroOfferToProduct(offer, tenantID)

	// Price should remain zero when parsing fails.
	assert.Equal(t, float64(0), product.Price)
	// No External.ID — ExternalID should be nil.
	assert.Nil(t, product.ExternalID)
}
