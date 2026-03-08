package notification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes notification HTTP endpoints.
type Handler struct {
	svc *Service
	hub *Hub
	log *zap.Logger
}

// NewHandler creates a new notification handler.
func NewHandler(svc *Service, hub *Hub, log *zap.Logger) *Handler {
	return &Handler{svc: svc, hub: hub, log: log}
}

// ListNotifications returns paginated notifications for the authenticated user.
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID")
		return
	}

	limit := int32(20)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = int32(n) //nolint:gosec // bounds checked above (1-50)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n) //nolint:gosec // bounds checked above (non-negative)
		}
	}

	resp, err := h.svc.List(r.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("list notifications failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal", "Failed to list notifications")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// GetUnreadCount returns the count of unread notifications.
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID")
		return
	}

	count, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		h.log.Error("unread count failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal", "Failed to get unread count")
		return
	}

	httpx.JSON(w, http.StatusOK, UnreadCountResponse{Count: count})
}

// MarkRead marks a single notification as read.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID")
		return
	}

	notifID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Invalid notification ID")
		return
	}

	if err := h.svc.MarkRead(r.Context(), notifID, userID); err != nil {
		h.log.Error("mark read failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal", "Failed to mark notification as read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead marks all notifications as read for the authenticated user.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID")
		return
	}

	if err := h.svc.MarkAllRead(r.Context(), userID); err != nil {
		h.log.Error("mark all read failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal", "Failed to mark all as read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Stream opens an SSE connection for real-time notification events.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	ch := h.hub.Subscribe(claims.UserID)
	defer h.hub.Unsubscribe(claims.UserID, ch)

	// Send initial unread count.
	userID, err := uuid.Parse(claims.UserID)
	if err == nil {
		count, cErr := h.svc.UnreadCount(r.Context(), userID)
		if cErr == nil {
			initEvt := Event{Type: "count_update", UnreadCount: count}
			if data, mErr := json.Marshal(initEvt); mErr == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}

	// Heartbeat ticker to keep the connection alive.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
