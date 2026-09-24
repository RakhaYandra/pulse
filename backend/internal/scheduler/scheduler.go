package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/RakhaYandra/pulse/internal/db"
	"github.com/RakhaYandra/pulse/pkg/logger"
)

type Job struct {
	MonitorID string `json:"monitor_id"`
}

const queueKey = "pulse:jobs"

// QueueKey is shared with workers so both ends use the same Redis list.
func QueueKey() string { return queueKey }

func Run(log *logger.Logger) error {
	pool, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr()})
	defer rdb.Close()

	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		enqueueDue(pool, rdb, log)
		<-tick.C
	}
}

func enqueueDue(pool *sql.DB, rdb *redis.Client, log *logger.Logger) {
	ctx := context.Background()
	rows, err := pool.Query(`SELECT id FROM monitors WHERE is_active
		AND (last_checked_at IS NULL OR last_checked_at + (interval_seconds || ' seconds')::interval < now())`)
	if err != nil {
		log.Error("due scan failed", "err", err)
		return
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		b, _ := json.Marshal(Job{MonitorID: id})
		if err := rdb.RPush(ctx, queueKey, b).Err(); err != nil {
			log.Error("enqueue failed", "err", err)
			continue
		}
		n++
	}
	if n > 0 {
		log.Info("enqueued jobs", "count", itoa(n))
	}
}

func redisAddr() string {
	if a := os.Getenv("REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
