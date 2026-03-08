package session

import (
	"errors"
	"net/http"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for session management.
type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// ListSessions handles GET /api/sessions.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user ID")
		return
	}

	resp, err := h.svc.ListSessions(r.Context(), userID, claims.SessionID)
	if err != nil {
		h.log.Error("list sessions failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not list sessions")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// DeleteSession handles DELETE /api/sessions/{id}.
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user ID")
		return
	}

	sessionDBID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid session ID")
		return
	}

	if err := h.svc.DeleteSession(r.Context(), sessionDBID, userID); err != nil {
		h.log.Error("delete session failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not delete session")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteOtherSessions handles DELETE /api/sessions.
func (h *Handler) DeleteOtherSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user ID")
		return
	}

	if claims.SessionID == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "no session ID in token")
		return
	}

	if err := h.svc.DeleteOtherSessions(r.Context(), userID, claims.SessionID); err != nil {
		if errors.Is(err, ErrCannotDeleteCurrentSession) {
			httpx.Error(w, http.StatusBadRequest, "bad_request", "cannot delete current session")
			return
		}
		h.log.Error("delete other sessions failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not delete sessions")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
