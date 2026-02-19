//go:build integration

package tests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var targetURL string

func TestMain(m *testing.M) {
	targetURL = os.Getenv("TARGET_URL")
	if targetURL == "" {
		fmt.Println("TARGET_URL not set, skipping integration tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(targetURL + path)
	require.NoError(t, err)
	return resp
}

func TestSecurity_Integration_SecurityHeadersPresent(t *testing.T) {
	resp := get(t, "/health")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.NotEmpty(t, resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
	assert.Empty(t, resp.Header.Get("Server"))
	assert.Empty(t, resp.Header.Get("X-Powered-By"))
}

func TestSecurity_Integration_ProtectedEndpointRequiresAuth(t *testing.T) {
	resp := get(t, "/v1/orders")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSecurity_Integration_InvalidJSONReturns400(t *testing.T) {
	resp, err := http.Post(targetURL+"/v1/auth/login", "application/json", strings.NewReader("not json"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSecurity_Integration_OversizedBodyReturns413(t *testing.T) {
	bigBody := strings.NewReader(strings.Repeat("x", 5*1024*1024)) // 5MB
	resp, err := http.Post(targetURL+"/v1/auth/login", "application/json", bigBody)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == http.StatusRequestEntityTooLarge || resp.StatusCode == http.StatusBadRequest)
}

func TestSecurity_Integration_PathTraversalBlocked(t *testing.T) {
	resp := get(t, "/v1/../../../etc/passwd")
	// Either 401 (auth blocks before routing) or 404 (route not found) is acceptable.
	// Both mean the traversal attempt was blocked — never 200.
	assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound,
		"expected 401 or 404, got %d", resp.StatusCode)
}

func TestSecurity_Integration_ErrorResponseNoStackTrace(t *testing.T) {
	resp := get(t, "/v1/nonexistent-route")
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.NotContains(t, bodyStr, "goroutine")
	assert.NotContains(t, bodyStr, ".go:")
	assert.NotContains(t, bodyStr, "runtime.")
}

func TestSecurity_Integration_SQLInjectionInQueryParams(t *testing.T) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", targetURL+"/v1/orders?search='+OR+1=1+--", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := client.Do(req)
	require.NoError(t, err)
	// Should be 401 (no auth) not 500 (SQL error)
	assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSecurity_Integration_HealthDoesNotLeakDBVersion(t *testing.T) {
	resp := get(t, "/health")
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.NotContains(t, bodyStr, "PostgreSQL")
	assert.NotContains(t, bodyStr, "postgres")
}
