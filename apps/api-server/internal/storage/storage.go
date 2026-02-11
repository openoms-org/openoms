package storage

import (
	"context"
	"io"
)

// ObjectStorage abstracts file storage backends (local disk, S3, etc.)
type ObjectStorage interface {
	// Upload stores a file and returns its public URL.
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	// Delete removes a file by key.
	Delete(ctx context.Context, key string) error
}
