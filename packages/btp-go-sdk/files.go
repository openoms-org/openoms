package btp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
)

// FileService handles temporary file storage operations.
type FileService struct {
	client *Client
}

// Add uploads a file to the user's temporary storage.
// Files are automatically deleted after 7 days unless attached to a business object.
func (s *FileService) Add(ctx context.Context, filename string, file io.Reader, description string) (*FileAddResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("btp: create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("btp: copy file data: %w", err)
	}

	if description != "" {
		if err := writer.WriteField("description", description); err != nil {
			return nil, fmt.Errorf("btp: write description field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("btp: close multipart writer: %w", err)
	}

	var resp FileAddResponse
	if err := s.client.doMultipart(ctx, "/Gateway/ClientApi/FileAdd", &buf, writer.FormDataContentType(), &resp); err != nil {
		return nil, fmt.Errorf("btp: add file: %w", err)
	}
	return &resp, nil
}

// Delete removes specified files from temporary storage.
func (s *FileService) Delete(ctx context.Context, fileIDs []string) (*APIResult, error) {
	req := FilesDeleteRequest{FileIDs: fileIDs}
	var resp APIResult
	if err := s.client.do(ctx, "DELETE", "/Gateway/ClientApi/FilesDelete", &req, &resp); err != nil {
		return nil, fmt.Errorf("btp: delete files: %w", err)
	}
	return &resp, nil
}

// Clear removes all files from the user's temporary storage.
func (s *FileService) Clear(ctx context.Context) (*APIResult, error) {
	var resp APIResult
	if err := s.client.do(ctx, "DELETE", "/Gateway/ClientApi/FilesClear", nil, &resp); err != nil {
		return nil, fmt.Errorf("btp: clear files: %w", err)
	}
	return &resp, nil
}
