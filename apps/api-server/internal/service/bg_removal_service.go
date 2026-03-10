package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
)

// BGRemovalService handles background removal from product images via remove.bg API.
type BGRemovalService struct {
	apiKey     string
	httpClient *http.Client
}

// NewBGRemovalService creates a new background removal service.
// If apiKey is empty, IsConfigured() returns false.
func NewBGRemovalService(apiKey string) *BGRemovalService {
	return &BGRemovalService{
		apiKey:     apiKey,
		httpClient: netutil.SafeHTTPClient(60 * time.Second),
	}
}

// IsConfigured returns true when the remove.bg API key is set.
func (s *BGRemovalService) IsConfigured() bool {
	return s.apiKey != ""
}

// RemoveBackground sends the image to remove.bg and returns the processed PNG bytes.
// The returned content type is always "image/png" (transparent background).
func (s *BGRemovalService) RemoveBackground(ctx context.Context, imageData []byte, filename string) ([]byte, string, error) {
	if !s.IsConfigured() {
		return nil, "", fmt.Errorf("background removal not configured — set REMOVEBG_API_KEY")
	}

	// Build multipart form body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image_file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, "", fmt.Errorf("write image data: %w", err)
	}

	// Set size to "auto" for best results
	if err := writer.WriteField("size", "auto"); err != nil {
		return nil, "", fmt.Errorf("write size field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.remove.bg/v1.0/removebg", &body)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Api-Key", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("remove.bg request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("remove.bg error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, "image/png", nil
}
