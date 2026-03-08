package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/devsin/coreapi/common/httpx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.uber.org/zap"
)

type contextKey string

const userClaimsKey contextKey = "user_claims"

// UserClaims represents the authenticated user details extracted from the JWT.
type UserClaims struct {
	UserID    string
	Email     string
	SessionID string
}

// Verifier wraps a jwk cache for JWKS-based verification.
type Verifier struct {
	cache   *jwk.Cache
	jwksURL string
}

// NewVerifier loads JWKS from the provided URL and configures periodic refresh.
func NewVerifier(jwksURL string, log *zap.Logger) (*Verifier, error) {
	ctx := context.Background()
	cache := jwk.NewCache(ctx)

	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(30*time.Minute)); err != nil {
		return nil, err
	}

	// Prime the cache once; log a warning if it fails but allow startup.
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		log.Warn("initial jwks fetch failed", zap.Error(err))
	}

	return &Verifier{cache: cache, jwksURL: jwksURL}, nil
}

// Middleware validates Supabase JWTs using ES256 + JWKS.
// Rejects requests without valid tokens.
// Supports both Authorization header and ?token= query parameter (for SSE).
func Middleware(verifier *Verifier, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			claims := &jwtRegisteredClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, verifier.keyfunc)
			if err != nil {
				log.Warn("jwt parse failed", zap.Error(err))
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}
			if !token.Valid {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			if claims.Subject == "" {
				log.Warn("jwt missing subject")
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			uc := UserClaims{UserID: claims.Subject, Email: claims.Email, SessionID: claims.SessionID}
			ctx := context.WithValue(r.Context(), userClaimsKey, uc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware validates Supabase JWTs if present, but allows unauthenticated requests.
// Use this for public routes where authentication is optional (e.g., for personalization).
func OptionalMiddleware(verifier *Verifier, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				// No token - proceed without authentication
				next.ServeHTTP(w, r)
				return
			}
			claims := &jwtRegisteredClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, verifier.keyfunc)
			if err != nil {
				log.Warn("jwt parse failed (optional)", zap.Error(err))
				// Invalid token - proceed without authentication
				next.ServeHTTP(w, r)
				return
			}
			if !token.Valid || claims.Subject == "" {
				// Invalid token - proceed without authentication
				next.ServeHTTP(w, r)
				return
			}

			uc := UserClaims{UserID: claims.Subject, Email: claims.Email, SessionID: claims.SessionID}
			ctx := context.WithValue(r.Context(), userClaimsKey, uc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext extracts UserClaims set by the middleware.
func FromContext(ctx context.Context) (UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(UserClaims)
	return claims, ok
}

type jwtRegisteredClaims struct {
	jwt.RegisteredClaims
	Email     string `json:"email"`
	SessionID string `json:"session_id"`
}

// keyfunc resolves the verification key from the JWKS cache.
func (v *Verifier) keyfunc(token *jwt.Token) (interface{}, error) {
	ctx := context.Background()
	set, err := v.cache.Get(ctx, v.jwksURL)
	if err != nil {
		return nil, err
	}

	kid, _ := token.Header["kid"].(string) //nolint:errcheck // type assertion ok is intentional
	if kid == "" {
		return nil, fmt.Errorf("missing kid header")
	}

	key, ok := set.LookupKeyID(kid)
	if !ok {
		return nil, fmt.Errorf("kid %s not found", kid)
	}

	var raw interface{}
	if err := key.Raw(&raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// extractToken gets the JWT from the Authorization header or ?token= query parameter.
// Header takes priority; the query param fallback supports SSE (EventSource can't set headers).
func extractToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return r.URL.Query().Get("token")
}
