package httpapi

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type WebhookAuditRecord struct {
	EventID            string `json:"event_id"`
	CallbackType       string `json:"callback_type"`
	CorrelationID      string `json:"correlation_id,omitempty"`
	ProposalID         string `json:"proposal_id,omitempty"`
	DocumentID         string `json:"document_id,omitempty"`
	Provider           string `json:"provider,omitempty"`
	EventType          string `json:"event_type,omitempty"`
	ReplayStatus       string `json:"replay_status,omitempty"`
	ProcessingStatus   string `json:"processing_status,omitempty"`
	ErrorCode          string `json:"error_code,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
	ReceivedAt         string `json:"received_at"`
	ProcessedAt        string `json:"processed_at,omitempty"`
	ExpiresAt          string `json:"expires_at,omitempty"`
	RetentionExpiresAt string `json:"retention_expires_at,omitempty"`
	ReplayReleasedAt   string `json:"replay_released_at,omitempty"`
	LastReplayAction   string `json:"last_replay_action,omitempty"`
}

type WebhookAuditFilter struct {
	EventID      string
	CallbackType string
	ProposalID   string
	Limit        int
}

type WebhookAuditStore interface {
	Upsert(ctx context.Context, record WebhookAuditRecord) error
	Get(ctx context.Context, eventID string) (WebhookAuditRecord, bool, error)
	List(ctx context.Context, filter WebhookAuditFilter) ([]WebhookAuditRecord, error)
	MarkReplayReleased(ctx context.Context, eventID string, releasedAt time.Time) error
	CleanupExpired(ctx context.Context, now time.Time) (int, error)
}

type memoryWebhookAuditStore struct {
	mu        sync.Mutex
	records   map[string]WebhookAuditRecord
	retention time.Duration
}

func NewMemoryWebhookAuditStore() WebhookAuditStore {
	return &memoryWebhookAuditStore{
		records:   make(map[string]WebhookAuditRecord),
		retention: 7 * 24 * time.Hour,
	}
}

func (s *memoryWebhookAuditStore) Upsert(_ context.Context, record WebhookAuditRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record = applyAuditRetention(record, s.retention)
	s.records[strings.TrimSpace(record.EventID)] = record
	return nil
}

func (s *memoryWebhookAuditStore) Get(_ context.Context, eventID string) (WebhookAuditRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[strings.TrimSpace(eventID)]
	return record, ok, nil
}

func (s *memoryWebhookAuditStore) List(_ context.Context, filter WebhookAuditFilter) ([]WebhookAuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]WebhookAuditRecord, 0, len(s.records))
	for _, record := range s.records {
		if !matchesAuditFilter(record, filter) {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].ReceivedAt > records[j].ReceivedAt
	})

	if limit := normalizedAuditLimit(filter.Limit); len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

func (s *memoryWebhookAuditStore) MarkReplayReleased(_ context.Context, eventID string, releasedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(eventID)
	record, ok := s.records[key]
	if !ok {
		return nil
	}
	record.ReplayReleasedAt = releasedAt.UTC().Format(time.RFC3339)
	record.LastReplayAction = "released"
	s.records[key] = record
	return nil
}

func (s *memoryWebhookAuditStore) CleanupExpired(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for key, record := range s.records {
		if !auditRecordExpired(record, now) {
			continue
		}
		delete(s.records, key)
		removed++
	}

	return removed, nil
}

func matchesAuditFilter(record WebhookAuditRecord, filter WebhookAuditFilter) bool {
	if value := strings.TrimSpace(filter.EventID); value != "" && record.EventID != value {
		return false
	}
	if value := strings.TrimSpace(filter.CallbackType); value != "" && record.CallbackType != value {
		return false
	}
	if value := strings.TrimSpace(filter.ProposalID); value != "" && record.ProposalID != value {
		return false
	}
	return true
}

func normalizedAuditLimit(value int) int {
	if value <= 0 {
		return 50
	}
	if value > 200 {
		return 200
	}
	return value
}

func applyAuditRetention(record WebhookAuditRecord, retention time.Duration) WebhookAuditRecord {
	if strings.TrimSpace(record.RetentionExpiresAt) != "" {
		return record
	}
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}

	record.RetentionExpiresAt = time.Now().UTC().Add(retention).Format(time.RFC3339)
	return record
}

func auditRecordExpired(record WebhookAuditRecord, now time.Time) bool {
	if strings.TrimSpace(record.RetentionExpiresAt) == "" {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, record.RetentionExpiresAt)
	if err != nil {
		return false
	}

	return !expiresAt.After(now.UTC())
}
