package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type webhookAuditPayload struct {
	ProposalID string `json:"proposal_id"`
	DocumentID string `json:"document_id"`
	Provider   string `json:"provider"`
	EventType  string `json:"event_type"`
}

func newWebhookAuditRecord(callbackType, correlationID string, r *http.Request, rawBody []byte, maxAge time.Duration) WebhookAuditRecord {
	now := time.Now().UTC()
	payload := webhookAuditPayload{}
	_ = json.Unmarshal(rawBody, &payload)

	record := WebhookAuditRecord{
		EventID:          strings.TrimSpace(r.Header.Get(headerWebhookEvent)),
		CallbackType:     callbackType,
		CorrelationID:    correlationID,
		ProposalID:       strings.TrimSpace(payload.ProposalID),
		DocumentID:       strings.TrimSpace(payload.DocumentID),
		Provider:         strings.TrimSpace(payload.Provider),
		EventType:        strings.TrimSpace(payload.EventType),
		ReceivedAt:       now.Format(time.RFC3339),
		ProcessingStatus: "received",
	}
	if timestamp, err := parseWebhookTimestamp(r.Header.Get(headerWebhookTime)); err == nil {
		record.ReceivedAt = timestamp.UTC().Format(time.RFC3339)
		if maxAge > 0 {
			record.ExpiresAt = timestamp.UTC().Add(maxAge).Format(time.RFC3339)
		}
	}

	return record
}

func (r WebhookAuditRecord) withReplayStatus(value string) WebhookAuditRecord {
	r.ReplayStatus = strings.TrimSpace(value)
	return r
}

func (r WebhookAuditRecord) withPayload(proposalID, documentID, provider, eventType string) WebhookAuditRecord {
	if strings.TrimSpace(proposalID) != "" {
		r.ProposalID = strings.TrimSpace(proposalID)
	}
	if strings.TrimSpace(documentID) != "" {
		r.DocumentID = strings.TrimSpace(documentID)
	}
	if strings.TrimSpace(provider) != "" {
		r.Provider = strings.TrimSpace(provider)
	}
	if strings.TrimSpace(eventType) != "" {
		r.EventType = strings.TrimSpace(eventType)
	}
	return r
}

func (r WebhookAuditRecord) processed() WebhookAuditRecord {
	r.ProcessingStatus = "processed"
	r.ProcessedAt = time.Now().UTC().Format(time.RFC3339)
	return r
}

func (r WebhookAuditRecord) failed(code, message string) WebhookAuditRecord {
	r.ProcessingStatus = "rejected"
	r.ErrorCode = strings.TrimSpace(code)
	r.ErrorMessage = strings.TrimSpace(message)
	r.ProcessedAt = time.Now().UTC().Format(time.RFC3339)
	return r
}

func validationErrorCode(err error) string {
	switch err {
	case errInvalidWebhookSignature:
		return "invalid_signature"
	case errMissingWebhookEventID:
		return "missing_event_id"
	case errInvalidWebhookTimestamp:
		return "invalid_timestamp"
	case errStaleWebhookTimestamp:
		return "stale_webhook"
	case errInvalidWebhookReplayStore:
		return "replay_store_error"
	default:
		return "invalid_request"
	}
}

func validationErrorMessage(err error) string {
	switch err {
	case errInvalidWebhookSignature:
		return "assinatura do webhook invalida"
	case errMissingWebhookEventID:
		return "header X-Webhook-Event-Id obrigatorio"
	case errInvalidWebhookTimestamp:
		return "header X-Webhook-Timestamp invalido"
	case errStaleWebhookTimestamp:
		return "timestamp do webhook fora da janela aceita"
	case errInvalidWebhookReplayStore:
		return "falha ao persistir deduplicacao do webhook"
	default:
		return "webhook invalido"
	}
}
