package queue

import (
	"context"
)

type MemoryQueue struct {
	items chan Job
}

func NewMemoryQueue(buffer int) *MemoryQueue {
	if buffer < 1 {
		buffer = 1
	}

	return &MemoryQueue{
		items: make(chan Job, buffer),
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.items <- job:
		return nil
	}
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (Job, error) {
	select {
	case <-ctx.Done():
		return Job{}, ctx.Err()
	case job := <-q.items:
		return job, nil
	}
}
