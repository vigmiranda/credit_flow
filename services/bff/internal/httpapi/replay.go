package httpapi

import (
	"strings"
	"sync"
	"time"
)

type webhookReplayProtector struct {
	mu     sync.Mutex
	events map[string]time.Time
}

func newWebhookReplayProtector() *webhookReplayProtector {
	return &webhookReplayProtector{
		events: make(map[string]time.Time),
	}
}

func (p *webhookReplayProtector) Mark(eventID string, expiresAt time.Time) bool {
	key := strings.TrimSpace(eventID)
	if key == "" {
		return false
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
		return true
	}

	p.events[key] = expiresAt
	return false
}

func (p *webhookReplayProtector) Release(eventID string) {
	key := strings.TrimSpace(eventID)
	if key == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.events, key)
}
