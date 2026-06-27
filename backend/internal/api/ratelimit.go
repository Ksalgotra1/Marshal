package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter tracks one token bucket per client IP.
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	r        rate.Limit
	burst    int
}

func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{visitors: make(map[string]*visitor), r: r, burst: burst}
}

func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.r, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// Cleanup evicts visitors idle longer than idleAfter. Run as a goroutine
// tied to the app's shutdown context.
func (l *IPRateLimiter) Cleanup(ctx context.Context, idleAfter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.mu.Lock()
			for ip, v := range l.visitors {
				if time.Since(v.lastSeen) > idleAfter {
					delete(l.visitors, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}

// Middleware rejects requests over the per-IP rate with 429.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.getLimiter(ip).Allow() {
			w.Header().Set("Retry-After", "3")
			WriteRequestError(w, r, http.StatusTooManyRequests,
				"too many requests, slow down", nil, "client_ip", ip)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP trusts the first hop of X-Forwarded-For, since Render's edge
// proxy sets it correctly. Falls back to RemoteAddr for local dev.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
