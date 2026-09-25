package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := NewLimiter(2, 2) // 2 req/s, burst 2
	r := gin.New()
	r.Use(l.Middleware())
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	allowed, rejected := 0, 0
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		r.ServeHTTP(w, req)
		if w.Code == 200 {
			allowed++
		} else if w.Code == 429 {
			rejected++
		}
	}
	if allowed != 2 || rejected != 3 {
		t.Fatalf("want 2 allowed + 3 rejected, got %d + %d", allowed, rejected)
	}
}
