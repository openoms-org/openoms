package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_Upload(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8080")

	ctx := context.Background()
	key := "tenant-abc/photo.jpg"
	content := "fake image data"

	url, err := store.Upload(ctx, key, strings.NewReader(content), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/uploads/tenant-abc/photo.jpg", url)

	// Verify file exists on disk with correct content
	data, err := os.ReadFile(filepath.Join(tmpDir, key))
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestLocalStorage_Upload_CreatesSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://example.com/")

	ctx := context.Background()
	key := "deep/nested/path/file.png"

	url, err := store.Upload(ctx, key, strings.NewReader("data"), "image/png")
	require.NoError(t, err)
	// baseURL trailing slash should be trimmed
	assert.Equal(t, "http://example.com/uploads/deep/nested/path/file.png", url)

	// Verify file exists
	_, err = os.Stat(filepath.Join(tmpDir, key))
	require.NoError(t, err)
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8080")

	ctx := context.Background()
	key := "tenant-abc/photo.jpg"

	// First upload a file
	_, err := store.Upload(ctx, key, strings.NewReader("data"), "image/jpeg")
	require.NoError(t, err)

	// Verify it exists
	fullPath := filepath.Join(tmpDir, key)
	_, err = os.Stat(fullPath)
	require.NoError(t, err)

	// Delete it
	err = store.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8080")

	// Deleting a non-existent file should not return an error
	err := store.Delete(context.Background(), "does/not/exist.jpg")
	assert.NoError(t, err)
}

func TestNewS3Storage_Constructor(t *testing.T) {
	// Verify the S3 constructor creates a valid instance with the given config.
	// This does not make real AWS calls; it only validates config parsing.
	s3Store, err := NewS3Storage(
		"eu-central-1",
		"my-bucket",
		"http://localhost:9000", // MinIO endpoint
		"minioadmin",
		"minioadmin",
		"https://cdn.example.com",
	)
	require.NoError(t, err)
	assert.NotNil(t, s3Store)
	assert.Equal(t, "my-bucket", s3Store.bucket)
	assert.Equal(t, "https://cdn.example.com", s3Store.publicURL)
}

func TestNewS3Storage_TrailingSlash(t *testing.T) {
	s3Store, err := NewS3Storage(
		"us-east-1",
		"test-bucket",
		"",
		"key",
		"secret",
		"https://cdn.example.com/",
	)
	require.NoError(t, err)
	// Trailing slash should be trimmed
	assert.Equal(t, "https://cdn.example.com", s3Store.publicURL)
}
