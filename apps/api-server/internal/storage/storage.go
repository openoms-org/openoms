package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is returned by Get when the object key does not exist.
var ErrNotFound = errors.New("object not found")

// ObjectStorage abstracts file storage backends (local disk, S3, etc.)
type ObjectStorage interface {
	// Upload stores a file and returns its public URL.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	// Get retrieves a file by key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes a file by key.
	Delete(ctx context.Context, key string) error
}
