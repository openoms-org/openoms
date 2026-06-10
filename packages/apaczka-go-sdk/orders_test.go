package apaczka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sampleOrder() *CreateOrderRequest {
	return &CreateOrderRequest{
		Order: OrderInput{
			ServiceID: "svc1",
			Address: AddressPair{
				Sender: Address{
					CountryCode: "PL",
					Name:        "Sender",
					Line1:       "ul. Nadawcy 1",
					PostalCode:  "00-001",
					City:        "Warszawa",
					Phone:       "+48100200300",
					Email:       "sender@example.com",
				},
				Receiver: Address{
					CountryCode: "PL",
					Name:        "Receiver",
					Line1:       "ul. Odbiorcy 2",
					PostalCode:  "30-001",
					City:        "Krakow",
					Phone:       "+48500600700",
					Email:       "receiver@example.com",
				},
			},
			Shipment: []ShipmentItem{
				{Dimension1: 20, Dimension2: 15, Dimension3: 10, Weight: 1.5, ShipmentTypeCode: "PACZKA"},
			},
			Pickup: &Pickup{Type: "COURIER", Date: "2024-01-15"},
		},
	}
}

func TestCreateOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"order": {
					"id": "ORD-123",
					"service_id": "svc1",
					"service_name": "DHL Domestic",
					"waybill_number": "123456789012",
					"tracking_url": "https://track.example.com/123456789012",
					"status": "new"
				}
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	order, err := c.CreateOrder(context.Background(), sampleOrder())
	if err != nil {
		t.Fatalf("CreateOrder() error: %v", err)
	}

	if order.ID != "ORD-123" {
		t.Errorf("ID = %q, want ORD-123", order.ID)
	}
	if order.WaybillNumber != "123456789012" {
		t.Errorf("WaybillNumber = %q, want 123456789012", order.WaybillNumber)
	}
	if order.TrackingURL != "https://track.example.com/123456789012" {
		t.Errorf("TrackingURL = %q, want https://track.example.com/123456789012", order.TrackingURL)
	}
	if order.Status != "new" {
		t.Errorf("Status = %q, want new", order.Status)
	}
}

func TestCreateOrderWithCOD(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotBody = r.FormValue("request")
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"order": {
					"id": "ORD-COD",
					"service_id": "svc1",
					"waybill_number": "COD123",
					"tracking_url": "",
					"status": "new"
				}
			}
		}`)
	}))
	defer srv.Close()

	req := sampleOrder()
	req.Order.COD = &COD{
		Amount:      4999,
		Currency:    "PLN",
		BankAccount: "PL12345678901234567890123456",
		Country:     "PL",
	}

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	order, err := c.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateOrder() with COD error: %v", err)
	}
	if order.ID != "ORD-COD" {
		t.Errorf("ID = %q, want ORD-COD", order.ID)
	}
	// Verify COD payload was serialised
	if !strings.Contains(gotBody, `"bankaccount"`) {
		t.Errorf("request payload %q does not contain bankaccount", gotBody)
	}
}

func TestGetOrderStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path contains order ID
		if !strings.Contains(r.URL.Path, "order/ORD-123/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"order": {
					"id": "ORD-123",
					"service_id": "svc1",
					"waybill_number": "123456789012",
					"status": "shipped"
				}
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	order, err := c.GetOrderStatus(context.Background(), "ORD-123")
	if err != nil {
		t.Fatalf("GetOrderStatus() error: %v", err)
	}
	if order.ID != "ORD-123" {
		t.Errorf("ID = %q, want ORD-123", order.ID)
	}
	if order.Status != "shipped" {
		t.Errorf("Status = %q, want shipped", order.Status)
	}
}

func TestCancelOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "cancel_order/ORD-123/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Empty response body on success
		fmt.Fprint(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	err := c.CancelOrder(context.Background(), "ORD-123")
	if err != nil {
		t.Fatalf("CancelOrder() error: %v", err)
	}
}

func TestCancelOrderAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":404,"message":"order not found"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	err := c.CancelOrder(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 404 {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
}
