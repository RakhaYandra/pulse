package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/RakhaYandra/pulse/internal/auth"
	"github.com/RakhaYandra/pulse/internal/db"
	"github.com/RakhaYandra/pulse/internal/monitor"
	"github.com/RakhaYandra/pulse/internal/read"
	"github.com/RakhaYandra/pulse/internal/scheduler"
	"github.com/RakhaYandra/pulse/internal/worker"
	"github.com/RakhaYandra/pulse/pkg/logger"
)

func main() {
	mode := flag.String("mode", "api", "run mode: api | worker | scheduler")
	flag.Parse()

	log_ := logger.New(os.Getenv("LOG_LEVEL"))

	switch *mode {
	case "worker":
		log_.Info("starting pulse worker")
		if err := worker.Run(log_); err != nil {
			log.Fatal(err)
		}
	case "scheduler":
		log_.Info("starting pulse scheduler")
		if err := scheduler.Run(log_); err != nil {
			log.Fatal(err)
		}
	default:
		pool, err := db.Connect(os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatal("db connect/migrate: ", err)
		}
		defer pool.Close()

		ah := auth.NewHandler(pool)
		mh := monitor.NewHandler(pool)
		rh := read.NewHandler(pool)
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
		v1 := r.Group("/api/v1")
		{
			v1.GET("/health", func(c *gin.Context) {
				if err := pool.Ping(); err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "down"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "healthy", "database": "healthy", "redis": "todo"})
			})
			v1.POST("/auth/register", ah.Register)
			v1.POST("/auth/login", ah.Login)
			v1.GET("/auth/me", auth.Middleware(), ah.Me)
			m := v1.Group("/monitors", auth.Middleware())
			{
				m.GET("", mh.List)
				m.POST("", mh.Create)
				m.GET("/:id", mh.Get)
				m.PATCH("/:id", mh.Update)
				m.DELETE("/:id", mh.Delete)
				m.POST("/:id/pause", mh.Pause)
				m.POST("/:id/resume", mh.Resume)
				m.GET("/:id/checks", rh.Checks)
				m.GET("/:id/incidents", rh.MonitorIncidents)
			}
			v1.GET("/incidents", auth.Middleware(), rh.Incidents)
			v1.GET("/dashboard/summary", auth.Middleware(), rh.Dashboard)
		}
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log_.Info("pulse api listening on :" + port)
		if err := r.Run(":" + port); err != nil {
			log.Fatal(err)
		}
	}
}
