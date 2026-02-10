package allegro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOffersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sale/offers" {
			t.Errorf("path = %q, want /sale/offers", r.URL.Path)
		}
		if lim := r.URL.Query().Get("limit"); lim != "20" {
			t.Errorf("limit = %q, want 20", lim)
		}
		if name := r.URL.Query().Get("name"); name != "Widget" {
			t.Errorf("name = %q, want Widget", name)
		}
		if ps := r.URL.Query().Get("publication.status"); ps != "ACTIVE" {
			t.Errorf("publication.status = %q, want ACTIVE", ps)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"offers": [
				{"id": "off-1", "name": "Widget Pro"},
				{"id": "off-2", "name": "Widget Lite"}
			],
			"count": 2,
			"totalCount": 50
		}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Offers.List(context.Background(), &ListOffersParams{
		Limit:             20,
		Name:              "Widget",
		PublicationStatus: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("Offers.List error: %v", err)
	}
	if list.Count != 2 {
		t.Errorf("Count = %d, want 2", list.Count)
	}
	if list.TotalCount != 50 {
		t.Errorf("TotalCount = %d, want 50", list.TotalCount)
	}
	if len(list.Offers) != 2 {
		t.Fatalf("len(Offers) = %d, want 2", len(list.Offers))
	}
	if list.Offers[0].Name != "Widget Pro" {
		t.Errorf("Offers[0].Name = %q, want %q", list.Offers[0].Name, "Widget Pro")
	}
}

func TestOffersListWithOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("offset"); q != "10" {
			t.Errorf("offset = %q, want 10", q)
		}
		w.Write([]byte(`{"offers":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.List(context.Background(), &ListOffersParams{Offset: 10})
	if err != nil {
		t.Fatalf("Offers.List error: %v", err)
	}
}

func TestOffersListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOffersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.Get(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOffersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sale/product-offers/off-abc" {
			t.Errorf("path = %q, want /sale/product-offers/off-abc", r.URL.Path)
		}
		w.Write([]byte(`{"id":"off-abc","name":"Super Widget"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	offer, err := c.Offers.Get(context.Background(), "off-abc")
	if err != nil {
		t.Fatalf("Offers.Get error: %v", err)
	}
	if offer.ID != "off-abc" {
		t.Errorf("ID = %q, want %q", offer.ID, "off-abc")
	}
	if offer.Name != "Super Widget" {
		t.Errorf("Name = %q, want %q", offer.Name, "Super Widget")
	}
}
