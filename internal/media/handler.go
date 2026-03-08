package media

import (
	"errors"
	"net/http"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for media operations.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new media handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Upload handles POST /api/upload?type={avatar|cover}.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid user ID")
		return
	}

	// Determine upload type from query parameter.
	uploadType := UploadType(r.URL.Query().Get("type"))
	if uploadType == "" {
		httpx.Error(w, http.StatusBadRequest, "missing_type", "query parameter 'type' is required (avatar or cover)")
		return
	}

	// Parse multipart form with a generous in-memory limit.
	// Files larger than this are written to temp files automatically.
	const maxMemory = 2 << 20 // 2 MB in-memory buffer
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_form", "invalid multipart form data")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll() //nolint:errcheck // best-effort cleanup
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "missing_file", "form field 'file' is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := h.svc.Upload(r.Context(), userID, uploadType, file, contentType, header.Size)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrFileTooLarge):
		httpx.Error(w, http.StatusRequestEntityTooLarge, "file_too_large", "file exceeds maximum allowed size")
	case errors.Is(err, ErrInvalidFileType):
		httpx.Error(w, http.StatusBadRequest, "invalid_file_type", "only JPEG, PNG, WebP, and GIF images are allowed")
	case errors.Is(err, ErrInvalidUploadType):
		httpx.Error(w, http.StatusBadRequest, "invalid_upload_type", "upload type must be 'avatar' or 'cover'")
	case errors.Is(err, ErrUploadFailed):
		httpx.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload file")
	default:
		h.log.Error("unexpected media error", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
