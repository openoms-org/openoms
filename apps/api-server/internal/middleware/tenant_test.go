package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTenantRequired_MissingHeader(t *testing.T) {
	handler := TenantRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing X-Tenant-ID")
}

func TestTenantRequired_InvalidUUID(t *testing.T) {
	handler := TenantRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "valid UUID")
}

func TestTenantRequired_Valid(t *testing.T) {
	tenantID := uuid.New()
	var capturedID uuid.UUID

	handler := TenantRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, tenantID, capturedID)
}

func TestTenantIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, uuid.Nil, TenantIDFromContext(ctx))
}
