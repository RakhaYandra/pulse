package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const Key = "pulse:jobs"

const claimScript = `
if redis.call("SET", KEYS[1], "1", "NX", "EX", ARGV[1]) then
  redis.call("RPUSH", KEYS[2], ARGV[2])
  return 1
else
  return 0
end`

type RedisQueue struct {
	RDB *redis.Client
}

// Enqueue claims the monitor (claim key with TTL) and pushes one job.
// A monitor already queued or recently checked is skipped — this bounds
// the queue even when workers are slower than the scheduler tick.
func (q RedisQueue) Enqueue(ctx context.Context, monitorID string, ttl time.Duration) (bool, error) {
	secs := int(ttl.Seconds())
	if secs < 1 {
		secs = 1
	}
	n, err := q.RDB.Eval(ctx, claimScript, []string{"pulse:queued:" + monitorID, Key}, secs, monitorID).Int()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (q RedisQueue) Release(ctx context.Context, monitorID string) error {
	return q.RDB.Del(ctx, "pulse:queued:"+monitorID).Err()
}

func (q RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (string, error) {
	vals, err := q.RDB.BLPop(ctx, timeout, Key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return vals[1], nil
}

func (q RedisQueue) Depth(ctx context.Context) (int64, error) {
	return q.RDB.LLen(ctx, Key).Result()
}
