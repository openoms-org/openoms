package apaczka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetValuation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"price_table": {
					"svc1": {"price": "999", "price_gross": "1229"},
					"svc2": {"price": "1499", "price_gross": "1844"}
				}
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))

	req := &ValuationRequest{
		Order: OrderInput{
			ServiceID: "",
			Address: AddressPair{
				Sender: Address{
					CountryCode: "PL",
					Name:        "Sender Co",
					Line1:       "ul. Testowa 1",
					PostalCode:  "00-001",
					City:        "Warszawa",
				},
				Receiver: Address{
					CountryCode: "PL",
					Name:        "Receiver",
					Line1:       "ul. Odbiorcza 5",
					PostalCode:  "30-001",
					City:        "Krakow",
				},
			},
			Shipment: []ShipmentItem{
				{Dimension1: 20, Dimension2: 15, Dimension3: 10, Weight: 1.5, ShipmentTypeCode: "PACZKA"},
			},
		},
	}

	resp, err := c.GetValuation(context.Background(), req)
	if err != nil {
		t.Fatalf("GetValuation() error: %v", err)
	}

	if len(resp.PriceTable) != 2 {
		t.Fatalf("len(PriceTable) = %d, want 2", len(resp.PriceTable))
	}

	row, ok := resp.PriceTable["svc1"]
	if !ok {
		t.Fatal("PriceTable missing key svc1")
	}
	if row.Price != "999" {
		t.Errorf("PriceTable[svc1].Price = %q, want 999", row.Price)
	}
	if row.PriceGross != "1229" {
		t.Errorf("PriceTable[svc1].PriceGross = %q, want 1229", row.PriceGross)
	}

	row2, ok := resp.PriceTable["svc2"]
	if !ok {
		t.Fatal("PriceTable missing key svc2")
	}
	if row2.Price != "1499" {
		t.Errorf("PriceTable[svc2].Price = %q, want 1499", row2.Price)
	}
}

func TestGetValuationAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":422,"message":"invalid order data"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.GetValuation(context.Background(), &ValuationRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 422 {
		t.Errorf("Status = %d, want 422", apiErr.Status)
	}
}
