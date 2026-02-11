package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

type UploadHandler struct {
	storage storage.ObjectStorage
	maxSize int64
}

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

	// Generate unique filename and storage key
	filename := uuid.New().String() + ext
	key := fmt.Sprintf("%s/%s", tenantID.String(), filename)

	// Upload via storage backend
	url, err := h.storage.Upload(r.Context(), key, file, contentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}
