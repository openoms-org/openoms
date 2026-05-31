package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v82"
)

// A malformed event payload is a permanent failure: it must be classified as
// ErrWebhookUnprocessable so the handler returns 4xx and Stripe stops retrying
// (transient failures stay 5xx and are retried). The unmarshal fails before any
// repository call, so a zero-value service is sufficient.
func TestStripeWebhook_MalformedCheckoutPayloadIsUnprocessable(t *testing.T) {
	svc := &StripeWebhookService{}
	event := stripe.Event{Data: &stripe.EventData{Raw: json.RawMessage(`not-json`)}}

	err := svc.handleCheckoutCompleted(context.Background(), event)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookUnprocessable)
}

func TestStripeWebhook_MalformedSubscriptionPayloadIsUnprocessable(t *testing.T) {
	svc := &StripeWebhookService{}
	event := stripe.Event{Data: &stripe.EventData{Raw: json.RawMessage(`{`)}}

	err := svc.handleSubscriptionUpdated(context.Background(), event)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookUnprocessable)
}
