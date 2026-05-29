package router

import (
	"os"
	"regexp"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
)

// excludedEndpoints are non-ready-adjacent endpoints intentionally left ungated
// (public/token-auth or shared with ready surfaces). Reviewed allowlist.
var excludedEndpoints = map[string]bool{
	"/v1/stats": true, "/v1/barcode": true,
}

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

// TestReadinessCoverage_EveryNonReadyFeatureIsGated ensures every non-ready feature
// in readiness.json is gated by at least one requireFeature(...) call in router.go,
// unless every one of its endpoints is on the reviewed excludedEndpoints allowlist.
func TestReadinessCoverage_EveryNonReadyFeatureIsGated(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for id, f := range readiness.NonReadyFeatures() {
		if gatedInSource(body, id) {
			continue
		}
		if allEndpointsExcluded(f) {
			continue
		}
		t.Errorf("non-ready feature %q in readiness.json is not gated by any requireFeature(...) in router.go", id)
	}
}

// gatedInSource reports whether router.go contains a requireFeature("id") call.
func gatedInSource(body, id string) bool {
	return regexp.MustCompile(`requireFeature\("` + regexp.QuoteMeta(id) + `"\)`).MatchString(body)
}

// allEndpointsExcluded reports whether every endpoint of the feature is on the
// reviewed excludedEndpoints allowlist (and the feature has at least one endpoint).
func allEndpointsExcluded(f readiness.Feature) bool {
	if len(f.Endpoints) == 0 {
		return false
	}
	for _, ep := range f.Endpoints {
		if !excludedEndpoints[ep] {
			return false
		}
	}
	return true
}
