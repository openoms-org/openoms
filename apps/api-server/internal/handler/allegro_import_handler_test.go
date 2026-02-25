package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

func TestAllegroImportHandler_MissingTenantContext(t *testing.T) {
	h := NewAllegroHandler(nil, nil, nil, nil)

	// No tenant ID in context — should return 401
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/import-offers", nil)
	rr := httptest.NewRecorder()

	h.ImportOffers(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAllegroImportHandler_NilImportService(t *testing.T) {
	// allegroImportService is nil — calling ImportOffers with valid tenant should panic
	// because the handler dereferences h.allegroImportService. The Recoverer middleware
	// catches panics in production. Here we verify that the handler reaches the service
	// call (tenant validation passed) by expecting a panic.
	h := NewAllegroHandler(nil, nil, nil, nil)

	tenantID := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/import-offers", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	assert.Panics(t, func() {
		h.ImportOffers(rr, req)
	})
}
