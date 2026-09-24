package queue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const Key = "pulse:jobs"

type RedisQueue struct {
	RDB *redis.Client
}

func (q RedisQueue) Enqueue(ctx context.Context, monitorID string) error {
	return q.RDB.RPush(ctx, Key, monitorID).Err()
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
