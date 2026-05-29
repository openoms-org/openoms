package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireFeature_BlocksNonReadyInClientReady(t *testing.T) {
	h := RequireFeature("repricing", "client-ready")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/repricing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestRequireFeature_AllowsInFullMode(t *testing.T) {
	called := false
	h := RequireFeature("repricing", "full")(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/repricing", nil))
	if !called {
		t.Fatal("handler should be reached in full mode")
	}
}
