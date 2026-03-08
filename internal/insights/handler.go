package insights

import (
	"context"
	"math"
	"net/http"
	"strconv"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for insights operations.
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new insights handler.
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// GetOverview handles GET /api/insights/overview?period=7d|30d|90d.
func (h *Handler) GetOverview(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid user ID")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}
	switch period {
	case "7d", "30d", "90d":
		// valid
	default:
		httpx.Error(w, http.StatusBadRequest, "bad_request", "period must be 7d, 30d, or 90d")
		return
	}

	overview, err := h.svc.GetOverview(r.Context(), userID, period)
	if err != nil {
		h.log.Error("failed to get insights overview", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load insights")
		return
	}

	httpx.JSON(w, http.StatusOK, overview)
}

// TrackProfileView handles POST /api/insights/profile/{userId}/view.
func (h *Handler) TrackProfileView(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "userId")
	profileUserID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user ID")
		return
	}

	ip := ExtractIP(r)
	userAgent := r.UserAgent()
	referrer := r.Header.Get("Referer")

	go h.svc.RecordProfileView(context.WithoutCancel(r.Context()), profileUserID, ip, userAgent, referrer)

	w.WriteHeader(http.StatusNoContent)
}

// GetEvents handles GET /api/insights/events?limit=&offset=.
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid user ID")
		return
	}

	limit := int32(50)
	offset := int32(0)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = int32(v) //nolint:gosec // bounds checked above (1-100)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 && v <= math.MaxInt32 {
			offset = int32(v) //nolint:gosec // bounds checked above
		}
	}

	events, err := h.svc.GetEvents(r.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("failed to get events", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load events")
		return
	}

	httpx.JSON(w, http.StatusOK, events)
}

// GetGeoData handles GET /api/insights/geo.
func (h *Handler) GetGeoData(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid user ID")
		return
	}

	geo, err := h.svc.GetGeoData(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get geo data", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load geo data")
		return
	}

	httpx.JSON(w, http.StatusOK, geo)
}
