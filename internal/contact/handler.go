package contact

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/devsin/coreapi/common/httpx"

	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for contact messages.
type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// SubmitContactMessage handles POST /api/contact.
func (h *Handler) SubmitContactMessage(w http.ResponseWriter, r *http.Request) {
	var req CreateContactMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	dto, err := h.svc.SubmitContactMessage(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNameRequired):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Name is required")
		case errors.Is(err, ErrNameTooLong):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Name is too long (max 100 characters)")
		case errors.Is(err, ErrEmailRequired):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "A valid email address is required")
		case errors.Is(err, ErrEmailTooLong):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Email is too long")
		case errors.Is(err, ErrMessageRequired):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Message is required")
		case errors.Is(err, ErrMessageTooLong):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Message is too long (max 5000 characters)")
		case errors.Is(err, ErrInvalidSubject):
			httpx.Error(w, http.StatusBadRequest, "validation_error", "Invalid subject category")
		default:
			h.log.Error("submit contact message failed", zap.Error(err))
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "Could not send message")
		}
		return
	}

	httpx.JSON(w, http.StatusCreated, dto)
}
