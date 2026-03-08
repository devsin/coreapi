package search

import (
	"net/http"
	"strconv"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"
	"github.com/google/uuid"

	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for search.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a search handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Search handles GET /api/search?q=&limit=&offset=.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "query parameter 'q' is required")
		return
	}

	limit := int32(defaultLimit)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 32); err == nil && v > 0 {
			limit = int32(v)
		}
	}

	offset := int32(0)
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.ParseInt(o, 10, 32); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	var viewerID *uuid.UUID
	if claims, ok := auth.FromContext(r.Context()); ok && claims.UserID != "" {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			viewerID = &id
		}
	}

	result, err := h.svc.Search(r.Context(), query, limit, offset, viewerID)
	if err != nil {
		h.log.Error("search failed", zap.String("query", query), zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "search failed")
		return
	}

	httpx.JSON(w, http.StatusOK, result)
}
