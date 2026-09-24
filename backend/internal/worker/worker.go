package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/RakhaYandra/pulse/internal/checker"
	"github.com/RakhaYandra/pulse/internal/db"
	"github.com/RakhaYandra/pulse/internal/incident"
	"github.com/RakhaYandra/pulse/internal/notify"
	"github.com/RakhaYandra/pulse/internal/scheduler"
	"github.com/RakhaYandra/pulse/pkg/logger"
)

func Run(log *logger.Logger) error {
	pool, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr()})
	defer rdb.Close()
	sender := notify.New(log)

	ctx := context.Background()
	log.Info("worker waiting for jobs")
	for {
		vals, err := rdb.BLPop(ctx, 30*time.Second, scheduler.QueueKey()).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.Error("queue pop failed", "err", err)
			time.Sleep(3 * time.Second)
			continue
		}
		var job scheduler.Job
		if err := json.Unmarshal([]byte(vals[1]), &job); err != nil {
			log.Error("bad job payload, dropped", "err", err)
			continue
		}
		// One monitor failing must never kill the worker or other monitors.
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("job panic recovered", "monitor", job.MonitorID)
				}
			}()
			process(pool, sender, log, job.MonitorID)
		}()
	}
}

func process(pool *sql.DB, sender *notify.Sender, log *logger.Logger, monitorID string) {
	var targetURL string
	var timeoutSec int
	err := pool.QueryRow(`SELECT url,timeout_seconds FROM monitors WHERE id=$1 AND is_active`, monitorID).Scan(&targetURL, &timeoutSec)
	if err != nil {
		return // monitor paused/deleted after enqueue — safe skip
	}
	res := checker.Check(targetURL, timeoutSec)
	transition, text, err := incident.Evaluate(pool, monitorID, res)
	if err != nil {
		log.Error("evaluate failed", "monitor", monitorID, "err", err)
		return
	}
	log.Info("check done", "monitor", monitorID, "status", res.Status)
	if transition != "" {
		sender.Incident(text)
	}
}

func redisAddr() string {
	if a := os.Getenv("REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}
