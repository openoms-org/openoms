package btp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaybillsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/WaybillsGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req WaybillsRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.WaybillNumbers) != 2 {
			t.Fatalf("expected 2 waybill numbers, got %d", len(req.WaybillNumbers))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(WaybillsResponse{
			Waybills: []Waybill{
				{WaybillNumber: "WB001", OrderNumber: "ORD001", CarrierName: "DHL", TrackingNumber: "TRACK001"},
			},
			NotFoundWaybills: []string{"WB999"},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Waybills.Get(context.Background(), &WaybillsRequest{
		WaybillNumbers: []string{"WB001", "WB999"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Waybills) != 1 {
		t.Fatalf("expected 1 waybill, got %d", len(resp.Waybills))
	}
	if resp.Waybills[0].CarrierName != "DHL" {
		t.Fatalf("expected DHL, got %s", resp.Waybills[0].CarrierName)
	}
	if len(resp.NotFoundWaybills) != 1 || resp.NotFoundWaybills[0] != "WB999" {
		t.Fatal("expected WB999 in not found")
	}
}

func TestWaybillConfigureNotify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/WaybillNotifyConfigure" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResult{ResultType: "INFORMATION"})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	result, err := c.Waybills.ConfigureNotify(context.Background(), &DocumentNotifyConfig{
		EndpointURL: "https://example.com/waybill-hook",
		HTTPMethod:  "POST",
		Mode:        "KeyOnly",
		Format:      "JSON",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResultType != "INFORMATION" {
		t.Fatalf("expected INFORMATION, got %s", result.ResultType)
	}
}
