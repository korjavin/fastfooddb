package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newRateLimiterStore(r float64, b int) *rateLimiterStore {
	rl := &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		r:        rate.Limit(r),
		b:        b,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiterStore) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if v, ok := rl.limiters[ip]; ok {
		v.lastSeen = time.Now()
		return v.limiter
	}
	l := rate.NewLimiter(rl.r, rl.b)
	rl.limiters[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

func (rl *rateLimiterStore) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.limiters {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a middleware that limits each IP to rps requests per second
// with a burst of burst requests.
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	store := newRateLimiterStore(rps, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !store.get(ip).Allow() {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the client IP from common proxy headers or RemoteAddr.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
