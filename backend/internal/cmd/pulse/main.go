package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	deliveryhttp "github.com/RakhaYandra/pulse/internal/delivery/http"
	"github.com/RakhaYandra/pulse/internal/delivery/scheduler"
	"github.com/RakhaYandra/pulse/internal/delivery/worker"
	"github.com/RakhaYandra/pulse/internal/infrastructure/httpcheck"
	"github.com/RakhaYandra/pulse/internal/infrastructure/notify"
	"github.com/RakhaYandra/pulse/internal/infrastructure/postgres"
	"github.com/RakhaYandra/pulse/internal/infrastructure/queue"
	"github.com/RakhaYandra/pulse/internal/infrastructure/redislimit"
	"github.com/RakhaYandra/pulse/internal/infrastructure/security"
	"github.com/RakhaYandra/pulse/internal/pkg/logger"
	"github.com/RakhaYandra/pulse/internal/usecase"
	"github.com/redis/go-redis/v9"
)

func redisAddr() string {
	if a := os.Getenv("REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}

// allowHosts parses PULSE_ALLOW_HOSTS (comma-separated hostnames exempt from
// SSRF IP filtering). Empty by default = strict. Set to "stub" for benchmarks.
func allowHosts() map[string]bool {
	out := map[string]bool{}
	for _, h := range strings.Split(os.Getenv("PULSE_ALLOW_HOSTS"), ",") {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			out[h] = true
		}
	}
	return out
}

func jwtSecret() (string, error) {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s, nil
	}
	return "", fmt.Errorf("JWT_SECRET is required (no default for production safety)")
}

// serveMetrics exposes this process's registry. Every mode (api, worker,
// scheduler) serves its own endpoint — counters are process-local.
func serveMetrics(log_ *logger.Logger) {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9100"
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log_.Error("metrics server failed", "err", err)
		}
	}()
}

func main() {
	mode := flag.String("mode", "api", "run mode: api | worker | scheduler")
	flag.Parse()

	log_ := logger.New(os.Getenv("LOG_LEVEL"))

	pool, err := postgres.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("db connect/migrate: ", err)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr()})
	defer rdb.Close()

	// Infrastructure adapters.
	users := postgres.UserRepo{DB: pool}
	monitors := postgres.MonitorRepo{DB: pool}
	checks := postgres.CheckRepo{DB: pool}
	incidents := postgres.IncidentRepo{DB: pool}
	dash := postgres.DashboardRepo{DB: pool}
	// JWT is only required in api mode (least privilege: workers and
	// scheduler never see the signing secret).
	var tokens security.JWTService
	if *mode == "api" {
		secret, err := jwtSecret()
		if err != nil {
			log.Fatal(err)
		}
		tokens = security.JWTService{Secret: secret}
	}
	jobQueue := queue.RedisQueue{RDB: rdb}
	notifier := notify.TelegramNotifier{
		Log:     log_,
		Token:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		ChatID:  os.Getenv("TELEGRAM_CHAT_ID"),
		Enabled: os.Getenv("TELEGRAM_BOT_TOKEN") != "" && os.Getenv("TELEGRAM_CHAT_ID") != "",
	}

	// Use-case services.
	authSvc := usecase.AuthService{Users: users, Hash: security.BcryptHasher{}, Tokens: tokens}
	monitorSvc := usecase.MonitorService{Monitors: monitors}
	monitoringSvc := usecase.MonitoringService{
		Monitors: monitors, Checks: checks, Incidents: incidents,
		Checker: httpcheck.Checker{Allow: allowHosts()},
	}
	dashSvc := usecase.DashboardService{Dash: dash}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveMetrics(log_)

	switch *mode {
	case "worker":
		log_.Info("starting pulse worker",
			"concurrency", strconv.Itoa(worker.Concurrency()),
			"allowlist", strconv.Itoa(len(allowHosts())))
		worker.Runner{Log: log_, Monitoring: monitoringSvc, Queue: jobQueue, Notifier: notifier}.Run(ctx)
	case "scheduler":
		log_.Info("starting pulse scheduler",
			"tick", scheduler.TickInterval().String())
		scheduler.Runner{Log: log_, Monitors: monitorSvc, Queue: jobQueue}.Run(ctx)
	default:
		h := deliveryhttp.Handler{Auth: authSvc, Monitors: monitorSvc, Dash: dashSvc, Tokens: tokens}
		rl := redislimit.Limiter{RDB: rdb}
		r := deliveryhttp.NewRouter(h, deliveryhttp.HealthChecker{
			DB:    pool.PingContext,
			Redis: func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
		}, deliveryhttp.RateLimit{
			Allow: func(ctx context.Context, class, ip string) (bool, error) {
				limit := deliveryhttp.LimitAPI
				if class == deliveryhttp.ClassAuth {
					limit = deliveryhttp.LimitAuth
				}
				return rl.Allow(ctx, class, ip, limit)
			},
		})
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
