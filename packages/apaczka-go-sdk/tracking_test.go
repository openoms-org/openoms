package apaczka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTracking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "tracking/123456789012/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"tracking": [
					{
						"status": "IN_TRANSIT",
						"status_original": "IN_TRANSIT_HUB",
						"description": "Parcel in transit",
						"place": "Warsaw Hub",
						"updated_at": "2024-01-15T10:30:00Z"
					},
					{
						"status": "DELIVERED",
						"status_original": null,
						"description": "Parcel delivered",
						"place": "Krakow",
						"updated_at": "2024-01-16T14:00:00Z"
					}
				]
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	resp, err := c.GetTracking(context.Background(), "123456789012")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}

	if len(resp.Tracking) != 2 {
		t.Fatalf("len(Tracking) = %d, want 2", len(resp.Tracking))
	}

	evt := resp.Tracking[0]
	if evt.Status != "IN_TRANSIT" {
		t.Errorf("Tracking[0].Status = %q, want IN_TRANSIT", evt.Status)
	}
	if evt.StatusOriginal != "IN_TRANSIT_HUB" {
		t.Errorf("Tracking[0].StatusOriginal = %q, want IN_TRANSIT_HUB", evt.StatusOriginal)
	}
	if evt.Description != "Parcel in transit" {
		t.Errorf("Tracking[0].Description = %q, want \"Parcel in transit\"", evt.Description)
	}
	if evt.Place != "Warsaw Hub" {
		t.Errorf("Tracking[0].Place = %q, want \"Warsaw Hub\"", evt.Place)
	}
	if evt.UpdatedAt != "2024-01-15T10:30:00Z" {
		t.Errorf("Tracking[0].UpdatedAt = %q, want 2024-01-15T10:30:00Z", evt.UpdatedAt)
	}

	evt2 := resp.Tracking[1]
	if evt2.Status != "DELIVERED" {
		t.Errorf("Tracking[1].Status = %q, want DELIVERED", evt2.Status)
	}
}

// TestGetTrackingNullStatusOriginal ensures a JSON null for status_original
// is decoded without error (marshalled as empty string).
func TestGetTrackingNullStatusOriginal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"tracking": [
					{
						"status": "DELIVERED",
						"status_original": null,
						"description": "done",
						"place": "city",
						"updated_at": null
					}
				]
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	resp, err := c.GetTracking(context.Background(), "WAYBILL")
	if err != nil {
		t.Fatalf("GetTracking() with null fields error: %v", err)
	}
	if len(resp.Tracking) != 1 {
		t.Fatalf("len(Tracking) = %d, want 1", len(resp.Tracking))
	}
	// null status_original decodes to empty string (Go zero value)
	if resp.Tracking[0].StatusOriginal != "" {
		t.Errorf("StatusOriginal = %q, want empty", resp.Tracking[0].StatusOriginal)
	}
}

func TestGetTrackingAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":404,"message":"waybill not found"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.GetTracking(context.Background(), "NONEXISTENT")
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
