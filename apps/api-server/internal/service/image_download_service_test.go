package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple image URL",
			url:      "https://example.com/images/product.jpg",
			expected: "product.jpg",
		},
		{
			name:     "URL with query params",
			url:      "https://cdn.example.com/photo.png?w=800&h=600",
			expected: "photo.png",
		},
		{
			name:     "URL with path segments",
			url:      "https://storage.example.com/bucket/tenant/products/image.webp",
			expected: "image.webp",
		},
		{
			name:     "URL ending with slash",
			url:      "https://example.com/images/",
			expected: "images",
		},
		{
			name: "empty URL returns UUID-based name",
			url:  "",
		},
		{
			name: "URL with no path",
			url:  "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filenameFromURL(tt.url)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				// Should be a UUID-based fallback
				assert.True(t, strings.HasSuffix(result, ".jpg"), "expected .jpg suffix, got: %s", result)
				assert.True(t, len(result) > 4, "expected non-empty filename")
			}
		})
	}
}

func TestIsLocalURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: true,
		},
		{
			name:     "local upload URL",
			url:      "http://localhost:8080/uploads/tenant-id/products/image.jpg",
			expected: true,
		},
		{
			name:     "external URL",
			url:      "https://cdn.example.com/images/photo.jpg",
			expected: false,
		},
		{
			name:     "allegro image URL",
			url:      "https://a.allegroimg.com/s512/some-hash",
			expected: false,
		},
		{
			name:     "relative path",
			url:      "/images/product.jpg",
			expected: true,
		},
		{
			name:     "S3-like URL with uploads",
			url:      "https://s3.eu-central-1.amazonaws.com/bucket/uploads/image.png",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageDownload_DownloadsExternalURLs(t *testing.T) {
	imgData := []byte("fake-image-data-png")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgData)
	}))
	defer server.Close()

	svc := &ImageDownloadService{
		storage: &mockObjectStorage{
			uploadURL: "https://cdn.example.com/uploaded/image.png",
		},
		httpClient: server.Client(),
	}

	body, contentType, err := svc.downloadImage(context.Background(), server.URL+"/image.png")
	require.NoError(t, err)
	defer body.Close()

	assert.Equal(t, "image/png", contentType)

	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, imgData, data)
}

func TestImageDownload_SkipsLocalURLs(t *testing.T) {
	assert.True(t, isLocalURL("http://localhost:8080/uploads/abc/image.jpg"))
	assert.True(t, isLocalURL("https://api.openoms.org/uploads/tenant/products/photo.png"))
	assert.False(t, isLocalURL("https://external-cdn.com/images/photo.jpg"))
}

func TestImageDownload_HandlesFailedDownloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc := &ImageDownloadService{
		storage:    &mockObjectStorage{},
		httpClient: server.Client(),
	}

	_, _, err := svc.downloadImage(context.Background(), server.URL+"/missing.jpg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestImageDownload_DefaultContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Write raw bytes without setting Content-Type explicitly
		w.Header().Set("Content-Type", "")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer server.Close()

	svc := &ImageDownloadService{
		storage:    &mockObjectStorage{},
		httpClient: server.Client(),
	}

	body, contentType, err := svc.downloadImage(context.Background(), server.URL+"/no-type.jpg")
	require.NoError(t, err)
	defer body.Close()

	// When Content-Type header is empty, should default to image/jpeg
	assert.Equal(t, "image/jpeg", contentType)
}

func TestImageDownload_DownloadAndUpload(t *testing.T) {
	imgData := []byte("test-image-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imgData)
	}))
	defer server.Close()

	mock := &mockObjectStorage{
		uploadURL: "https://cdn.example.com/tenant/products/id/photo.webp",
	}
	svc := &ImageDownloadService{
		storage:    mock,
		httpClient: server.Client(),
	}

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	productID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	newURL, err := svc.downloadAndUpload(context.Background(), tenantID, productID, server.URL+"/photo.webp")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/tenant/products/id/photo.webp", newURL)
	assert.Equal(t, "image/webp", mock.lastContentType)
	assert.Contains(t, mock.lastKey, tenantID.String())
	assert.Contains(t, mock.lastKey, productID.String())
	assert.Contains(t, mock.lastKey, "photo.webp")
}

// mockObjectStorage implements storage.ObjectStorage for testing.
type mockObjectStorage struct {
	uploadURL       string
	lastKey         string
	lastContentType string
}

func (m *mockObjectStorage) Upload(_ context.Context, key string, reader io.Reader, contentType string) (string, error) {
	m.lastKey = key
	m.lastContentType = contentType
	_, _ = io.ReadAll(reader)
	return m.uploadURL, nil
}

func (m *mockObjectStorage) Delete(_ context.Context, _ string) error {
	return nil
}
