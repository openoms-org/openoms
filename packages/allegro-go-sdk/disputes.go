package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// DisputeService handles Allegro post-purchase disputes.
type DisputeService struct {
	client *Client
}

// List retrieves a paginated list of the seller's disputes.
func (s *DisputeService) List(ctx context.Context, params *ListDisputesParams) (*DisputeList, error) {
	path := "/sale/disputes"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Status != "" {
			v.Set("status", params.Status)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result DisputeList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single dispute by ID.
func (s *DisputeService) Get(ctx context.Context, disputeID string) (*Dispute, error) {
	var result Dispute
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/disputes/%s", disputeID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListMessages retrieves the messages in a dispute.
func (s *DisputeService) ListMessages(ctx context.Context, disputeID string) (*DisputeMessageList, error) {
	var result DisputeMessageList
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/disputes/%s/messages", disputeID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendMessage sends a message or response in a dispute.
func (s *DisputeService) SendMessage(ctx context.Context, disputeID string, msg DisputeMessageRequest) (*DisputeMessage, error) {
	var result DisputeMessage
	if err := s.client.do(ctx, "POST", fmt.Sprintf("/sale/disputes/%s/messages", disputeID), msg, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
