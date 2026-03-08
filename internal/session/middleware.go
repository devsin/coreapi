package session

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/devsin/coreapi/internal/auth"
	"github.com/devsin/coreapi/internal/insights"
)

// TrackingMiddleware records the active session on every authenticated request.
// Relies on auth middleware having already set UserClaims in the context.
// Uses in-memory dedup (Service.seen) so only the first request per dedupTTL
// actually touches the database.
func TrackingMiddleware(svc *Service, geo *insights.GeoIPResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if ok && claims.SessionID != "" {
				userID, err := uuid.Parse(claims.UserID)
				if err == nil {
					ip := insights.ExtractIP(r)
					ua := r.UserAgent()
					parsed := geo.ParseUserAgent(ua)

					svc.RecordSession( //nolint:contextcheck // intentional background context for fire-and-forget
						userID, claims.SessionID,
						ip, ua,
						parsed.Browser, parsed.OS, parsed.DeviceType,
					)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
