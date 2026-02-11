package woocommerce

import (
	"context"
	"fmt"
)

// WebhookService handles communication with the webhook-related WooCommerce endpoints.
type WebhookService struct {
	client *Client
}

// WooWebhook represents a WooCommerce webhook.
type WooWebhook struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Topic       string `json:"topic"`
	Resource    string `json:"resource"`
	Event       string `json:"event"`
	DeliveryURL string `json:"delivery_url"`
	Secret      string `json:"secret"`
}

// Create creates a new webhook for a given topic.
func (s *WebhookService) Create(ctx context.Context, topic, deliveryURL, secret string) (*WooWebhook, error) {
	body := map[string]any{
		"name":         "OpenOMS " + topic,
		"topic":        topic,
		"delivery_url": deliveryURL,
		"secret":       secret,
		"status":       "active",
	}

	var result WooWebhook
	if err := s.client.do(ctx, "POST", "/webhooks", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a webhook by ID.
func (s *WebhookService) Delete(ctx context.Context, id int) error {
	return s.client.do(ctx, "DELETE", fmt.Sprintf("/webhooks/%d?force=true", id), nil, nil)
}

// List retrieves all webhooks.
func (s *WebhookService) List(ctx context.Context) ([]WooWebhook, error) {
	var result []WooWebhook
	if err := s.client.do(ctx, "GET", "/webhooks", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
