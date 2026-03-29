package queue

import (
	"context"
	"sync"
)

type MemoryQueue struct {
	items chan Job
	mu    sync.Mutex
	dlq   []Job
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

func (q *MemoryQueue) Length(context.Context) (int64, error) {
	return int64(len(q.items)), nil
}

func (q *MemoryQueue) DeadLetter(_ context.Context, job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.dlq = append(q.dlq, job)
	return nil
}

func (q *MemoryQueue) DeadLetterLength(context.Context) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return int64(len(q.dlq)), nil
}
