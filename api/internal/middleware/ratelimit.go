package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// ipRateLimiter is a simple fixed-window per-IP rate limiter.
// Memory: entries for IPs not seen in the last window are cleaned up periodically.
type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*windowEntry
	limit    int
	window   time.Duration
}

type windowEntry struct {
	count     int
	windowEnd time.Time
}

// NewIPRateLimiter creates a limiter allowing up to limit requests per window per IP.
func NewIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	rl := &ipRateLimiter{
		visitors: make(map[string]*windowEntry),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *ipRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.visitors[ip]
	if !ok || time.Now().After(e.windowEnd) {
		rl.visitors[ip] = &windowEntry{count: 1, windowEnd: time.Now().Add(rl.window)}
		return true
	}
	if e.count >= rl.limit {
		return false
	}
	e.count++
	return true
}

// cleanup removes stale entries every 10 minutes to prevent unbounded growth.
func (rl *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for ip, e := range rl.visitors {
			if now.After(e.windowEnd) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an http.Handler that rate-limits by client IP.
// With Cloudflare Tunnel, the real client IP is in CF-Connecting-IP.
func (rl *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from the request.
// Prefers CF-Connecting-IP (set by Cloudflare Tunnel) over RemoteAddr.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}
