package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

type UploadHandler struct {
	uploadDir string
	maxSize   int64
	baseURL   string
}

func NewUploadHandler(uploadDir string, maxSize int64, baseURL string) *UploadHandler {
	return &UploadHandler{
		uploadDir: uploadDir,
		maxSize:   maxSize,
		baseURL:   baseURL,
	}
}

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxSize)

	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect content type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	contentType := http.DetectContentType(buf[:n])

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported file type: only JPEG, PNG, and WEBP are allowed")
		return
	}

	// Seek back to beginning
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Create tenant directory
	tenantDir := filepath.Join(h.uploadDir, tenantID.String())
	if err := os.MkdirAll(tenantDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + ext
	filePath := filepath.Join(tenantDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(filePath) // cleanup on error
		writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	// Build URL
	url := fmt.Sprintf("%s/uploads/%s/%s", strings.TrimRight(h.baseURL, "/"), tenantID.String(), filename)

	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}
