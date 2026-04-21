package middleware

import (
	"net/http"
	"sync"
	"time"

	"lumina/internal/httpx"

	"github.com/gin-gonic/gin"
)

// ipLimiter is a minimal fixed-window per-IP counter, memory-only.
// Adequate for brute-force protection on /register and /login; not a
// general-purpose rate limiter.
type ipLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	hits   map[string]*ipHitState
}

type ipHitState struct {
	count      int
	windowFrom time.Time
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	return &ipLimiter{
		window: window,
		limit:  limit,
		hits:   make(map[string]*ipHitState),
	}
}

// allow reports whether the IP is under the limit and advances the counter.
func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	st, ok := l.hits[ip]
	if !ok || now.Sub(st.windowFrom) >= l.window {
		l.hits[ip] = &ipHitState{count: 1, windowFrom: now}
		// Opportunistic GC so the map doesn't grow unboundedly under attack.
		if len(l.hits) > 10000 {
			for k, v := range l.hits {
				if now.Sub(v.windowFrom) >= l.window {
					delete(l.hits, k)
				}
			}
		}
		return true
	}
	if st.count >= l.limit {
		return false
	}
	st.count++
	return true
}

// RateLimitPerMinute returns middleware allowing at most `limit` hits per
// minute from a single client IP. On excess, responds 429.
func RateLimitPerMinute(limit int) gin.HandlerFunc {
	limiter := newIPLimiter(limit, time.Minute)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			httpx.Error(c, http.StatusTooManyRequests, httpx.CodeRateLimited, "请求过于频繁，请稍后再试")
			return
		}
		c.Next()
	}
}
