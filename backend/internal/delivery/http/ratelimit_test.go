package http

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var mu sync.Mutex
	count := map[string]int{}
	allow := func(ctx context.Context, class, ip string) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		count[ip]++
		return count[ip] <= 2, nil // budget 2 per IP
	}
	r := gin.New()
	r.Use(Limiter{Allow: allow, Class: ClassAPI, Limit: 2}.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	allowed, rejected := 0, 0
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		r.ServeHTTP(w, req)
		switch w.Code {
		case 200:
			allowed++
		case 429:
			rejected++
		}
	}
	if allowed != 2 || rejected != 3 {
		t.Fatalf("want 2 allowed + 3 rejected, got %d + %d", allowed, rejected)
	}
}

func TestLimiterFailOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Limiter{Allow: func(ctx context.Context, class, ip string) (bool, error) {
		return false, context.DeadlineExceeded // backend down → allow
	}, Class: ClassAPI, Limit: 1}.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Code != 200 {
		t.Fatalf("fail-open expected 200, got %d", w.Code)
	}
}
