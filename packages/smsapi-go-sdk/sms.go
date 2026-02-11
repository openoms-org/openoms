package smsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SendSMS sends an SMS message via the SMSAPI.pl API.
// The API uses form-encoded POST requests with Bearer token authentication.
func (c *Client) SendSMS(ctx context.Context, req SendSMSRequest) (*SendSMSResponse, error) {
	if req.To == "" {
		return nil, fmt.Errorf("smsapi: recipient phone number (to) is required")
	}
	if req.Message == "" {
		return nil, fmt.Errorf("smsapi: message is required")
	}

	// Use default from if not specified in request
	from := req.From
	if from == "" {
		from = c.from
	}

	// Build form-encoded body
	form := url.Values{}
	form.Set("to", req.To)
	form.Set("message", req.Message)
	form.Set("format", "json")
	form.Set("encoding", "utf-8")
	if from != "" {
		form.Set("from", from)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sms.do", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("smsapi: failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("smsapi: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("smsapi: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(body) > 0 {
			_ = json.Unmarshal(body, apiErr)
		}
		return nil, apiErr
	}

	var result SendSMSResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("smsapi: failed to decode response: %w", err)
	}

	// Check for per-message errors
	for _, item := range result.List {
		if item.Error != nil && *item.Error != 0 {
			return &result, fmt.Errorf("smsapi: message to %s failed with error code %d", item.Number, *item.Error)
		}
	}

	return &result, nil
}
