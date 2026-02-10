package allegro

import (
	"context"
	"net/url"
	"strings"
)

// EventService handles communication with the order event endpoints.
type EventService struct {
	client *Client
}

// Poll retrieves order events starting from the given event ID.
// Optionally filter by event types (e.g. "BOUGHT", "FILLED_IN", "READY_FOR_PROCESSING").
func (s *EventService) Poll(ctx context.Context, fromEventID string, eventTypes ...string) (*EventList, error) {
	v := url.Values{}
	if fromEventID != "" {
		v.Set("from", fromEventID)
	}
	if len(eventTypes) > 0 {
		v.Set("type", strings.Join(eventTypes, ","))
	}

	path := "/order/events"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result EventList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
