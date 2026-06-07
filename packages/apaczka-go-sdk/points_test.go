package apaczka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "points/INPOST/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"status": 200,
			"response": {
				"points": [
					{
						"id": "KRA01N",
						"name": "Paczkomat Krakow 01",
						"line1": "ul. Testowa 1",
						"postal_code": "30-001",
						"city": "Krakow",
						"country": "PL",
						"lat": 50.06143,
						"lng": 19.93658
					},
					{
						"id": "WAW01N",
						"name": "Paczkomat Warszawa 01",
						"line1": "ul. Centralna 5",
						"postal_code": "00-001",
						"city": "Warszawa",
						"country": "PL",
						"lat": 52.22977,
						"lng": 21.01178
					}
				]
			}
		}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	points, err := c.SearchPoints(context.Background(), "INPOST", "PL", "")
	if err != nil {
		t.Fatalf("SearchPoints() error: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}

	p := points[0]
	if p.ID != "KRA01N" {
		t.Errorf("points[0].ID = %q, want KRA01N", p.ID)
	}
	if p.Name != "Paczkomat Krakow 01" {
		t.Errorf("points[0].Name = %q, want \"Paczkomat Krakow 01\"", p.Name)
	}
	if p.PostalCode != "30-001" {
		t.Errorf("points[0].PostalCode = %q, want 30-001", p.PostalCode)
	}
	if p.City != "Krakow" {
		t.Errorf("points[0].City = %q, want Krakow", p.City)
	}
	if p.Lat != 50.06143 {
		t.Errorf("points[0].Lat = %f, want 50.06143", p.Lat)
	}
	if p.Lng != 19.93658 {
		t.Errorf("points[0].Lng = %f, want 19.93658", p.Lng)
	}

	p2 := points[1]
	if p2.ID != "WAW01N" {
		t.Errorf("points[1].ID = %q, want WAW01N", p2.ID)
	}
}

func TestSearchPointsDifferentTypes(t *testing.T) {
	tests := []struct {
		pointType string
	}{
		{"INPOST"},
		{"UPS"},
		{"POCZTA"},
	}

	for _, tc := range tests {
		t.Run(tc.pointType, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "points/"+tc.pointType+"/") {
					t.Errorf("expected path to contain points/%s/, got %s", tc.pointType, r.URL.Path)
				}
				fmt.Fprint(w, `{"status":200,"response":{"points":[]}}`)
			}))
			defer srv.Close()

			c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
			pts, err := c.SearchPoints(context.Background(), tc.pointType, "PL", "")
			if err != nil {
				t.Fatalf("SearchPoints(%q) error: %v", tc.pointType, err)
			}
			if pts == nil {
				t.Error("expected non-nil slice (may be empty)")
			}
		})
	}
}

// TestSearchPointsSubtypeSent verifies the optional subtype is included in the
// request payload when provided.
func TestSearchPointsSubtypeSent(t *testing.T) {
	var gotRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotRequest = r.FormValue("request")
		fmt.Fprint(w, `{"status":200,"response":{"points":[]}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.SearchPoints(context.Background(), "INPOST", "PL", "PARCEL_LOCKER")
	if err != nil {
		t.Fatalf("SearchPoints() error: %v", err)
	}
	if !strings.Contains(gotRequest, `"subtype":"PARCEL_LOCKER"`) {
		t.Errorf("request payload %q does not contain subtype", gotRequest)
	}
}

// TestSearchPointsSubtypeOmitted verifies an empty subtype is omitted from the
// request payload.
func TestSearchPointsSubtypeOmitted(t *testing.T) {
	var gotRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotRequest = r.FormValue("request")
		fmt.Fprint(w, `{"status":200,"response":{"points":[]}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.SearchPoints(context.Background(), "INPOST", "PL", "")
	if err != nil {
		t.Fatalf("SearchPoints() error: %v", err)
	}
	if strings.Contains(gotRequest, "subtype") {
		t.Errorf("request payload %q should not contain subtype when empty", gotRequest)
	}
}

// TestSearchPointsNilReturnsEmptySlice verifies a null/absent points array
// yields a non-nil empty slice.
func TestSearchPointsNilReturnsEmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":200,"response":{}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	pts, err := c.SearchPoints(context.Background(), "INPOST", "PL", "")
	if err != nil {
		t.Fatalf("SearchPoints() error: %v", err)
	}
	if pts == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(pts) != 0 {
		t.Errorf("len(pts) = %d, want 0", len(pts))
	}
}

func TestSearchPointsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":400,"message":"invalid point type"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.SearchPoints(context.Background(), "INVALID", "PL", "")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Errorf("Status = %d, want 400", apiErr.Status)
	}
}
