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
	dlqName     string
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
		dlqName:     queueName + ":dlq",
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

func (q *RedisQueue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}

func (q *RedisQueue) DeadLetter(ctx context.Context, job Job) error {
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return q.client.RPush(ctx, q.dlqName, raw).Err()
}

func (q *RedisQueue) DeadLetterLength(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.dlqName).Result()
}

func (q *RedisQueue) ListDeadLetters(ctx context.Context) ([]Job, error) {
	items, err := q.client.LRange(ctx, q.dlqName, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(items))
	for _, item := range items {
		var job Job
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (q *RedisQueue) RequeueDeadLetters(ctx context.Context, proposalID string) (int, error) {
	items, err := q.client.LRange(ctx, q.dlqName, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	selected := make([]string, 0, len(items))
	remaining := make([]string, 0, len(items))
	for _, item := range items {
		var job Job
		if err := json.Unmarshal([]byte(item), &job); err != nil {
			return 0, err
		}
		if proposalID == "" || job.ProposalID == proposalID {
			job.Attempt = 0
			job.EnqueuedAt = ""
			job.LastError = ""
			raw, err := json.Marshal(job)
			if err != nil {
				return 0, err
			}
			selected = append(selected, string(raw))
			continue
		}
		remaining = append(remaining, item)
	}

	pipe := q.client.TxPipeline()
	pipe.Del(ctx, q.dlqName)
	if len(remaining) > 0 {
		values := make([]any, len(remaining))
		for index, item := range remaining {
			values[index] = item
		}
		pipe.RPush(ctx, q.dlqName, values...)
	}
	if len(selected) > 0 {
		values := make([]any, len(selected))
		for index, item := range selected {
			values[index] = item
		}
		pipe.RPush(ctx, q.queueName, values...)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return len(selected), nil
}
