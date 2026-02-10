package inpost

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

const pointsBaseURL = "https://api-pl-points.easypack24.net"

// pointCodePattern matches InPost point codes like "KRA01M", "WAW123M", "POP-KRA356".
var pointCodePattern = regexp.MustCompile(`^[A-Z]{2,4}[-]?[A-Z]*\d`)

// PointService handles InPost point/paczkomat search operations.
type PointService struct {
	client  *Client
	baseURL string
}

// Search searches for InPost points. If the query looks like a point code (e.g. "KRA01M"),
// it searches by exact name. Otherwise it searches by city name.
func (s *PointService) Search(ctx context.Context, query string, pointType PointType, perPage int) (*PointSearchResponse, error) {
	if perPage <= 0 || perPage > 25 {
		perPage = 10
	}

	var filterParam string
	if pointCodePattern.MatchString(query) {
		filterParam = "name=" + url.QueryEscape(query)
	} else {
		filterParam = "city=" + url.QueryEscape(query)
	}

	path := fmt.Sprintf("/v1/points?%s&type=%s&per_page=%d",
		filterParam,
		url.QueryEscape(string(pointType)),
		perPage,
	)

	var resp PointSearchResponse
	if err := s.doPoints(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a single point by its exact name (e.g., "WAW123M").
func (s *PointService) Get(ctx context.Context, name string) (*Point, error) {
	path := fmt.Sprintf("/v1/points/%s", url.PathEscape(name))
	var point Point
	if err := s.doPoints(ctx, path, &point); err != nil {
		return nil, err
	}
	return &point, nil
}

// doPoints performs a GET request against the InPost Points API (separate host from ShipX).
func (s *PointService) doPoints(ctx context.Context, path string, result any) error {
	base := s.baseURL
	if base == "" {
		base = pointsBaseURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return fmt.Errorf("inpost: failed to create request: %w", err)
	}

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("inpost: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("inpost: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(body) > 0 {
			_ = json.Unmarshal(body, apiErr)
		}
		return apiErr
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("inpost: failed to decode response: %w", err)
		}
	}

	return nil
}
