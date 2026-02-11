package smsapi

import (
	"net/http"
)

// Client is an SMSAPI.pl API client.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
	from       string // default sender name
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
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// WithFrom sets the default sender name for outgoing SMS messages.
func WithFrom(from string) Option {
	return func(cl *Client) {
		cl.from = from
	}
}

// NewClient creates a new SMSAPI.pl API client.
// apiToken is the Bearer token for authentication.
func NewClient(apiToken string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    "https://api.smsapi.pl",
		apiToken:   apiToken,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}
