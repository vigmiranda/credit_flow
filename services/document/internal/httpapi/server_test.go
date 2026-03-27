package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/document/internal/domain"
)

type stubStore struct {
	items map[string][]domain.Document
}

func newStubStore() *stubStore {
	return &stubStore{items: map[string][]domain.Document{}}
}

func (s *stubStore) Create(_ context.Context, document domain.Document) error {
	s.items[document.ProposalID] = append(s.items[document.ProposalID], document)
	return nil
}

func (s *stubStore) ListByProposalID(_ context.Context, proposalID string) ([]domain.Document, error) {
	return s.items[proposalID], nil
}

func (s *stubStore) MarkUploaded(_ context.Context, proposalID, documentID string, uploadedAt time.Time) (domain.Document, error) {
	documents := s.items[proposalID]
	for index, document := range documents {
		if document.ID == documentID {
			updated := document.MarkUploaded(uploadedAt)
			documents[index] = updated
			s.items[proposalID] = documents
			return updated, nil
		}
	}

	return domain.Document{}, domain.ErrDocumentNotFound
}

func TestCreateUploadURL(t *testing.T) {
	srv := NewServer(newStubStore(), "http://localhost:4566/mock-upload")
	body := bytes.NewBufferString(`{"document_type":"id_front","file_name":"rg.jpg","content_type":"image/jpeg"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/documents/upload-url", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var document domain.Document
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if document.UploadURL == "" {
		t.Fatal("expected upload url")
	}
}

func TestMarkDocumentReceived(t *testing.T) {
	store := newStubStore()
	document := domain.NewDocument("prop_123", domain.UploadRequest{
		DocumentType: "id_front",
		FileName:     "rg.jpg",
		ContentType:  "image/jpeg",
	}, "http://localhost:4566/mock-upload", time.Now().UTC())
	_ = store.Create(context.Background(), document)

	srv := NewServer(store, "http://localhost:4566/mock-upload")
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/documents/"+document.ID+"/received", nil)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}
}

func TestAnalyzeDocuments(t *testing.T) {
	store := newStubStore()
	document := domain.NewDocument("prop_123", domain.UploadRequest{
		DocumentType: "id_front",
		FileName:     "rg.jpg",
		ContentType:  "image/jpeg",
	}, "http://localhost:4566/mock-upload", time.Now().UTC()).MarkUploaded(time.Now().UTC())
	_ = store.Create(context.Background(), document)

	srv := NewServer(store, "http://localhost:4566/mock-upload")
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/documents/analyze", nil)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}
