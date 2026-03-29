package httpapi

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisWebhookRateLimitStore struct {
	client *redis.Client
	prefix string
}

func NewRedisWebhookRateLimitStore(redisURL, prefix string) (WebhookRateLimitStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &redisWebhookRateLimitStore{
		client: redis.NewClient(options),
		prefix: strings.TrimSpace(prefix),
	}, nil
}

func (s *redisWebhookRateLimitStore) Increment(ctx context.Context, callbackType, provider string, limit int, window time.Duration) (WebhookRateLimitResult, error) {
	if limit <= 0 || window <= 0 {
		return WebhookRateLimitResult{Allowed: true, Remaining: limit}, nil
	}

	key := s.key(callbackType, provider)
	if key == "" {
		return WebhookRateLimitResult{Allowed: true, Remaining: limit}, nil
	}

	result, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return WebhookRateLimitResult{}, err
	}

	if result == 1 {
		if err := s.client.Expire(ctx, key, window).Err(); err != nil {
			return WebhookRateLimitResult{}, err
		}
	}

	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return WebhookRateLimitResult{}, err
	}
	if ttl <= 0 {
		ttl = window
	}

	remaining := limit - int(result)
	if remaining < 0 {
		remaining = 0
	}

	return WebhookRateLimitResult{
		Allowed:   int(result) <= limit,
		Count:     int(result),
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   time.Now().UTC().Add(ttl),
	}, nil
}

func (s *redisWebhookRateLimitStore) key(callbackType, provider string) string {
	prefix := strings.TrimSpace(s.prefix)
	if prefix == "" {
		prefix = "bff:webhook:ratelimit"
	}

	return prefix + ":" + sanitizeRateLimitSegment(callbackType) + ":" + sanitizeRateLimitSegment(provider)
}

func sanitizeRateLimitSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer(" ", "_", "/", "_", ":", "_")
	return replacer.Replace(trimmed)
}
