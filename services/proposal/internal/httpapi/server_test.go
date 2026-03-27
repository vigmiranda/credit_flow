package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/proposal/internal/domain"
)

type stubStore struct {
	items map[string]domain.Proposal
}

func newStubStore() *stubStore {
	return &stubStore{items: map[string]domain.Proposal{}}
}

func (s *stubStore) Create(_ context.Context, proposal domain.Proposal) error {
	s.items[proposal.ID] = proposal
	return nil
}

func (s *stubStore) GetByID(_ context.Context, proposalID string) (domain.Proposal, error) {
	proposal, ok := s.items[proposalID]
	if !ok {
		return domain.Proposal{}, domain.ErrProposalNotFound
	}

	return proposal, nil
}

func (s *stubStore) UpdateStatus(_ context.Context, proposalID, status string, updatedAt time.Time) (domain.Proposal, error) {
	proposal, ok := s.items[proposalID]
	if !ok {
		return domain.Proposal{}, domain.ErrProposalNotFound
	}

	proposal.Status = status
	proposal.UpdatedAt = updatedAt
	s.items[proposalID] = proposal
	return proposal, nil
}

func TestCreateProposal(t *testing.T) {
	srv := NewServer(newStubStore())
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals", nil)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.Code)
	}

	var proposal domain.Proposal
	if err := json.NewDecoder(resp.Body).Decode(&proposal); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if proposal.ID == "" {
		t.Fatal("expected proposal id to be generated")
	}
}

func TestUpdateProposalStatusRejectsInvalidStatus(t *testing.T) {
	store := newStubStore()
	proposal := domain.NewProposal("corr-test", time.Now().UTC())
	_ = store.Create(context.Background(), proposal)

	srv := NewServer(store)
	body := bytes.NewBufferString(`{"status":"unknown"}`)
	req := httptest.NewRequest(http.MethodPatch, "/internal/proposals/"+proposal.ID+"/status", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
}
