package httpapi

import (
	"context"
	"strings"
	"sync"
	"time"
)

type WebhookRateLimitResult struct {
	Allowed   bool      `json:"allowed"`
	Count     int       `json:"count"`
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

type WebhookRateLimitStore interface {
	Increment(ctx context.Context, callbackType, provider string, limit int, window time.Duration) (WebhookRateLimitResult, error)
}

type memoryWebhookRateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]memoryWebhookRateBucket
}

type memoryWebhookRateBucket struct {
	count   int
	resetAt time.Time
}

func NewMemoryWebhookRateLimitStore() WebhookRateLimitStore {
	return &memoryWebhookRateLimitStore{
		buckets: make(map[string]memoryWebhookRateBucket),
	}
}

func (s *memoryWebhookRateLimitStore) Increment(_ context.Context, callbackType, provider string, limit int, window time.Duration) (WebhookRateLimitResult, error) {
	if limit <= 0 || window <= 0 {
		return WebhookRateLimitResult{Allowed: true, Remaining: limit}, nil
	}

	key := buildWebhookRateLimitKey(callbackType, provider)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	for bucketKey, bucket := range s.buckets {
		if !bucket.resetAt.After(now) {
			delete(s.buckets, bucketKey)
		}
	}

	bucket, ok := s.buckets[key]
	if !ok || !bucket.resetAt.After(now) {
		bucket = memoryWebhookRateBucket{
			count:   0,
			resetAt: now.Add(window),
		}
	}
	bucket.count++
	s.buckets[key] = bucket

	remaining := limit - bucket.count
	if remaining < 0 {
		remaining = 0
	}

	return WebhookRateLimitResult{
		Allowed:   bucket.count <= limit,
		Count:     bucket.count,
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   bucket.resetAt,
	}, nil
}

func buildWebhookRateLimitKey(callbackType, provider string) string {
	normalizedType := strings.TrimSpace(callbackType)
	if normalizedType == "" {
		normalizedType = "unknown"
	}

	normalizedProvider := strings.TrimSpace(provider)
	if normalizedProvider == "" {
		normalizedProvider = "unknown"
	}

	return normalizedType + ":" + normalizedProvider
}
