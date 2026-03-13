package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

// UploadHandler handles file upload requests.
type UploadHandler struct {
	storage storage.ObjectStorage
	maxSize int64
}

// NewUploadHandler creates a new UploadHandler.
func NewUploadHandler(store storage.ObjectStorage, maxSize int64) *UploadHandler {
	return &UploadHandler{
		storage: store,
		maxSize: maxSize,
	}
}

var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Upload handles multipart file upload and stores the file in object storage.
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxSize)

	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	// Read first 512 bytes to detect content type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		writeServerError(w, "failed to read file", err)
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
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			writeServerError(w, "failed to reset file position", err)
			return
		}
	}

	// Generate unique filename and storage key
	filename := uuid.New().String() + ext
	key := fmt.Sprintf("%s/%s", tenantID.String(), filename)

	// Upload via storage backend
	url, err := h.storage.Upload(r.Context(), key, file, contentType)
	if err != nil {
		writeServerError(w, "failed to upload file", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}
