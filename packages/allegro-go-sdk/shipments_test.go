package allegro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const officialMiniKurierServiceJSON = `{
	"services": [
		{
			"id": {
				"deliveryMethodId": "c3066682-97a3-42fe-9eb5-3beeccab840c",
				"credentialsId": null
			},
			"name": "Allegro miniKurier24 InPost",
			"carrierId": "INPOST",
			"owner": "ALLEGRO"
		}
	]
}`

func TestDeliveryServiceList_OfficialServicesKeyListsOneService(t *testing.T) {
	var list DeliveryServiceList
	if err := json.Unmarshal([]byte(officialMiniKurierServiceJSON), &list); err != nil {
		t.Fatalf("unmarshal official services envelope: %v", err)
	}
	if len(list.DeliveryServices) != 1 {
		t.Fatalf("len(DeliveryServices) = %d, want 1", len(list.DeliveryServices))
	}
	svc := list.DeliveryServices[0]
	if svc.ID != "c3066682-97a3-42fe-9eb5-3beeccab840c" {
		t.Errorf("ID = %q, want deliveryMethodId from DeliveryServiceIdDto", svc.ID)
	}
	if svc.Name != "Allegro miniKurier24 InPost" {
		t.Errorf("Name = %q, want %q", svc.Name, "Allegro miniKurier24 InPost")
	}
	if svc.CarrierID != "INPOST" {
		t.Errorf("CarrierID = %q, want INPOST", svc.CarrierID)
	}
	if svc.Owner != "ALLEGRO" {
		t.Errorf("Owner = %q, want ALLEGRO", svc.Owner)
	}
}

func TestDeliveryServiceList_EmptyServicesStaysEmpty(t *testing.T) {
	var list DeliveryServiceList
	if err := json.Unmarshal([]byte(`{"services":[]}`), &list); err != nil {
		t.Fatalf("unmarshal empty services: %v", err)
	}
	if len(list.DeliveryServices) != 0 {
		t.Fatalf("len(DeliveryServices) = %d, want 0", len(list.DeliveryServices))
	}
}

func TestDeliveryServiceList_DeliveryServicesFallback(t *testing.T) {
	const legacy = `{
		"deliveryServices": [
			{"id":"legacy-1","name":"Legacy WzA","carrierId":"INPOST","owner":"CLIENT"}
		]
	}`
	var list DeliveryServiceList
	if err := json.Unmarshal([]byte(legacy), &list); err != nil {
		t.Fatalf("unmarshal deliveryServices fallback: %v", err)
	}
	if len(list.DeliveryServices) != 1 {
		t.Fatalf("len(DeliveryServices) = %d, want 1", len(list.DeliveryServices))
	}
	svc := list.DeliveryServices[0]
	if svc.ID != "legacy-1" {
		t.Errorf("ID = %q, want legacy-1", svc.ID)
	}
	if svc.Name != "Legacy WzA" {
		t.Errorf("Name = %q, want Legacy WzA", svc.Name)
	}
	if svc.Owner != "CLIENT" {
		t.Errorf("Owner = %q, want CLIENT", svc.Owner)
	}
}

func TestDeliveryService_IDAcceptsString(t *testing.T) {
	var svc DeliveryService
	if err := json.Unmarshal([]byte(`{"id":"bare-string-id","name":"X","carrierId":"INPOST"}`), &svc); err != nil {
		t.Fatalf("unmarshal string id: %v", err)
	}
	if svc.ID != "bare-string-id" {
		t.Errorf("ID = %q, want bare-string-id", svc.ID)
	}
}

func TestListDeliveryServices_OfficialEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shipment-management/delivery-services" {
			t.Errorf("path = %q, want /shipment-management/delivery-services", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(officialMiniKurierServiceJSON))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	services, err := c.ShipmentManagement.ListDeliveryServices(context.Background())
	if err != nil {
		t.Fatalf("ListDeliveryServices: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("len(services) = %d, want 1", len(services))
	}
	if services[0].Name != "Allegro miniKurier24 InPost" {
		t.Errorf("Name = %q, want Allegro miniKurier24 InPost", services[0].Name)
	}
	if services[0].ID != "c3066682-97a3-42fe-9eb5-3beeccab840c" {
		t.Errorf("ID = %q, want deliveryMethodId", services[0].ID)
	}
	if services[0].Owner != "ALLEGRO" {
		t.Errorf("Owner = %q, want ALLEGRO", services[0].Owner)
	}
}
