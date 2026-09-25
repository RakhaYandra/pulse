package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthChecker struct {
	DB    func(ctx context.Context) error
	Redis func(ctx context.Context) error
}

func NewRouter(h Handler, health HealthChecker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	v1 := r.Group("/api/v1")
	v1.Use(NewLimiter(100.0/60, 100).Middleware())      // 100 req/min/IP global
	authLimiter := NewLimiter(10.0/60, 10).Middleware() // 10 req/min/IP auth
	{
		v1.GET("/health", func(c *gin.Context) {
			dbOK := health.DB(c.Request.Context()) == nil
			redisOK := health.Redis(c.Request.Context()) == nil
			status := http.StatusOK
			if !dbOK || !redisOK {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{"status": map[bool]string{true: "healthy", false: "unhealthy"}[dbOK && redisOK],
				"database": map[bool]string{true: "healthy", false: "down"}[dbOK],
				"redis":    map[bool]string{true: "healthy", false: "down"}[redisOK]})
		})
		v1.POST("/auth/register", authLimiter, h.Register)
		v1.POST("/auth/login", authLimiter, h.Login)
		v1.GET("/auth/me", h.AuthMiddleware(), h.Me)
		m := v1.Group("/monitors", h.AuthMiddleware())
		{
			m.GET("", h.ListMonitors)
			m.POST("", h.CreateMonitor)
			m.GET("/:id", h.GetMonitor)
			m.PATCH("/:id", h.UpdateMonitor)
			m.DELETE("/:id", h.DeleteMonitor)
			m.POST("/:id/pause", h.PauseMonitor)
			m.POST("/:id/resume", h.ResumeMonitor)
			m.GET("/:id/checks", h.Checks)
			m.GET("/:id/incidents", h.MonitorIncidents)
		}
		v1.GET("/incidents", h.AuthMiddleware(), h.Incidents)
		v1.GET("/dashboard/summary", h.AuthMiddleware(), h.Dashboard)
	}
	return r
}
