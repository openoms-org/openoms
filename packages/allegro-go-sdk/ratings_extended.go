package allegro

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// RatingService handles Allegro user rating management (responses, removal requests).
type RatingService struct {
	client *Client
}

// List retrieves a paginated list of user ratings.
func (s *RatingService) List(ctx context.Context, params *RatingsParams) (*RatingList, error) {
	path := "/sale/user-ratings"

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

	var result RatingList
	if err := s.client.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAnswer retrieves the seller's answer to a rating.
func (s *RatingService) GetAnswer(ctx context.Context, ratingID string) (*RatingAnswer, error) {
	var result RatingAnswer
	if err := s.client.do(ctx, "GET", fmt.Sprintf("/sale/user-ratings/%s/answer", ratingID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateAnswer creates or updates the seller's answer to a rating.
func (s *RatingService) CreateAnswer(ctx context.Context, ratingID string, answer RatingAnswerRequest) (*RatingAnswer, error) {
	var result RatingAnswer
	if err := s.client.do(ctx, "PUT", fmt.Sprintf("/sale/user-ratings/%s/answer", ratingID), answer, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteAnswer removes the seller's answer to a rating.
func (s *RatingService) DeleteAnswer(ctx context.Context, ratingID string) error {
	return s.client.do(ctx, "DELETE", fmt.Sprintf("/sale/user-ratings/%s/answer", ratingID), nil, nil)
}

// RequestRemoval requests removal of a rating.
func (s *RatingService) RequestRemoval(ctx context.Context, ratingID string, reason string) error {
	body := RatingRemovalRequest{Reason: reason}
	return s.client.do(ctx, "POST", fmt.Sprintf("/sale/user-ratings/%s/removal", ratingID), body, nil)
}
