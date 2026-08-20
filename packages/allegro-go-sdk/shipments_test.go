package allegro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

const officialDeliveryProposalsJSON = `{
	"orderId": "19829450-9c54-11f1-bd08-9328d2ed1733",
	"suggestedInput": {
		"deliveryMethodId": "c3066682-97a3-42fe-9eb5-3beeccab840c",
		"credentialsId": null,
		"sender": {
			"name": "OpenOMS Sandbox",
			"street": "Główna 30",
			"postalCode": "10-200",
			"city": "Warszawa",
			"countryCode": "PL",
			"phone": "500600700"
		},
		"receiver": {
			"name": "Anna Kupująca",
			"street": "Marszałkowska 1",
			"postalCode": "00-001",
			"city": "Warszawa",
			"countryCode": "PL",
			"email": "anna@example.pl",
			"phone": "600700800"
		},
		"packages": [{
			"type": "PACKAGE",
			"length": {"value": 30, "unit": "CENTIMETER"},
			"width": {"value": 20, "unit": "CENTIMETER"},
			"height": {"value": 15, "unit": "CENTIMETER"},
			"weight": {"value": "1.00", "unit": "KILOGRAMS"}
		}]
	}
}`

func TestDeliveryProposals_OfficialSuggestedInput(t *testing.T) {
	var proposals DeliveryProposals
	if err := json.Unmarshal([]byte(officialDeliveryProposalsJSON), &proposals); err != nil {
		t.Fatalf("unmarshal official delivery-proposals: %v", err)
	}
	if proposals.OrderID != "19829450-9c54-11f1-bd08-9328d2ed1733" {
		t.Errorf("OrderID = %q", proposals.OrderID)
	}
	in := proposals.SuggestedInput
	if in.DeliveryMethodID != "c3066682-97a3-42fe-9eb5-3beeccab840c" {
		t.Errorf("DeliveryMethodID = %q", in.DeliveryMethodID)
	}
	if in.Sender.PostalCode != "10-200" {
		t.Errorf("Sender.PostalCode = %q, want official postalCode", in.Sender.PostalCode)
	}
	if in.Receiver.PostalCode != "00-001" {
		t.Errorf("Receiver.PostalCode = %q", in.Receiver.PostalCode)
	}
	if len(in.Packages) != 1 || in.Packages[0].Weight == nil || in.Packages[0].Weight.Value != 1 {
		t.Errorf("weight value = %+v, want 1 from string \"1.00\"", in.Packages)
	}
	if in.Packages[0].Length == nil || in.Packages[0].Length.Unit != "CENTIMETER" {
		t.Errorf("length unit = %+v, want CENTIMETER", in.Packages[0].Length)
	}
}

func TestShipmentAddress_ZipCodeFallbackAndPostalCodeMarshal(t *testing.T) {
	var addr ShipmentAddress
	if err := json.Unmarshal([]byte(`{"street":"Główna 30","city":"Warszawa","zipCode":"10-200","countryCode":"PL"}`), &addr); err != nil {
		t.Fatalf("unmarshal zipCode: %v", err)
	}
	if addr.PostalCode != "10-200" {
		t.Errorf("PostalCode = %q, want zipCode fallback", addr.PostalCode)
	}
	raw, err := json.Marshal(addr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !jsonContains(raw, `"postalCode":"10-200"`) {
		t.Errorf("marshal = %s, want postalCode for Allegro create-commands", raw)
	}
	if jsonContains(raw, `"zipCode"`) {
		t.Errorf("marshal = %s, must not send zipCode to Allegro", raw)
	}
}

func TestGetDeliveryProposals_OfficialPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shipment-management/delivery-proposals/19829450-9c54-11f1-bd08-9328d2ed1733" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(officialDeliveryProposalsJSON))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	proposals, err := c.ShipmentManagement.GetDeliveryProposals(context.Background(), "19829450-9c54-11f1-bd08-9328d2ed1733")
	if err != nil {
		t.Fatalf("GetDeliveryProposals: %v", err)
	}
	if proposals.SuggestedInput.DeliveryMethodID != "c3066682-97a3-42fe-9eb5-3beeccab840c" {
		t.Errorf("DeliveryMethodID = %q", proposals.SuggestedInput.DeliveryMethodID)
	}
}

func TestCreateShipment_PollsCommandStatusForShipmentID(t *testing.T) {
	var posts, gets int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/shipment-management/shipments/create-commands":
			posts++
			_, _ = w.Write([]byte(`{"commandId":"14e142cf-e8e0-48cc-bcf6-399b5fd90b32"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/shipment-management/shipments/create-commands/14e142cf-e8e0-48cc-bcf6-399b5fd90b32":
			gets++
			_, _ = w.Write([]byte(`{"commandId":"14e142cf-e8e0-48cc-bcf6-399b5fd90b32","status":"SUCCESS","shipmentId":"ba88f0fb-acf3-438a-877e-580da50c0874"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	resp, err := c.ShipmentManagement.CreateShipment(context.Background(), CreateShipmentCommand{
		CommandID: "14e142cf-e8e0-48cc-bcf6-399b5fd90b32",
		Input: CreateShipmentInput{
			DeliveryMethodID: "c3066682-97a3-42fe-9eb5-3beeccab840c",
			Sender:           ShipmentAddress{Street: "Główna 30", City: "Warszawa", PostalCode: "10-200", CountryCode: "PL"},
			Receiver:         ShipmentAddress{Street: "Marszałkowska 1", City: "Warszawa", PostalCode: "00-001", CountryCode: "PL"},
		},
	})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if posts != 1 || gets != 1 {
		t.Errorf("posts=%d gets=%d, want 1 and 1 (POST then poll GET)", posts, gets)
	}
	if resp.ShipmentID != "ba88f0fb-acf3-438a-877e-580da50c0874" {
		t.Errorf("ShipmentID = %q", resp.ShipmentID)
	}
	if resp.Status != "SUCCESS" {
		t.Errorf("Status = %q", resp.Status)
	}
}

func jsonContains(raw []byte, needle string) bool {
	return strings.Contains(string(raw), needle)
}
