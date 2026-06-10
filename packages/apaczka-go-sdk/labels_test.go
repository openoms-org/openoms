package apaczka

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetWaybill(t *testing.T) {
	// Simulate a minimal PDF header as raw bytes.
	rawPDF := []byte("%PDF-1.4 fake label content")
	encoded := base64.StdEncoding.EncodeToString(rawPDF)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "waybill/ORD-123/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"status":200,"response":{"waybill":%q,"type":"pdf"}}`, encoded)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	data, err := c.GetWaybill(context.Background(), "ORD-123")
	if err != nil {
		t.Fatalf("GetWaybill() error: %v", err)
	}

	if string(data) != string(rawPDF) {
		t.Errorf("GetWaybill() returned %q, want %q", data, rawPDF)
	}
}

func TestGetWaybillInvalidBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":200,"response":{"waybill":"!!!not-base64!!!","type":"pdf"}}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.GetWaybill(context.Background(), "ORD-BAD")
	if err == nil {
		t.Fatal("expected base64 decode error")
	}
	if !strings.Contains(err.Error(), "base64") {
		t.Errorf("error %q does not mention base64", err.Error())
	}
}

func TestGetWaybillAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":404,"message":"order not found"}`)
	}))
	defer srv.Close()

	c := NewClient("a", "b", WithBaseURL(srv.URL+"/"), WithNow(fixedNow))
	_, err := c.GetWaybill(context.Background(), "NONEXISTENT")
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
