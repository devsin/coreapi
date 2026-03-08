package app

import (
	"net/http"

	"github.com/devsin/coreapi/common/httpx"
	"github.com/devsin/coreapi/internal/auth"
	"github.com/devsin/coreapi/internal/config"
	"github.com/devsin/coreapi/internal/contact"
	"github.com/devsin/coreapi/internal/insights"
	"github.com/devsin/coreapi/internal/media"
	"github.com/devsin/coreapi/internal/notification"
	"github.com/devsin/coreapi/internal/search"
	"github.com/devsin/coreapi/internal/session"
	"github.com/devsin/coreapi/internal/settings"
	"github.com/devsin/coreapi/internal/users"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// RouterDeps holds all dependencies for the HTTP router.
type RouterDeps struct {
	Config   config.Config
	Log      *zap.Logger
	Limiters struct {
		Public   *httpx.RateLimiter
		Tracking *httpx.RateLimiter
	}
	Handlers struct {
		User         *users.Handler
		Settings     *settings.Handler
		Insights     *insights.Handler
		Session      *session.Handler
		Search       *search.Handler
		Contact      *contact.Handler
		Notification *notification.Handler
		Media        *media.Handler
	}
	SessionSvc  *session.Service
	GeoResolver *insights.GeoIPResolver
}

// NewRouter wires routes and middleware.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(httpx.WithRequestID)
	r.Use(httpx.Recover(deps.Log))
	r.Use(httpx.AccessLog(deps.Log))
	r.Use(httpx.CORS(deps.Config.CORS.AllowedOrigins))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	verifier, err := auth.NewVerifier(deps.Config.JWT.JWKSURL, deps.Log)
	if err != nil {
		deps.Log.Fatal("failed to load JWKS", zap.Error(err))
	}

	r.Route("/api", func(api chi.Router) {
		authMiddleware := auth.Middleware(verifier, deps.Log)
		optionalAuthMiddleware := auth.OptionalMiddleware(verifier, deps.Log)

		// Public routes with optional auth (for personalization)
		api.Group(func(public chi.Router) {
			public.Use(optionalAuthMiddleware)
			public.Use(deps.Limiters.Public.Middleware)
			public.Get("/users/discover", deps.Handlers.User.DiscoverUsers)
			public.Get("/users/check-username/{username}", deps.Handlers.User.CheckUsername)
			public.Get("/users/username/{username}", deps.Handlers.User.GetByUsername)
			public.Get("/users/username/{username}/followers", deps.Handlers.User.GetFollowers)
			public.Get("/users/username/{username}/following", deps.Handlers.User.GetFollowing)
			public.Get("/search", deps.Handlers.Search.Search)
			public.Post("/contact", deps.Handlers.Contact.SubmitContactMessage)
		})

		// Rate-limited public tracking endpoints
		api.Group(func(tracking chi.Router) {
			tracking.Use(optionalAuthMiddleware)
			tracking.Use(deps.Limiters.Tracking.Middleware)
			tracking.Post("/insights/profile/{userId}/view", deps.Handlers.Insights.TrackProfileView)
		})

		// Protected routes requiring authentication
		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware)
			protected.Use(session.TrackingMiddleware(deps.SessionSvc, deps.GeoResolver))

			// User routes
			protected.Get("/me", deps.Handlers.User.Me)
			protected.Put("/me", deps.Handlers.User.UpdateMe)

			// Follow routes
			protected.Post("/users/{id}/follow", deps.Handlers.User.FollowUser)
			protected.Delete("/users/{id}/follow", deps.Handlers.User.UnfollowUser)

			// Settings routes
			protected.Get("/settings", deps.Handlers.Settings.GetSettings)
			protected.Put("/settings", deps.Handlers.Settings.UpdateSettings)

			// Insights routes
			protected.Get("/insights/overview", deps.Handlers.Insights.GetOverview)
			protected.Get("/insights/events", deps.Handlers.Insights.GetEvents)
			protected.Get("/insights/geo", deps.Handlers.Insights.GetGeoData)

			// Session routes
			protected.Get("/sessions", deps.Handlers.Session.ListSessions)
			protected.Delete("/sessions", deps.Handlers.Session.DeleteOtherSessions)
			protected.Delete("/sessions/{id}", deps.Handlers.Session.DeleteSession)

			// Notification routes
			protected.Get("/notifications", deps.Handlers.Notification.ListNotifications)
			protected.Get("/notifications/unread-count", deps.Handlers.Notification.GetUnreadCount)
			protected.Get("/notifications/stream", deps.Handlers.Notification.Stream)
			protected.Put("/notifications/{id}/read", deps.Handlers.Notification.MarkRead)
			protected.Put("/notifications/read-all", deps.Handlers.Notification.MarkAllRead)

			// Media upload
			protected.Post("/upload", deps.Handlers.Media.Upload)
		})
	})

	return r
}
