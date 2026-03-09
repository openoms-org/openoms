package ebay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var offer Offer
		if err := json.Unmarshal(body, &offer); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if offer.SKU != "SKU-001" {
			t.Errorf("sku = %q, want SKU-001", offer.SKU)
		}
		if offer.MarketplaceID != "EBAY_PL" {
			t.Errorf("marketplaceId = %q, want EBAY_PL", offer.MarketplaceID)
		}
		if offer.Format != "FIXED_PRICE" {
			t.Errorf("format = %q, want FIXED_PRICE", offer.Format)
		}
		if offer.CategoryID != "12345" {
			t.Errorf("categoryId = %q, want 12345", offer.CategoryID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateOfferResponse{OfferID: "OFFER-NEW-001"})
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Offers.CreateOffer(context.Background(), Offer{
		SKU:           "SKU-001",
		MarketplaceID: "EBAY_PL",
		Format:        "FIXED_PRICE",
		CategoryID:    "12345",
		PricingSummary: &OfferPricing{
			Price: Amount{Value: "99.99", Currency: "PLN"},
		},
		ListingPolicies: &ListingPolicies{
			FulfillmentPolicyID: "FP-1",
			ReturnPolicyID:      "RP-1",
			PaymentPolicyID:     "PP-1",
		},
	})
	if err != nil {
		t.Fatalf("CreateOffer error: %v", err)
	}
	if result.OfferID != "OFFER-NEW-001" {
		t.Errorf("OfferID = %q, want OFFER-NEW-001", result.OfferID)
	}
}

func TestGetOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer/12345" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer/12345", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		resp := Offer{
			OfferID:       "12345",
			SKU:           "SKU-GET",
			MarketplaceID: "EBAY_PL",
			Format:        "FIXED_PRICE",
			CategoryID:    "999",
			Status:        "PUBLISHED",
			Listing:       &OfferListing{ListingID: "LISTING-555"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Offers.GetOffer(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetOffer error: %v", err)
	}
	if result.OfferID != "12345" {
		t.Errorf("OfferID = %q, want 12345", result.OfferID)
	}
	if result.SKU != "SKU-GET" {
		t.Errorf("SKU = %q, want SKU-GET", result.SKU)
	}
	if result.Status != "PUBLISHED" {
		t.Errorf("Status = %q, want PUBLISHED", result.Status)
	}
	if result.Listing == nil || result.Listing.ListingID != "LISTING-555" {
		t.Errorf("Listing.ListingID = %v, want LISTING-555", result.Listing)
	}
}

func TestGetOffers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer", r.URL.Path)
		}
		if r.URL.Query().Get("sku") != "SKU-FILTER" {
			t.Errorf("sku = %q, want SKU-FILTER", r.URL.Query().Get("sku"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", r.URL.Query().Get("limit"))
		}

		resp := OffersResponse{
			Offers: []Offer{
				{OfferID: "O-1", SKU: "SKU-FILTER", MarketplaceID: "EBAY_PL", Format: "FIXED_PRICE", CategoryID: "100"},
				{OfferID: "O-2", SKU: "SKU-FILTER", MarketplaceID: "EBAY_PL", Format: "FIXED_PRICE", CategoryID: "200"},
			},
			Total: 2,
			Size:  2,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Offers.GetOffers(context.Background(), "SKU-FILTER", 10, 0)
	if err != nil {
		t.Fatalf("GetOffers error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if len(result.Offers) != 2 {
		t.Fatalf("len(Offers) = %d, want 2", len(result.Offers))
	}
	if result.Offers[0].OfferID != "O-1" {
		t.Errorf("Offers[0].OfferID = %q, want O-1", result.Offers[0].OfferID)
	}
	if result.Offers[1].OfferID != "O-2" {
		t.Errorf("Offers[1].OfferID = %q, want O-2", result.Offers[1].OfferID)
	}
}

func TestPublishOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer/OFFER-PUB/publish" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer/OFFER-PUB/publish", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PublishResponse{ListingID: "LISTING-9999"})
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Offers.PublishOffer(context.Background(), "OFFER-PUB")
	if err != nil {
		t.Fatalf("PublishOffer error: %v", err)
	}
	if result.ListingID != "LISTING-9999" {
		t.Errorf("ListingID = %q, want LISTING-9999", result.ListingID)
	}
}

func TestWithdrawOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer/OFFER-WD/withdraw" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer/OFFER-WD/withdraw", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Offers.WithdrawOffer(context.Background(), "OFFER-WD")
	if err != nil {
		t.Fatalf("WithdrawOffer error: %v", err)
	}
}

func TestDeleteOffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/offer/OFFER-DEL" {
			t.Errorf("path = %q, want /sell/inventory/v1/offer/OFFER-DEL", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Offers.DeleteOffer(context.Background(), "OFFER-DEL")
	if err != nil {
		t.Fatalf("DeleteOffer error: %v", err)
	}
}
