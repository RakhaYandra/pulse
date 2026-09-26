package redislimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter is a Redis fixed-window rate limiter shared across api replicas.
// Key layout: pulse:rl:<class>:<ip>:<unix-minute>. Fail-open: Redis errors
// allow the request (the API must not die when Redis flaps); callers observe
// process_check-style metrics via the returned error instead.
type Limiter struct {
	RDB *redis.Client
}

func (l Limiter) Allow(ctx context.Context, class, ip string, limit int) (bool, error) {
	window := time.Now().UTC().Format("200601021504")
	key := "pulse:rl:" + class + ":" + ip + ":" + window
	n, err := l.RDB.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if n == 1 {
		_ = l.RDB.Expire(ctx, key, 70*time.Second).Err()
	}
	return n <= int64(limit), nil
}
