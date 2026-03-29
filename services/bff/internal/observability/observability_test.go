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

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var payload struct {
		Webhooks map[string]int64 `json:"webhooks"`
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
	if payload.Webhooks["replay:processed"] != 1 {
		t.Fatalf("expected replay processed counter 1, got %d", payload.Webhooks["replay:processed"])
	}
}
