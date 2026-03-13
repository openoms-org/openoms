package dhl

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

const (
	productionBaseURL = "https://dhl24.com.pl/webapi2"
	sandboxBaseURL    = "https://dhl24.com.pl/webapi2" // DHL24 has no separate sandbox URL
)

// Client is a DHL24 WebAPI2 SOAP client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	accountNum string

	Shipments *ShipmentService
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// WithSandbox is a no-op for DHL24 — there is no separate sandbox environment.
func WithSandbox() Option {
	return func(_ *Client) {}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// NewClient creates a new DHL24 SOAP API client.
func NewClient(username, password, accountNumber string, opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    productionBaseURL,
		username:   username,
		password:   password,
		accountNum: accountNumber,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Shipments = &ShipmentService{client: c}

	return c
}

// AccountNumber returns the configured DHL account number.
func (c *Client) AccountNumber() string {
	return c.accountNum
}

// soapEnvelope wraps any SOAP body for XML marshaling.
type soapEnvelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	NS      string   `xml:"xmlns:soap,attr"`
	Body    soapBody
}

type soapBody struct {
	XMLName xml.Name `xml:"soap:Body"`
	Content any      `xml:",omitempty"`
}

// authData is embedded in every DHL24 SOAP request.
type authData struct {
	XMLName  xml.Name `xml:"authData"`
	Username string   `xml:"username"`
	Password string   `xml:"password"`
}

// doSOAP sends a SOAP request and returns the raw XML response body.
func (c *Client) doSOAP(ctx context.Context, soapAction string, body any) ([]byte, error) {
	env := soapEnvelope{
		NS:   "http://schemas.xmlsoap.org/soap/envelope/",
		Body: soapBody{Content: body},
	}

	xmlBytes, err := xml.MarshalIndent(env, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("dhl: failed to marshal SOAP request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(xmlBytes))
	if err != nil {
		return nil, fmt.Errorf("dhl: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapAction)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dhl: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dhl: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to extract SOAP fault message
		faultMsg := extractSOAPFault(respBody)
		if faultMsg != "" {
			return nil, fmt.Errorf("dhl: api error %d: %s", resp.StatusCode, faultMsg)
		}
		return nil, fmt.Errorf("dhl: api error %d", resp.StatusCode)
	}

	// Check for SOAP fault in 200 response
	if fault := extractSOAPFault(respBody); fault != "" {
		return nil, fmt.Errorf("dhl: soap fault: %s", fault)
	}

	return respBody, nil
}

// extractSOAPFault extracts the faultstring from a SOAP Fault response.
func extractSOAPFault(body []byte) string {
	type soapFaultBody struct {
		Fault struct {
			FaultString string `xml:"faultstring"`
		} `xml:"Body>Fault"`
	}

	var f soapFaultBody
	if err := xml.Unmarshal(body, &f); err == nil && f.Fault.FaultString != "" {
		return f.Fault.FaultString
	}
	return ""
}

// APIError represents an error from the DHL API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("dhl: api error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("dhl: api error %d", e.StatusCode)
}
