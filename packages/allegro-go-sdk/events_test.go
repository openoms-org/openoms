package allegro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEventsPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/events" {
			t.Errorf("path = %q, want /order/events", r.URL.Path)
		}
		if from := r.URL.Query().Get("from"); from != "evt-100" {
			t.Errorf("from = %q, want evt-100", from)
		}
		if typ := r.URL.Query().Get("type"); typ != "BOUGHT,FILLED_IN" {
			t.Errorf("type = %q, want BOUGHT,FILLED_IN", typ)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"events": [
				{
					"id": "evt-101",
					"type": "BOUGHT",
					"occurredAt": "2024-01-15T10:30:00Z",
					"order": {"checkoutForm": {"id": "order-1"}}
				},
				{
					"id": "evt-102",
					"type": "FILLED_IN",
					"occurredAt": "2024-01-15T10:31:00Z",
					"order": {"checkoutForm": {"id": "order-2"}}
				}
			]
		}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Events.Poll(context.Background(), "evt-100", "BOUGHT", "FILLED_IN")
	if err != nil {
		t.Fatalf("Events.Poll error: %v", err)
	}
	if len(list.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(list.Events))
	}
	if list.Events[0].ID != "evt-101" {
		t.Errorf("Events[0].ID = %q, want %q", list.Events[0].ID, "evt-101")
	}
	if list.Events[0].Order.CheckoutForm.ID != "order-1" {
		t.Errorf("Events[0].Order.CheckoutForm.ID = %q, want %q", list.Events[0].Order.CheckoutForm.ID, "order-1")
	}
	if list.Events[1].Type != "FILLED_IN" {
		t.Errorf("Events[1].Type = %q, want %q", list.Events[1].Type, "FILLED_IN")
	}
}

func TestEventsPollError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Events.Poll(context.Background(), "evt-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventsPollNoParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		w.Write([]byte(`{"events":[]}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Events.Poll(context.Background(), "")
	if err != nil {
		t.Fatalf("Events.Poll error: %v", err)
	}
	if len(list.Events) != 0 {
		t.Errorf("len(Events) = %d, want 0", len(list.Events))
	}
}
