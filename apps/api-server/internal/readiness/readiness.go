// Package readiness is the embedded single source of truth for feature-readiness gating.
package readiness

import (
	_ "embed"
	"encoding/json"
)

//go:embed readiness.json
var registryJSON []byte

// State is a feature/provider readiness level, matching the dashboard registry.
type State string

// Readiness states, mirroring the dashboard FeatureReadiness vocabulary.
const (
	Ready      State = "ready"
	Controlled State = "controlled"
	Verify     State = "verify"
	Beta       State = "beta"
	Blocked    State = "blocked"
)

// Feature is a gated capability: its readiness state plus the dashboard routes
// and API endpoint prefixes it covers.
type Feature struct {
	State     State    `json:"state"`
	Routes    []string `json:"routes"`
	Endpoints []string `json:"endpoints"`
}

type registry struct {
	Providers map[string]State   `json:"providers"`
	Features  map[string]Feature `json:"features"`
}

var reg registry

func init() {
	if err := json.Unmarshal(registryJSON, &reg); err != nil {
		panic("readiness: invalid embedded readiness.json: " + err.Error())
	}
}

// isVisible mirrors dashboard readiness.ts isReadinessVisible: blocked is never
// visible; "full" allows any non-blocked state; "client-ready" allows only "ready".
func isVisible(s State, mode string) bool {
	if s == Blocked {
		return false
	}
	if mode == "full" {
		return true
	}
	return s == Ready
}

// IsFeatureEnabled reports whether a feature is reachable under the surface mode.
// Unknown feature ids are treated as non-ready (state "verify"), matching the
// frontend getRouteReadiness fallback.
func IsFeatureEnabled(featureID, mode string) bool {
	f, ok := reg.Features[featureID]
	if !ok {
		return isVisible(Verify, mode)
	}
	return isVisible(f.State, mode)
}

// IsProviderEnabled reports whether a provider key is selectable under the mode.
// Unknown providers are treated as non-ready.
func IsProviderEnabled(providerKey, mode string) bool {
	s, ok := reg.Providers[providerKey]
	if !ok {
		return isVisible(Verify, mode)
	}
	return isVisible(s, mode)
}

// LookupFeature returns the feature entry and whether it exists.
func LookupFeature(id string) (Feature, bool) { f, ok := reg.Features[id]; return f, ok }

// NonReadyFeatures exposes the endpoint prefixes per non-ready feature for the
// coverage drift guard.
func NonReadyFeatures() map[string]Feature {
	out := map[string]Feature{}
	for id, f := range reg.Features {
		if f.State != Ready {
			out[id] = f
		}
	}
	return out
}
