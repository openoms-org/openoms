package btp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientInfoGetClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/ClientGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ClientInfo{
			ClientID:          "CL001",
			Name:              "Test Company",
			TaxID:             "1234567890",
			DefaultCurrencyID: "PLN",
			DefaultLangID:     "pl",
			DeliveryPoints: []ClientDeliveryPoint{
				{
					DeliveryPointID: "DP01",
					Name:            "Main Warehouse",
					AddressLine:     "ul. Testowa 1",
					City:            "Warszawa",
					PostalCode:      "00-001",
					Country:         "PL",
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	info, err := c.ClientInfo.GetClient(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ClientID != "CL001" {
		t.Fatalf("expected CL001, got %s", info.ClientID)
	}
	if info.Name != "Test Company" {
		t.Fatalf("expected Test Company, got %s", info.Name)
	}
	if len(info.DeliveryPoints) != 1 {
		t.Fatalf("expected 1 delivery point, got %d", len(info.DeliveryPoints))
	}
	if info.DeliveryPoints[0].City != "Warszawa" {
		t.Fatalf("expected Warszawa, got %s", info.DeliveryPoints[0].City)
	}
}

func TestClientInfoGetDeliveryModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/DeliveryModesGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DeliveryModesResponse{
			Modes: []DeliveryMode{
				{ID: "DM01", Name: "Standard", Description: "Standard delivery"},
				{ID: "DM02", Name: "Express", Description: "Express delivery"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.ClientInfo.GetDeliveryModes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(resp.Modes))
	}
}

func TestClientInfoGetCarriers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/CarriersGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CarriersResponse{
			Carriers: []Carrier{
				{ID: "DHL", Name: "DHL Express"},
				{ID: "INPOST", Name: "InPost"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.ClientInfo.GetCarriers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Carriers) != 2 {
		t.Fatalf("expected 2 carriers, got %d", len(resp.Carriers))
	}
}

func TestClientInfoGetPaymentMethods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/PaymentMethodsGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PaymentMethodsResponse{
			PaymentMethods: []PaymentMethod{
				{ID: "PM01", Name: "Bank Transfer"},
				{ID: "PM02", Name: "Cash on Delivery"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.ClientInfo.GetPaymentMethods(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.PaymentMethods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(resp.PaymentMethods))
	}
}

func TestClientInfoGetCountryGovAreas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/CountryGovAreasGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if q := r.URL.Query().Get("countryId"); q != "PL" {
			t.Fatalf("expected countryId=PL, got %s", q)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GovAreasResponse{
			GovAreas: []CountryGovArea{
				{CountryID: "PL", GovAreaID: "14", Name: "mazowieckie", Enabled: true},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.ClientInfo.GetCountryGovAreas(context.Background(), "PL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GovAreas) != 1 {
		t.Fatalf("expected 1 area, got %d", len(resp.GovAreas))
	}
	if resp.GovAreas[0].Name != "mazowieckie" {
		t.Fatalf("expected mazowieckie, got %s", resp.GovAreas[0].Name)
	}
}
