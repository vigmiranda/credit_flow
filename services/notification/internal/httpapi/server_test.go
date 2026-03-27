package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"creditflow/services/notification/internal/domain"
)

type stubStore struct {
	items map[string][]domain.Notification
}

func newStubStore() *stubStore {
	return &stubStore{items: map[string][]domain.Notification{}}
}

func (s *stubStore) Create(_ context.Context, notification domain.Notification) error {
	s.items[notification.ProposalID] = append(s.items[notification.ProposalID], notification)
	return nil
}

func (s *stubStore) ListByProposalID(_ context.Context, proposalID string) ([]domain.Notification, error) {
	return s.items[proposalID], nil
}

func TestCreateNotification(t *testing.T) {
	srv := NewServer(newStubStore())
	body := bytes.NewBufferString(`{"channel":"email","template":"proposal_status_changed","recipient":"maria@example.com","message":"Sua proposta foi aprovada.","trigger_status":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/notifications", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}
}
