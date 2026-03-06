package kaufland

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnitUpdateStock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/units/12345/" {
			t.Errorf("path = %q, want /units/12345/", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Shop-Client-Key") == "" {
			t.Error("Shop-Client-Key is empty")
		}
		if r.Header.Get("Shop-Signature") == "" {
			t.Error("Shop-Signature is empty")
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		qty, ok := payload["quantity"]
		if !ok {
			t.Fatal("body missing 'quantity' field")
		}
		if int(qty.(float64)) != 10 {
			t.Errorf("quantity = %v, want 10", qty)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("api_key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Units.UpdateStock(context.Background(), 12345, 10)
	if err != nil {
		t.Fatalf("UpdateStock error: %v", err)
	}
}

func TestUnitUpdatePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/units/67890/" {
			t.Errorf("path = %q, want /units/67890/", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		price, ok := payload["listing_price"]
		if !ok {
			t.Fatal("body missing 'listing_price' field")
		}
		if price.(float64) != 29.99 {
			t.Errorf("listing_price = %v, want 29.99", price)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("api_key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Units.UpdatePrice(context.Background(), 67890, 29.99)
	if err != nil {
		t.Fatalf("UpdatePrice error: %v", err)
	}
}

func TestUnitUpdateStock_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Unit not found","code":404}`))
	}))
	defer srv.Close()

	c := NewClient("api_key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Units.UpdateStock(context.Background(), 99999, 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestUnitUpdatePrice_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"Rate limit exceeded","code":429}`))
	}))
	defer srv.Close()

	c := NewClient("api_key", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Units.UpdatePrice(context.Background(), 12345, 19.99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}
