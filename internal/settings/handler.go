package settings

import (
	"encoding/json"
	"net/http"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for user settings operations.
type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// GetSettings handles GET /api/settings.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	settings, err := h.svc.GetOrCreateSettings(r.Context(), claims)
	if err != nil {
		h.log.Error("get settings failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not get settings")
		return
	}

	httpx.JSON(w, http.StatusOK, settings)
}

// UpdateSettings handles PUT /api/settings.
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	settings, err := h.svc.UpdateSettings(r.Context(), claims, req)
	if err != nil {
		h.log.Error("update settings failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not update settings")
		return
	}

	httpx.JSON(w, http.StatusOK, settings)
}
