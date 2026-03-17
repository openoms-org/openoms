package linear_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_FetchTodoIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "lin_api_test")

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"issues": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"id":          "uuid-1",
							"identifier":  "OPE-150",
							"title":       "Test issue",
							"description": "Test desc",
							"priority":    2,
							"state":       map[string]interface{}{"name": "Todo"},
							"labels":      map[string]interface{}{"nodes": []interface{}{}},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	issues, err := client.FetchTodoIssues(context.Background())
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "OPE-150", issues[0].Identifier)
}

func TestClient_UpdateIssueState(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"issueUpdate": map[string]interface{}{
					"success": true,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	err := client.UpdateIssueState(context.Background(), "issue-uuid", "state-uuid-inprogress")
	require.NoError(t, err)

	query, ok := receivedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "issueUpdate")
}

func TestClient_AddComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"commentCreate": map[string]interface{}{"success": true},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	err := client.AddComment(context.Background(), "issue-uuid", "PR created: https://github.com/...")
	require.NoError(t, err)
}
