package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func contextWithOwner(ctx context.Context, tenantID, userID uuid.UUID, role string) context.Context {
	ctx = newContextWithTenantAndUser(ctx, tenantID, userID)
	return context.WithValue(ctx, middleware.ClaimsKey, &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()},
		TenantID:         tenantID,
		Role:             role,
		Type:             "access",
	})
}

func TestAPITokenHandler_Create_InvalidJSON(t *testing.T) {
	h := NewAPITokenHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/api-tokens", strings.NewReader("bad"))
	req = req.WithContext(contextWithOwner(req.Context(), uuid.New(), uuid.New(), "owner"))
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAPITokenHandler_Create_RejectsNonOwner(t *testing.T) {
	svc := service.NewAPITokenService(nil, nil, nil)
	h := NewAPITokenHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/v1/api-tokens", strings.NewReader(`{"name":"cli"}`))
	req = req.WithContext(contextWithOwner(req.Context(), uuid.New(), uuid.New(), "admin"))
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "owner")
}

func TestAPITokenHandler_Create_RequiresAuth(t *testing.T) {
	h := NewAPITokenHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/api-tokens", strings.NewReader(`{"name":"cli"}`))
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
