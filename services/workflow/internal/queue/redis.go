package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client      *redis.Client
	queueName   string
	blockPeriod time.Duration
}

func NewRedisQueue(redisURL, queueName string, blockPeriod time.Duration) (*RedisQueue, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &RedisQueue{
		client:      redis.NewClient(options),
		queueName:   queueName,
		blockPeriod: blockPeriod,
	}, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return q.client.RPush(ctx, q.queueName, raw).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (Job, error) {
	for {
		if err := ctx.Err(); err != nil {
			return Job{}, err
		}

		items, err := q.client.BLPop(ctx, q.blockPeriod, q.queueName).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return Job{}, err
		}
		if len(items) != 2 {
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(items[1]), &job); err != nil {
			return Job{}, err
		}

		return job, nil
	}
}
