package httpx

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

// WithRequestID attaches a request ID to the context and response.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AccessLog logs request/response details.
func AccessLog(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(recorder, r)
			duration := time.Since(start)

			reqID, _ := r.Context().Value(requestIDKey).(string) //nolint:errcheck // type assertion ok is intentional
			log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", recorder.status),
				zap.Int64("bytes", recorder.size),
				zap.Duration("duration", duration),
				zap.String("request_id", reqID),
			)
		})
	}
}

// Recover guards against panics in handlers.
func Recover(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", zap.Any("error", rec), zap.ByteString("stack", debug.Stack()))
					Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS applies a simple allowlist policy when origins are provided.
// Use "*" to allow all origins.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			break
		}
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// No origin header - not a CORS request
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed
			if !allowAll {
				if _, ok := allowed[origin]; !ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight OPTIONS request - return immediately
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int64
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += int64(n)
	return n, err
}

// Flush implements http.Flusher, required for SSE streaming through the logging middleware.
func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
