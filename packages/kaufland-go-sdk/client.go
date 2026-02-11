package kaufland

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	productionBaseURL = "https://sellerapi.kaufland.com/v2"
	sandboxBaseURL    = "https://sellerapi.kaufland.com/v2" // Kaufland uses same URL with sandbox header
)

// Client is the Kaufland Seller API v2 client.
// Authentication uses HMAC-SHA256 signed requests with API key and secret key.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	secretKey  string
	sandbox    bool

	Orders *OrderService
}

// Option configures a Client.
type Option func(*Client)

// WithSandbox configures the client to target the sandbox environment.
func WithSandbox() Option {
	return func(c *Client) {
		c.sandbox = true
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithBaseURL overrides the full API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(u, "/")
	}
}

// NewClient creates a new Kaufland Seller API client.
func NewClient(apiKey, secretKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    productionBaseURL,
		apiKey:     apiKey,
		secretKey:  secretKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Orders = &OrderService{client: c}

	return c
}

// sign generates the HMAC-SHA256 signature for a Kaufland API request.
// The signature is computed over: method + url + body + timestamp.
func (c *Client) sign(method, fullURL string, body []byte, timestamp string) string {
	data := strings.Join([]string{
		method,
		fullURL,
		string(body),
		timestamp,
	}, "\n")

	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// do executes an authenticated API request using HMAC-SHA256 signing.
func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("kaufland: marshal request body: %w", err)
		}
	}

	fullURL := c.baseURL + path
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.sign(method, fullURL, bodyBytes, timestamp)

	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("kaufland: create request: %w", err)
	}

	req.Header.Set("Shop-Client-Key", c.apiKey)
	req.Header.Set("Shop-Timestamp", timestamp)
	req.Header.Set("Shop-Signature", signature)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.sandbox {
		req.Header.Set("Shop-Storefront", "sandbox")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kaufland: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("kaufland: decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the Kaufland API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Code       int    `json:"code"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "kaufland: HTTP %d", e.StatusCode)
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

// Unwrap returns a sentinel error based on the HTTP status code.
func (e *APIError) Unwrap() error {
	switch {
	case e.StatusCode == 401:
		return ErrUnauthorized
	case e.StatusCode == 403:
		return ErrForbidden
	case e.StatusCode == 404:
		return ErrNotFound
	case e.StatusCode == 429:
		return ErrRateLimited
	case e.StatusCode >= 500:
		return ErrServerError
	default:
		return nil
	}
}
