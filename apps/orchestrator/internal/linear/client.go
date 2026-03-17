package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.linear.app/graphql"

type Client struct {
	apiKey  string
	teamID  string
	baseURL string
	http    *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

func NewClient(apiKey, teamID string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		teamID:  teamID,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Issue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
}

func (c *Client) FetchTodoIssues(ctx context.Context) ([]Issue, error) {
	query := `query($teamId: String!) {
		issues(filter: {
			team: { key: { eq: $teamId } }
			state: { name: { eq: "Todo" } }
		}, first: 50) {
			nodes {
				id
				identifier
				title
				description
				priority
				state { name }
				labels { nodes { name } }
			}
		}
	}`

	var result struct {
		Data struct {
			Issues struct {
				Nodes []Issue `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"teamId": c.teamID}, &result); err != nil {
		return nil, fmt.Errorf("fetch todo issues: %w", err)
	}
	return result.Data.Issues.Nodes, nil
}

func (c *Client) UpdateIssueState(ctx context.Context, issueID, stateID string) error {
	query := `mutation($id: String!, $stateId: String!) {
		issueUpdate(id: $id, input: { stateId: $stateId }) {
			success
		}
	}`

	var result struct {
		Data struct {
			IssueUpdate struct {
				Success bool `json:"success"`
			} `json:"issueUpdate"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"id": issueID, "stateId": stateID}, &result); err != nil {
		return fmt.Errorf("update issue state: %w", err)
	}
	return nil
}

func (c *Client) AddComment(ctx context.Context, issueID, body string) error {
	query := `mutation($issueId: String!, $body: String!) {
		commentCreate(input: { issueId: $issueId, body: $body }) {
			success
		}
	}`

	var result struct {
		Data struct {
			CommentCreate struct {
				Success bool `json:"success"`
			} `json:"commentCreate"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"issueId": issueID, "body": body}, &result); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

func (c *Client) graphql(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("linear API error: status %d, body: %s", resp.StatusCode, respBody)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
