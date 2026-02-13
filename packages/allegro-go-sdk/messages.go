package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// MessageService handles communication with the messaging endpoints.
type MessageService struct {
	client *Client
}

// ListThreads retrieves a paginated list of messaging threads.
func (s *MessageService) ListThreads(ctx context.Context, params *ListThreadsParams) (*ThreadList, error) {
	path := "/messaging/threads"

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result ThreadList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetThread retrieves a single messaging thread by ID.
func (s *MessageService) GetThread(ctx context.Context, threadID string) (*Thread, error) {
	var result Thread
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/messaging/threads/%s", threadID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListMessages retrieves a paginated list of messages in a thread.
func (s *MessageService) ListMessages(ctx context.Context, threadID string, params *ListMessagesParams) (*MessageList, error) {
	path := fmt.Sprintf("/messaging/threads/%s/messages", threadID)

	if params != nil {
		v := url.Values{}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			v.Set("offset", strconv.Itoa(params.Offset))
		}
		if params.Before != "" {
			v.Set("before", params.Before)
		}
		if encoded := v.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	var result MessageList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendMessage sends a new message in a thread.
func (s *MessageService) SendMessage(ctx context.Context, threadID string, msg SendMessageRequest) (*Message, error) {
	var result Message
	if err := s.client.do(ctx, "POST", fmt.Sprintf("/messaging/threads/%s/messages", threadID), msg, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
