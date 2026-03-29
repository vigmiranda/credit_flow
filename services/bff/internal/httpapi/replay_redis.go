package httpapi

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisWebhookReplayStore struct {
	client *redis.Client
	prefix string
}

func NewRedisWebhookReplayStore(redisURL, prefix string) (WebhookReplayStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &redisWebhookReplayStore{
		client: redis.NewClient(options),
		prefix: strings.TrimSpace(prefix),
	}, nil
}

func (s *redisWebhookReplayStore) Mark(ctx context.Context, eventID string, expiresAt time.Time) (bool, error) {
	key := s.key(eventID)
	if key == "" {
		return false, nil
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}

	stored, err := s.client.SetNX(ctx, key, expiresAt.Format(time.RFC3339), ttl).Result()
	if err != nil {
		return false, err
	}

	return !stored, nil
}

func (s *redisWebhookReplayStore) Release(ctx context.Context, eventID string) error {
	key := s.key(eventID)
	if key == "" {
		return nil
	}

	return s.client.Del(ctx, key).Err()
}

func (s *redisWebhookReplayStore) key(eventID string) string {
	trimmed := strings.TrimSpace(eventID)
	if trimmed == "" {
		return ""
	}

	prefix := strings.TrimSpace(s.prefix)
	if prefix == "" {
		prefix = "bff:webhook:events"
	}
	return prefix + ":" + trimmed
}
