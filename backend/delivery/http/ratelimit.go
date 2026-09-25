package http

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Limiter is per-IP token-bucket rate limiting middleware.
// In-memory: with multiple api replicas each enforces its own budget
// (documented limitation; use a Redis bucket for exact global limits).
type Limiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	r        rate.Limit
	burst    int
	ttl      time.Duration
	lastSeen map[string]time.Time
}

func NewLimiter(rps float64, burst int) *Limiter {
	l := &Limiter{
		visitors: map[string]*rate.Limiter{},
		r:        rate.Limit(rps),
		burst:    burst,
		lastSeen: map[string]time.Time{},
	}
	go l.reap()
	return l
}

func (l *Limiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.visitors[ip]
	if !ok {
		v = rate.NewLimiter(l.r, l.burst)
		l.visitors[ip] = v
	}
	l.lastSeen[ip] = time.Now()
	return v
}

// reap drops idle buckets every minute to bound memory.
func (l *Limiter) reap() {
	for range time.Tick(time.Minute) {
		cutoff := time.Now().Add(-10 * time.Minute)
		l.mu.Lock()
		for ip, t := range l.lastSeen {
			if t.Before(cutoff) {
				delete(l.visitors, ip)
				delete(l.lastSeen, ip)
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(c *gin.Context) string {
	if h := c.GetHeader("X-Forwarded-For"); h != "" {
		return h
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// Middleware rejects over-budget requests with 429 + Retry-After.
func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.get(clientIP(c)).Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
