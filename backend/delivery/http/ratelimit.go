package http

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// AllowFunc checks a rate budget; implemented by infrastructure/redislimit.
// Error forces fail-open (allow) so Redis outages don't kill the API.
type AllowFunc func(ctx context.Context, class, ip string) (bool, error)

// Limiter enforces per-IP, per-class budgets via an injected backend.
type Limiter struct {
	Allow AllowFunc
	Class string
	Limit int
}

func clientIP(c *gin.Context) string {
	// X-Forwarded-For is spoofable; trust it only behind a known proxy.
	if os.Getenv("TRUST_PROXY") == "1" {
		if h := c.GetHeader("X-Forwarded-For"); h != "" {
			return h
		}
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// Middleware rejects over-budget requests with 429 + Retry-After.
func (l Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, err := l.Allow(c.Request.Context(), l.Class, clientIP(c))
		if err != nil || ok {
			c.Next()
			return
		}
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
	}
}

// Classes and budgets (requests/minute/IP).
const (
	ClassAuth = "auth"
	ClassAPI  = "api"
)

const (
	LimitAuth = 10
	LimitAPI  = 100
)
