package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadHandler_Upload_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir, "http://localhost:8080")
	h := NewUploadHandler(store, 1<<20)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create a multipart request without a file field
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("other", "value")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Upload(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "missing file field", resp["error"])
}

func TestUploadHandler_Upload_UnsupportedFileType(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir, "http://localhost:8080")
	h := NewUploadHandler(store, 1<<20)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create a multipart request with a text file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	part.Write([]byte("This is a plain text file, not an image"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Upload(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "unsupported file type")
}

func TestUploadHandler_Upload_ValidPNG(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir, "http://localhost:8080")
	h := NewUploadHandler(store, 1<<20)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create a minimal valid PNG file (8-byte header + minimum IHDR chunk)
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	// IHDR chunk: length(13) + type(IHDR) + data(13 bytes) + CRC(4 bytes)
	ihdr := []byte{
		0x00, 0x00, 0x00, 0x0D, // length: 13
		'I', 'H', 'D', 'R', // chunk type
		0x00, 0x00, 0x00, 0x01, // width: 1
		0x00, 0x00, 0x00, 0x01, // height: 1
		0x08, // bit depth: 8
		0x02, // color type: RGB
		0x00, 0x00, 0x00, // compression, filter, interlace
		0x00, 0x00, 0x00, 0x00, // CRC (fake but sufficient for content type detection)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "image.png")
	require.NoError(t, err)
	part.Write(pngHeader)
	part.Write(ihdr)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Upload(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["url"], tenantID.String())
	assert.Contains(t, resp["url"], ".png")

	// Verify file was actually created on disk
	tenantDir := filepath.Join(tmpDir, tenantID.String())
	entries, err := os.ReadDir(tenantDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestUploadHandler_Upload_InvalidMultipartForm(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewLocalStorage(tmpDir, "http://localhost:8080")
	h := NewUploadHandler(store, 1<<20)

	tenantID := uuid.New()
	userID := uuid.New()

	// Send a non-multipart request
	req := httptest.NewRequest(http.MethodPost, "/v1/upload", bytes.NewReader([]byte("not multipart")))
	req = req.WithContext(newContextWithTenantAndUser(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.Upload(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUploadHandler_NewUploadHandler(t *testing.T) {
	store := storage.NewLocalStorage("/tmp/uploads", "http://example.com")
	h := NewUploadHandler(store, 5*1024*1024)
	assert.NotNil(t, h)
}
