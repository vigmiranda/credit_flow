package queue

import "context"

type Job struct {
	ProposalID    string `json:"proposal_id"`
	CorrelationID string `json:"correlation_id"`
	Attempt       int    `json:"attempt"`
	EnqueuedAt    string `json:"enqueued_at"`
}

type Queue interface {
	Enqueue(ctx context.Context, job Job) error
	Dequeue(ctx context.Context) (Job, error)
}
