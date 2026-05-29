package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
)

// RequireFeature gates a route group by feature readiness. When the feature is not
// enabled for the active surface mode it returns 404 with a JSON body, hiding the
// capability from clients (mirrors the dashboard readiness route guard).
func RequireFeature(featureID, surfaceMode string) func(http.Handler) http.Handler {
	enabled := readiness.IsFeatureEnabled(featureID, surfaceMode)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "feature_not_available"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
