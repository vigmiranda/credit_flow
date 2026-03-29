package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisWebhookAuditStore struct {
	client    *redis.Client
	prefix    string
	retention time.Duration
}

func NewRedisWebhookAuditStore(redisURL, prefix string, retention time.Duration) (WebhookAuditStore, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}

	return &redisWebhookAuditStore{
		client:    redis.NewClient(options),
		prefix:    strings.TrimSpace(prefix),
		retention: retention,
	}, nil
}

func (s *redisWebhookAuditStore) Upsert(ctx context.Context, record WebhookAuditRecord) error {
	key := s.recordKey(record.EventID)
	if key == "" {
		return nil
	}

	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}

	score := float64(time.Now().UTC().Unix())
	if parsed, err := time.Parse(time.RFC3339, record.ReceivedAt); err == nil {
		score = float64(parsed.Unix())
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, key, raw, s.retention)
	pipe.ZAdd(ctx, s.indexKey(), redis.Z{Score: score, Member: strings.TrimSpace(record.EventID)})
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisWebhookAuditStore) Get(ctx context.Context, eventID string) (WebhookAuditRecord, bool, error) {
	key := s.recordKey(eventID)
	if key == "" {
		return WebhookAuditRecord{}, false, nil
	}

	raw, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return WebhookAuditRecord{}, false, nil
	}
	if err != nil {
		return WebhookAuditRecord{}, false, err
	}

	var record WebhookAuditRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return WebhookAuditRecord{}, false, err
	}

	return record, true, nil
}

func (s *redisWebhookAuditStore) List(ctx context.Context, filter WebhookAuditFilter) ([]WebhookAuditRecord, error) {
	limit := normalizedAuditLimit(filter.Limit)
	ids, err := s.client.ZRevRange(ctx, s.indexKey(), 0, int64(limit*5)).Result()
	if err != nil {
		return nil, err
	}

	records := make([]WebhookAuditRecord, 0, limit)
	for _, eventID := range ids {
		record, ok, err := s.Get(ctx, eventID)
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = s.client.ZRem(ctx, s.indexKey(), eventID).Err()
			continue
		}
		if !matchesAuditFilter(record, filter) {
			continue
		}
		records = append(records, record)
		if len(records) >= limit {
			break
		}
	}

	return records, nil
}

func (s *redisWebhookAuditStore) MarkReplayReleased(ctx context.Context, eventID string, releasedAt time.Time) error {
	record, ok, err := s.Get(ctx, eventID)
	if err != nil || !ok {
		return err
	}
	record.ReplayReleasedAt = releasedAt.UTC().Format(time.RFC3339)
	record.LastReplayAction = "released"
	return s.Upsert(ctx, record)
}

func (s *redisWebhookAuditStore) recordKey(eventID string) string {
	trimmed := strings.TrimSpace(eventID)
	if trimmed == "" {
		return ""
	}
	return s.basePrefix() + ":record:" + trimmed
}

func (s *redisWebhookAuditStore) indexKey() string {
	return s.basePrefix() + ":index"
}

func (s *redisWebhookAuditStore) basePrefix() string {
	if strings.TrimSpace(s.prefix) == "" {
		return "bff:webhook:audit"
	}
	return strings.TrimSpace(s.prefix)
}
