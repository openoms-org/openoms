package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestPlanGuard_ActivePlan_Passes(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	cache.Set(tenantID, "plus", nil)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPlanGuard_Suspended_BlocksAll(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	cache.Set(tenantID, "suspended", nil)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	assert.Contains(t, rr.Body.String(), "subscription_suspended")
}

func TestPlanGuard_PastDue_AllowsGET(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	cache.Set(tenantID, "past_due", nil)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPlanGuard_PastDue_BlocksPOST(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	cache.Set(tenantID, "past_due", nil)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	assert.Contains(t, rr.Body.String(), "payment_past_due")
}

func TestPlanGuard_FreePlan_NoLimits(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	cache.Set(tenantID, "free", nil)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPlanGuard_SubscriptionPastDue_BlocksPOST(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	settings := json.RawMessage(`{"subscription_status":"past_due"}`)
	cache.Set(tenantID, "plus", settings)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	assert.Contains(t, rr.Body.String(), "payment_past_due")
}

func TestPlanGuard_SubscriptionPastDue_AllowsGET(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	settings := json.RawMessage(`{"subscription_status":"past_due"}`)
	cache.Set(tenantID, "plus", settings)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestPlanGuard_SubscriptionCanceled_BlocksPOST(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	settings := json.RawMessage(`{"subscription_status":"canceled"}`)
	cache.Set(tenantID, "plus", settings)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	assert.Contains(t, rr.Body.String(), "subscription_inactive")
}

func TestPlanGuard_SubscriptionSuspended_BlocksAll(t *testing.T) {
	cache := service.NewPlanCache(5 * time.Minute)
	tenantID := uuid.New()
	settings := json.RawMessage(`{"subscription_status":"suspended"}`)
	cache.Set(tenantID, "plus", settings)

	mw := TenantPlanGuard(cache, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), TenantIDKey, tenantID))
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
	assert.Contains(t, rr.Body.String(), "subscription_suspended")
}
