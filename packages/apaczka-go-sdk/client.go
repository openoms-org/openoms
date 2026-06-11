// Package apaczka provides a client for the Apaczka (R2G/Alsendo) broker API v2.
//
// API base URL: https://www.apaczka.pl/api/v2/
// All requests are HTTP POST with Content-Type: application/x-www-form-urlencoded.
// Each request is signed with HMAC-SHA256 over the message:
//
//	app_id + ":" + route + ":" + request + ":" + expires
//
// NOTE: The official documentation does not clearly state whether the signed
// "route" string should include the "api/v2/" prefix or only the terminal path
// segment (e.g. "order_send/"). The current implementation uses the same string
// for both the URL path and the HMAC input — confirm on the first live call.
package apaczka

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://www.apaczka.pl/api/v2/"
	maxBodyBytes   = 10 * 1024 * 1024 // 10 MB
)

// Client is an Apaczka API v2 client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	appID      string
	appSecret  string
	now        func() int64 // returns current unix timestamp; overridable for tests
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
// The URL must end with a trailing slash.
func WithBaseURL(u string) Option {
	return func(cl *Client) {
		cl.baseURL = u
	}
}

// WithNow sets a custom clock function returning a unix timestamp.
// This is used for deterministic signatures in tests.
func WithNow(fn func() int64) Option {
	return func(cl *Client) {
		cl.now = fn
	}
}

// NewClient creates a new Apaczka API client.
func NewClient(appID, appSecret string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    defaultBaseURL,
		appID:      appID,
		appSecret:  appSecret,
		now:        func() int64 { return time.Now().Unix() },
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents an error response from the Apaczka API.
type APIError struct {
	// Status is the numeric status code from the Apaczka response envelope.
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("apaczka: api error %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("apaczka: api error %d", e.Status)
}

// sign computes the HMAC-SHA256 signature for an Apaczka request.
//
// msg = appID + ":" + route + ":" + request + ":" + strconv.FormatInt(expires, 10)
// signature = hex(HMAC-SHA256(key=appSecret, msg))
func sign(appID, route, request string, expires int64, appSecret string) string {
	msg := appID + ":" + route + ":" + request + ":" + strconv.FormatInt(expires, 10)
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// envelope is the outer JSON wrapper for every Apaczka API response.
type envelope struct {
	Status   int             `json:"status"`
	Message  string          `json:"message"`
	Response json.RawMessage `json:"response"`
}

// do performs a signed POST request to the Apaczka API.
//
// route is the path appended to baseURL (e.g. "order_send/" or "order/123/").
// payload is the Go value to marshal as the "request" form field (nil → "{}").
// result is the Go value into which response.response is decoded; may be nil.
func (c *Client) do(ctx context.Context, route string, payload any, result any) error {
	// Marshal payload to JSON string ("request" field).
	var requestJSON string
	if payload == nil {
		requestJSON = "{}"
	} else {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("apaczka: marshal payload: %w", err)
		}
		requestJSON = string(b)
	}

	expires := c.now() + 1800
	sig := sign(c.appID, route, requestJSON, expires, c.appSecret)

	form := url.Values{}
	form.Set("app_id", c.appID)
	form.Set("request", requestJSON)
	form.Set("expires", strconv.FormatInt(expires, 10))
	form.Set("signature", sig)

	reqURL := c.baseURL + route
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("apaczka: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apaczka: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("apaczka: read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("apaczka: decode response envelope: %w", err)
	}

	if env.Status != 200 {
		return &APIError{Status: env.Status, Message: env.Message}
	}

	if result != nil && len(env.Response) > 0 {
		if err := json.Unmarshal(env.Response, result); err != nil {
			return fmt.Errorf("apaczka: decode response body: %w", err)
		}
	}

	return nil
}
