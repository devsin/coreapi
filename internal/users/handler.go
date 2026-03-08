package users

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for user operations.
type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Me handles GET /api/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	user, err := h.svc.GetOrCreateMe(r.Context(), claims)
	if err != nil {
		h.log.Error("get or create me failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not process user")
		return
	}

	httpx.JSON(w, http.StatusOK, user)
}

// UpdateMe handles PUT /api/me.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	user, err := h.svc.UpdateMe(r.Context(), claims, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameTaken):
			h.log.Warn("username taken", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusConflict, "username_taken", "this username is already taken")
		case errors.Is(err, ErrUsernameInvalid):
			h.log.Warn("username invalid", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusBadRequest, "username_invalid", "username must be 3-20 characters, letters, numbers, underscores only")
		case errors.Is(err, ErrUsernameReserved):
			h.log.Warn("username reserved", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusBadRequest, "username_reserved", "this username is reserved")
		default:
			h.log.Error("update me failed", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not update profile")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, user)
}

// CheckUsername handles GET /api/users/check-username/{username}.
func (h *Handler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	available, reason, err := h.svc.CheckUsernameAvailability(r.Context(), username)
	if err != nil {
		h.log.Error("check username failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not check username")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"available": available,
		"reason":    reason,
	})
}

// GetByUsername handles GET /api/users/username/{username}.
func (h *Handler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	// Get viewer ID if authenticated (for is_following flag)
	var viewerID *uuid.UUID
	if claims, ok := auth.FromContext(r.Context()); ok {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			viewerID = &id
		}
	}

	user, err := h.svc.GetUserWithStats(r.Context(), username, viewerID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.log.Warn("user not found", zap.String("username", username), zap.String("path", r.URL.Path))
			httpx.Error(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		h.log.Error("get user by username failed", zap.String("username", username), zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not get user")
		return
	}

	httpx.JSON(w, http.StatusOK, user)
}

// FollowUser handles POST /api/users/{id}/follow.
func (h *Handler) FollowUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	followerID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	followingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	err = h.svc.FollowUser(r.Context(), followerID, followingID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCannotFollowSelf):
			h.log.Warn("cannot follow self", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusBadRequest, "cannot_follow_self", "you cannot follow yourself")
		case errors.Is(err, ErrUserNotFound):
			h.log.Warn("follow target not found", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusNotFound, "user_not_found", "user not found")
		default:
			h.log.Error("follow user failed", zap.String("path", r.URL.Path), zap.Error(err))
			httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not follow user")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]bool{"following": true})
}

// UnfollowUser handles DELETE /api/users/{id}/follow.
func (h *Handler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}

	followerID, err := uuid.Parse(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	followingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}

	err = h.svc.UnfollowUser(r.Context(), followerID, followingID)
	if err != nil {
		h.log.Error("unfollow user failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not unfollow user")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]bool{"following": false})
}

// GetFollowers handles GET /api/users/username/{username}/followers.
func (h *Handler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	limit, offset := parsePagination(r, 20, 100)

	// Get current user ID if authenticated (for determining follow status)
	var currentUserID *uuid.UUID
	if claims, ok := auth.FromContext(r.Context()); ok {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			currentUserID = &id
		}
	}

	resp, err := h.svc.GetFollowers(r.Context(), username, currentUserID, limit, offset)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.log.Warn("user not found", zap.String("username", username), zap.String("path", r.URL.Path))
			httpx.Error(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		if errors.Is(err, ErrProfilePrivate) {
			h.log.Warn("profile private", zap.String("username", username), zap.String("path", r.URL.Path))
			httpx.Error(w, http.StatusForbidden, "profile_private", "this profile is private")
			return
		}
		h.log.Error("get followers failed", zap.String("username", username), zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not get followers")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// GetFollowing handles GET /api/users/username/{username}/following.
func (h *Handler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	limit, offset := parsePagination(r, 20, 100)

	// Get current user ID if authenticated (for determining follow status)
	var currentUserID *uuid.UUID
	if claims, ok := auth.FromContext(r.Context()); ok {
		if id, err := uuid.Parse(claims.UserID); err == nil {
			currentUserID = &id
		}
	}

	resp, err := h.svc.GetFollowing(r.Context(), username, currentUserID, limit, offset)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.log.Warn("user not found", zap.String("username", username), zap.String("path", r.URL.Path))
			httpx.Error(w, http.StatusNotFound, "user_not_found", "user not found")
			return
		}
		if errors.Is(err, ErrProfilePrivate) {
			h.log.Warn("profile private", zap.String("username", username), zap.String("path", r.URL.Path))
			httpx.Error(w, http.StatusForbidden, "profile_private", "this profile is private")
			return
		}
		h.log.Error("get following failed", zap.String("username", username), zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not get following")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// DiscoverUsers handles GET /api/users/discover.
func (h *Handler) DiscoverUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 20, 50)
	sort := r.URL.Query().Get("sort") // "popular" (default) or "new"

	claims, _ := auth.FromContext(r.Context())
	var currentUserID *uuid.UUID
	if claims.UserID != "" {
		if uid, err := uuid.Parse(claims.UserID); err == nil {
			currentUserID = &uid
		}
	}

	resp, err := h.svc.DiscoverUsers(r.Context(), sort, currentUserID, limit, offset)
	if err != nil {
		h.log.Error("discover users failed", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load users")
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

// parsePagination extracts limit and offset from query params with defaults.
func parsePagination(r *http.Request, defaultLimit, maxLimit int32) (limit, offset int32) {
	limit = defaultLimit
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed) //nolint:gosec // clamped to maxLimit below
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 && parsed <= math.MaxInt32 {
			offset = int32(parsed) //nolint:gosec // bounds checked above
		}
	}

	return limit, offset
}
