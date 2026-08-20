package allegro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const officialCheckoutShipmentsJSON = `{
	"shipments": [
		{
			"id": "cb92efe4-1b2f-4cac-9e35-da69b0000001",
			"waybill": "605500867604760112200733",
			"carrierId": "INPOST",
			"carrierName": "Allegro Paczkomaty InPost",
			"createdAt": "2026-08-20T06:20:00.000Z"
		}
	]
}`

const officialManagedShipmentJSON = `{
	"id": "cb92efe4-1b2f-4cac-9e35-da69b0000001",
	"deliveryMethodId": "allegro-paczkomaty-inpost",
	"sender": {
		"name": "OpenOMS Sandbox",
		"street": "Główna 30",
		"postalCode": "10-200",
		"city": "Warszawa",
		"countryCode": "PL"
	},
	"receiver": {
		"name": "Ewa Testowa",
		"street": "Paczkomat",
		"postalCode": "00-001",
		"city": "Warszawa",
		"countryCode": "PL",
		"point": "WAW123"
	},
	"packages": [
		{
			"waybill": "605500867604760112200733",
			"type": "PACKAGE",
			"transportingInfo": [
				{"carrierId": "INPOST", "carrierWaybill": "605500867604760112200733"}
			]
		}
	],
	"carrier": "ALLEGRO",
	"labelFormat": "PDF"
}`

func TestManagedShipment_OfficialPackagesWaybill(t *testing.T) {
	var sh ManagedShipment
	if err := json.Unmarshal([]byte(officialManagedShipmentJSON), &sh); err != nil {
		t.Fatalf("unmarshal official shipment: %v", err)
	}
	if sh.ID != "cb92efe4-1b2f-4cac-9e35-da69b0000001" {
		t.Errorf("ID = %q", sh.ID)
	}
	if sh.PackageWaybill() != "605500867604760112200733" {
		t.Errorf("PackageWaybill = %q, want 605500867604760112200733", sh.PackageWaybill())
	}
	if sh.DeliveryMethodID != "allegro-paczkomaty-inpost" {
		t.Errorf("DeliveryMethodID = %q", sh.DeliveryMethodID)
	}
	if sh.Carrier != "ALLEGRO" {
		t.Errorf("Carrier = %q, want ALLEGRO (WzA-billed, not own-contract)", sh.Carrier)
	}
}

func TestOrderShipmentList_OfficialCheckoutEnvelope(t *testing.T) {
	var list ShipmentList
	if err := json.Unmarshal([]byte(officialCheckoutShipmentsJSON), &list); err != nil {
		t.Fatalf("unmarshal checkout shipments: %v", err)
	}
	if len(list.Shipments) != 1 {
		t.Fatalf("len = %d, want 1", len(list.Shipments))
	}
	s := list.Shipments[0]
	if s.Waybill != "605500867604760112200733" {
		t.Errorf("Waybill = %q", s.Waybill)
	}
	if s.ID != "cb92efe4-1b2f-4cac-9e35-da69b0000001" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.CarrierID != "INPOST" {
		t.Errorf("CarrierID = %q", s.CarrierID)
	}
}

func TestFindExistingWzA_ReadsCheckoutShipmentsAndHydratesUUID(t *testing.T) {
	var posts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts = append(posts, r.Method+" "+r.URL.Path)
			t.Errorf("unexpected write %s %s", r.Method, r.URL.Path)
			http.Error(w, "create is forbidden on import", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/order/checkout-forms/2d8367d0-9c96-11f1-bd08-9328d2ed1733/shipments":
			_, _ = w.Write([]byte(officialCheckoutShipmentsJSON))
		case r.Method == http.MethodGet && r.URL.Path == "/shipment-management/shipments/cb92efe4-1b2f-4cac-9e35-da69b0000001":
			_, _ = w.Write([]byte(officialManagedShipmentJSON))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	found, err := c.FindExistingWzA(context.Background(), "2d8367d0-9c96-11f1-bd08-9328d2ed1733")
	if err != nil {
		t.Fatalf("FindExistingWzA: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("len(found) = %d, want 1", len(found))
	}
	got := found[0]
	if got.ShipmentID != "cb92efe4-1b2f-4cac-9e35-da69b0000001" {
		t.Errorf("ShipmentID = %q", got.ShipmentID)
	}
	if got.Waybill != "605500867604760112200733" {
		t.Errorf("Waybill = %q", got.Waybill)
	}
	if got.Carrier != "ALLEGRO" {
		t.Errorf("Carrier = %q, want ALLEGRO", got.Carrier)
	}
	if len(posts) != 0 {
		t.Errorf("write calls = %v, want none", posts)
	}
}

func TestFindExistingWzA_EmptyCheckoutShipmentsFailsClosed(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			t.Errorf("unexpected write %s %s", r.Method, r.URL.Path)
			http.Error(w, "create is forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/order/checkout-forms/2d8367d0-9c96-11f1-bd08-9328d2ed1733/shipments" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"shipments":[]}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	found, err := c.FindExistingWzA(context.Background(), "2d8367d0-9c96-11f1-bd08-9328d2ed1733")
	if err != ErrWzANoExistingShipment {
		t.Fatalf("err = %v, want ErrWzANoExistingShipment", err)
	}
	if found != nil {
		t.Errorf("found = %+v, want nil", found)
	}
	if posts != 0 {
		t.Errorf("posts = %d, want 0", posts)
	}
}

func TestFindExistingWzA_TrackingAssignmentIDIsNotWzAGet(t *testing.T) {
	var gotShipment bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "create-commands") || r.URL.Path == "/shipment-management/label" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/order/checkout-forms/2d8367d0-9c96-11f1-bd08-9328d2ed1733/shipments" {
			_, _ = w.Write([]byte(`{"shipments":[{
				"id":"REhMOjEyMzQ1Njc4OTEwUEw=",
				"waybill":"605500867604760112200733",
				"carrierId":"INPOST",
				"createdAt":"2026-08-20T06:20:00.000Z"
			}]}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/shipment-management/shipments/") {
			gotShipment = true
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	found, err := c.FindExistingWzA(context.Background(), "2d8367d0-9c96-11f1-bd08-9328d2ed1733")
	if err != nil {
		t.Fatalf("FindExistingWzA: %v", err)
	}
	if gotShipment {
		t.Fatal("must not GET /shipment-management/shipments/{base64 tracking id}")
	}
	if len(found) != 1 || found[0].Waybill != "605500867604760112200733" {
		t.Fatalf("found = %+v", found)
	}
	if found[0].ShipmentID != "" {
		t.Errorf("ShipmentID = %q, tracking assignment id is not a WzA UUID", found[0].ShipmentID)
	}
}

func TestGetLabel_PostsShipmentIDsOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/shipment-management/label" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "create-commands") {
			t.Fatal("label download must not create")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		ids, _ := body["shipmentIds"].([]any)
		if len(ids) != 1 || ids[0] != "cb92efe4-1b2f-4cac-9e35-da69b0000001" {
			t.Errorf("shipmentIds = %#v", body["shipmentIds"])
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 wza"))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	pdf, err := c.ShipmentManagement.GetLabel(context.Background(), []string{"cb92efe4-1b2f-4cac-9e35-da69b0000001"})
	if err != nil {
		t.Fatalf("GetLabel: %v", err)
	}
	if string(pdf) != "%PDF-1.4 wza" {
		t.Errorf("pdf = %q", pdf)
	}
}
