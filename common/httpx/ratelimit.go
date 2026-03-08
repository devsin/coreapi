package httpx

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter provides IP-based rate limiting using a token bucket algorithm.
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     int           // tokens added per interval
	interval time.Duration // how often tokens are added
	burst    int           // max tokens (bucket size)
	stopCh   chan struct{}
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter that allows `rate` requests per `interval` with a burst capacity.
// Example: NewRateLimiter(60, time.Minute, 10) → 60 req/min steady, burst of 10.
func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		interval: interval,
		burst:    burst,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Allow checks if the given IP is within rate limits. Returns true if allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:   float64(rl.burst) - 1, // consume one token
			lastSeen: now,
		}
		return true
	}

	// Replenish tokens based on elapsed time
	elapsed := now.Sub(v.lastSeen)
	tokensToAdd := elapsed.Seconds() * (float64(rl.rate) / rl.interval.Seconds())
	v.tokens += tokensToAdd
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.lastSeen = now

	if v.tokens >= 1 {
		v.tokens--
		return true
	}

	return false
}

// Middleware returns a chi-compatible middleware that rate limits by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !rl.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Stop shuts down the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// cleanup removes stale visitor entries every 5 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// extractIP gets the client IP from an HTTP request (for rate limiting).
func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
