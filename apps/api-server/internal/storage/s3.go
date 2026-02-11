package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage stores files in an S3-compatible bucket (AWS S3, MinIO, DigitalOcean Spaces, etc.)
type S3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string // optional CDN/public URL prefix
}

// NewS3Storage creates a new S3Storage instance.
// endpoint can be empty for standard AWS S3, or set for S3-compatible services.
// publicURL is the CDN/public URL prefix used to construct public file URLs.
func NewS3Storage(region, bucket, endpoint, accessKey, secretKey, publicURL string) (*S3Storage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	clientOpts := func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required for MinIO and most S3-compatible services
		}
	}

	client := s3.NewFromConfig(cfg, clientOpts)

	return &S3Storage{
		client:    client,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
	}, nil
}

// Upload stores a file in S3 and returns its public URL.
// The key is expected to be in the form "tenantID/filename.ext".
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("uploading to S3 (bucket=%s, key=%s): %w", s.bucket, key, err)
	}

	url := fmt.Sprintf("%s/%s", s.publicURL, key)
	return url, nil
}

// Delete removes a file from S3 by key.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if _, err := s.client.DeleteObject(ctx, input); err != nil {
		return fmt.Errorf("deleting from S3 (bucket=%s, key=%s): %w", s.bucket, key, err)
	}

	return nil
}
