package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerIncludesWebhookCounters(t *testing.T) {
	metrics := NewMetrics("bff")
	metrics.RecordWebhook("credit", "partner-a", "processed")
	metrics.RecordWebhook("credit", "partner-a", "duplicate_ignored")
	metrics.RecordWebhookCleanup(3)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var payload struct {
		Webhooks       map[string]int64 `json:"webhooks"`
		WebhookCleanup map[string]int64 `json:"webhook_cleanup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Webhooks["type:credit"] != 2 {
		t.Fatalf("expected type counter 2, got %d", payload.Webhooks["type:credit"])
	}
	if payload.Webhooks["provider:partner-a"] != 2 {
		t.Fatalf("expected provider counter 2, got %d", payload.Webhooks["provider:partner-a"])
	}
	if payload.Webhooks["outcome:processed"] != 1 {
		t.Fatalf("expected processed counter 1, got %d", payload.Webhooks["outcome:processed"])
	}
	if payload.Webhooks["type:credit|outcome:duplicate_ignored"] != 1 {
		t.Fatalf("expected duplicate_ignored typed counter 1, got %d", payload.Webhooks["type:credit|outcome:duplicate_ignored"])
	}
	if payload.WebhookCleanup["runs"] != 1 {
		t.Fatalf("expected cleanup runs 1, got %d", payload.WebhookCleanup["runs"])
	}
	if payload.WebhookCleanup["removed_total"] != 3 {
		t.Fatalf("expected cleanup removed total 3, got %d", payload.WebhookCleanup["removed_total"])
	}
}
