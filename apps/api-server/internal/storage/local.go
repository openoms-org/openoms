// Package storage implements local filesystem-backed object storage.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	baseDir string
	baseURL string
}

// NewLocalStorage creates a new LocalStorage instance.
// baseDir is the root directory for uploads (e.g., "./uploads").
// baseURL is the public URL prefix (e.g., "http://localhost:8080").
func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{
		baseDir: baseDir,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Upload saves a file to baseDir/key and returns its public URL.
// The key is expected to be in the form "tenantID/filename.ext".
func (s *LocalStorage) Upload(_ context.Context, key string, reader io.Reader, _ string) (string, error) {
	fullPath := filepath.Clean(filepath.Join(s.baseDir, key))

	// Ensure the parent directory exists.
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}

	f, err := os.Create(fullPath) // #nosec G304 -- path is cleaned and joined under baseDir
	if err != nil {
		return "", fmt.Errorf("creating file %s: %w", fullPath, err)
	}

	if _, err := io.Copy(f, reader); err != nil {
		_ = f.Close()
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("writing file %s: %w", fullPath, err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("closing file %s: %w", fullPath, err)
	}

	url := fmt.Sprintf("%s/uploads/%s", s.baseURL, key)
	return url, nil
}

// Delete removes a file at baseDir/key.
func (s *LocalStorage) Delete(_ context.Context, key string) error {
	fullPath := filepath.Join(s.baseDir, key)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file %s: %w", fullPath, err)
	}
	return nil
}
