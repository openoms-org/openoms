package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

// PersistShipmentLabel writes label bytes to object storage and returns the
// canonical /uploads/{tenant}/{file} URL used by GET /v1/shipments/{id}/label.
func PersistShipmentLabel(ctx context.Context, store storage.ObjectStorage, baseURL string, tenantID uuid.UUID, filename, contentType string, data []byte) (string, error) {
	return storeLabel(ctx, store, baseURL, tenantID, filename, contentType, data)
}

func storeLabel(ctx context.Context, store storage.ObjectStorage, baseURL string, tenantID uuid.UUID, filename, contentType string, data []byte) (string, error) {
	if store == nil {
		return "", fmt.Errorf("object storage is not configured")
	}
	if err := validateLabelFilename(filename); err != nil {
		return "", err
	}
	key := tenantID.String() + "/" + filename
	if _, err := store.Upload(ctx, key, bytes.NewReader(data), contentType); err != nil {
		return "", fmt.Errorf("saving label file: %w", err)
	}
	return fmt.Sprintf("%s/uploads/%s", strings.TrimRight(baseURL, "/"), key), nil
}

func readLabelObject(ctx context.Context, store storage.ObjectStorage, labelURL string) ([]byte, error) {
	key, err := labelStorageKey(labelURL)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrLabelNotAvailable
	}
	reader, err := store.Get(ctx, key)
	if err != nil {
		if isMissingObject(err) {
			return nil, ErrLabelNotAvailable
		}
		return nil, fmt.Errorf("reading label object: %w", err)
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(reader)
}

func labelStorageKey(labelURL string) (string, error) {
	if labelURL == "" {
		return "", ErrLabelNotAvailable
	}
	const marker = "/uploads/"
	_, after, ok := strings.Cut(labelURL, marker)
	if !ok {
		return "", fmt.Errorf("%w: invalid label URL format", ErrLabelNotAvailable)
	}
	relPath := filepath.Clean(after)
	if relPath == "." || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return "", fmt.Errorf("invalid label path")
	}
	return relPath, nil
}

func validateLabelFilename(filename string) error {
	cleaned := filepath.Clean(filename)
	if filename == "" || cleaned != filename || strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return fmt.Errorf("invalid label filename")
	}
	return nil
}

func labelContentType(ext string) string {
	switch ext {
	case "pdf":
		return "application/pdf"
	case "png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

func isMissingObject(err error) bool {
	return errors.Is(err, storage.ErrNotFound)
}
