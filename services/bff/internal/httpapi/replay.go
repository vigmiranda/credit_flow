package httpapi

import (
	"context"
	"strings"
	"sync"
	"time"
)

type WebhookReplayStore interface {
	Mark(ctx context.Context, eventID string, expiresAt time.Time) (bool, error)
	Release(ctx context.Context, eventID string) error
}

type memoryWebhookReplayStore struct {
	mu     sync.Mutex
	events map[string]time.Time
}

func NewMemoryWebhookReplayStore() WebhookReplayStore {
	return &memoryWebhookReplayStore{
		events: make(map[string]time.Time),
	}
}

func (p *memoryWebhookReplayStore) Mark(_ context.Context, eventID string, expiresAt time.Time) (bool, error) {
	key := strings.TrimSpace(eventID)
	if key == "" {
		return false, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now().UTC()
	for storedID, storedExpiry := range p.events {
		if now.After(storedExpiry) {
			delete(p.events, storedID)
		}
	}

	if storedExpiry, exists := p.events[key]; exists && now.Before(storedExpiry) {
		return true, nil
	}

	p.events[key] = expiresAt
	return false, nil
}

func (p *memoryWebhookReplayStore) Release(_ context.Context, eventID string) error {
	key := strings.TrimSpace(eventID)
	if key == "" {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.events, key)
	return nil
}
