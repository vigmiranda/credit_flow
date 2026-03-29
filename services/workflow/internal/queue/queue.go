package queue

import "context"

type Job struct {
	ProposalID    string `json:"proposal_id"`
	CorrelationID string `json:"correlation_id"`
	Attempt       int    `json:"attempt"`
	EnqueuedAt    string `json:"enqueued_at"`
	LastError     string `json:"last_error,omitempty"`
}

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context) (Job, error)
	Length(ctx context.Context) (int64, error)
	DeadLetter(ctx context.Context, job Job) error
	DeadLetterLength(ctx context.Context) (int64, error)
}
