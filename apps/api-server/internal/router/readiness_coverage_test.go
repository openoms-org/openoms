package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// excludedEndpoints are non-ready-adjacent endpoints intentionally left ungated
// (public/token-auth or shared with ready surfaces). Reviewed allowlist.
var excludedEndpoints = map[string]bool{
	"/v1/stats": true, "/v1/barcode": true,
}

// endpointProbe is a representative request that exercises a non-ready feature's
// endpoint prefix. The method+path MUST match a route registered in router.go so
// that chi runs the route group's requireFeature middleware (which short-circuits
// before any handler is invoked).
type endpointProbe struct {
	method, path string
}

// endpointProbes maps every non-ready endpoint prefix in readiness.json to a
// concrete request that hits a real route under that prefix. Keeping this table
// next to the registry forces a new non-ready endpoint to get an explicit probe
// (or an excludedEndpoints entry) — that is what proves per-endpoint gating
// rather than mere requireFeature(...) text presence.
var endpointProbes = map[string]endpointProbe{
	"/v1/invoices":                          {http.MethodGet, "/v1/invoices/"},
	"/v1/products/{productId}/variants":     {http.MethodGet, "/v1/products/" + sampleID + "/variants/"},
	"/v1/products/{productId}/listings":     {http.MethodGet, "/v1/products/" + sampleID + "/listings/"},
	"/v1/warehouses":                        {http.MethodGet, "/v1/warehouses/"},
	"/v1/warehouse-documents":               {http.MethodGet, "/v1/warehouse-documents/"},
	"/v1/stocktakes":                        {http.MethodGet, "/v1/stocktakes/"},
	"/v1/pick-pack/sessions":                {http.MethodGet, "/v1/pick-pack/sessions/"},
	"/v1/purchase-orders":                   {http.MethodGet, "/v1/purchase-orders/"},
	"/v1/suppliers":                         {http.MethodGet, "/v1/suppliers/"},
	"/v1/supplier-products":                 {http.MethodGet, "/v1/supplier-products"},
	"/v1/dropship-orders":                   {http.MethodGet, "/v1/dropship-orders/"},
	"/v1/orders/{order_id}/dropship":        {http.MethodPost, "/v1/orders/" + sampleID + "/dropship"},
	"/v1/orders/{order_id}/dropship-orders": {http.MethodGet, "/v1/orders/" + sampleID + "/dropship-orders"},
	"/v1/loyalty":                           {http.MethodGet, "/v1/loyalty/programs/"},
	"/v1/recurring-orders":                  {http.MethodGet, "/v1/recurring-orders/"},
	"/v1/repricing":                         {http.MethodGet, "/v1/repricing/rules"},
	"/v1/reconciliation":                    {http.MethodGet, "/v1/reconciliation/settlements"},
	"/v1/stock-sync":                        {http.MethodGet, "/v1/stock-sync/channels"},
	"/v1/listing-sync/configs":              {http.MethodGet, "/v1/listing-sync/configs/"},
	"/v1/segments":                          {http.MethodGet, "/v1/segments/"},
	"/v1/forecast":                          {http.MethodGet, "/v1/forecast/products"},
	"/v1/carbon":                            {http.MethodGet, "/v1/carbon/stats"},
	"/v1/vat-oss":                           {http.MethodGet, "/v1/vat-oss/rates"},
	"/v1/workflows":                         {http.MethodGet, "/v1/workflows/templates"},
	"/v1/automation":                        {http.MethodGet, "/v1/automation/delayed"},
	"/v1/sync-jobs":                         {http.MethodGet, "/v1/sync-jobs/"},
	"/v1/message-templates":                 {http.MethodGet, "/v1/message-templates/"},
	"/v1/ai":                                {http.MethodPost, "/v1/ai/categorize"},
	"/v1/images/remove-background":          {http.MethodPost, "/v1/images/remove-background"},
	"/v1/products/{id}/images/{index}/remove-background": {http.MethodPost, "/v1/products/" + sampleID + "/images/0/remove-background"},
	"/v1/price-lists":    {http.MethodGet, "/v1/price-lists/"},
	"/v1/exchange-rates": {http.MethodGet, "/v1/exchange-rates/"},
	"/v1/marketing":      {http.MethodGet, "/v1/marketing/status"},
}

var sampleID = "00000000-0000-0000-0000-000000000001"

// requireFeatureCallRe matches requireFeature("feature_id") calls in router.go.
var requireFeatureCallRe = regexp.MustCompile(`requireFeature\("([a-z_]+)"\)`)

// TestReadinessCoverage_EveryGateHasFeature ensures every requireFeature(...) call
// in router.go references a feature id that exists in the embedded readiness.json.
func TestReadinessCoverage_EveryGateHasFeature(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range requireFeatureCallRe.FindAllStringSubmatch(string(src), -1) {
		id := m[1]
		if _, ok := readiness.LookupFeature(id); !ok {
			t.Errorf("requireFeature(%q) has no entry in readiness.json", id)
		}
	}
}

// TestReadinessCoverage_EveryNonReadyEndpointIsGated is the keystone drift guard.
// For EVERY non-ready feature in readiness.json, for EVERY endpoint prefix it
// declares, it builds the router in client-ready mode and HTTP-probes a real
// request to that prefix, asserting the response is the feature_not_available 404.
// This proves the endpoint is actually gated at the HTTP layer (not merely that a
// requireFeature string exists somewhere in the source). An endpoint may instead
// be on the reviewed excludedEndpoints allowlist.
//
// A non-ready endpoint with neither a probe nor an exclusion fails the test,
// forcing whoever adds a new non-ready endpoint to wire its gate and prove it.
func TestReadinessCoverage_EveryNonReadyEndpointIsGated(t *testing.T) {
	r, token := buildClientReadyRouter(t)

	for id, f := range readiness.NonReadyFeatures() {
		if len(f.Endpoints) == 0 {
			t.Errorf("non-ready feature %q has no endpoints[] in readiness.json", id)
			continue
		}
		for _, ep := range f.Endpoints {
			if excludedEndpoints[ep] {
				continue
			}
			probe, ok := endpointProbes[ep]
			if !ok {
				t.Errorf("non-ready feature %q endpoint %q has no probe in endpointProbes (add a probe or an excludedEndpoints entry)", id, ep)
				continue
			}
			t.Run(id+" "+ep, func(t *testing.T) {
				rr := serveAuthed(r, token, probe.method, probe.path)
				require.Truef(t, isFeatureGate404(t, rr),
					"endpoint %q (feature %q) is NOT gated: probe %s %s returned %d body %q",
					ep, id, probe.method, probe.path, rr.Code, rr.Body.String())
			})
		}
	}
}

// buildClientReadyRouter wires the router in client-ready mode with non-nil stub
// handlers for the conditionally-registered non-ready groups so their routes mount
// and the requireFeature middleware actually runs. The readiness gate short-circuits
// with a 404 before any handler method executes, so zero-value handler pointers are
// safe (never dereferenced). It returns the router and an owner access token signed
// by the same token service the router verifies with.
func buildClientReadyRouter(t *testing.T) (http.Handler, string) {
	t.Helper()
	tokenSvc, err := service.NewTokenService("test-jwt-secret-for-readiness-coverage")
	require.NoError(t, err)

	user := model.User{ID: uuid.New(), TenantID: uuid.New(), Email: "owner@example.com", Role: "owner"}
	token, err := tokenSvc.GenerateAccessToken(user)
	require.NoError(t, err)

	r := New(RouterDeps{
		Config:          &config.Config{Env: "development", FrontendURL: "http://localhost:3000", UploadDir: t.TempDir(), APISurfaceMode: "client-ready"},
		TokenSvc:        tokenSvc,
		BGRemoval:       &handler.BGRemovalHandler{},
		AllegroListings: &handler.AllegroListingsHandler{},
		Segment:         &handler.SegmentHandler{},
		Loyalty:         &handler.LoyaltyHandler{},
		Forecast:        &handler.ForecastHandler{},
		Carbon:          &handler.CarbonHandler{},
		MessageTemplate: &handler.MessageTemplateHandler{},
		StockSync:       &handler.StockSyncHandler{},
		ListingSync:     &handler.ListingSyncHandler{},
	})
	return r, token
}

// serveAuthed sends an owner-authenticated request through the router so that
// role/permission and CSRF gates pass and the readiness feature gate is the only
// thing that can short-circuit the request with a 404. For state-changing methods
// it supplies a matching double-submit CSRF cookie+header so CSRF does not mask the
// readiness gate.
func serveAuthed(r http.Handler, token, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		const csrf = "test-csrf-token"
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrf})
		req.Header.Set("X-CSRF-Token", csrf)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}
