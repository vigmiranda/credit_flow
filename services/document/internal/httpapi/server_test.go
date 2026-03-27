package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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

func (s *stubStore) GetByProposalIDAndDocumentID(_ context.Context, proposalID, documentID string) (domain.Document, error) {
	for _, document := range s.items[proposalID] {
		if document.ID == documentID {
			return document, nil
		}
	}

	return domain.Document{}, domain.ErrDocumentNotFound
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

type stubStorage struct {
	uploaded bool
}

func (s *stubStorage) Upload(_ context.Context, fileKey, contentType string, body io.Reader, size int64) error {
	s.uploaded = size > 0 && fileKey != "" && contentType != ""
	return nil
}

func TestCreateUploadURL(t *testing.T) {
	srv := NewServer(newStubStore(), "http://localhost:9000/proposal-documents", &stubStorage{})
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

	if document.StorageURL == "" {
		t.Fatal("expected storage url")
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

	srv := NewServer(store, "http://localhost:9000/proposal-documents", &stubStorage{})
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

	srv := NewServer(store, "http://localhost:9000/proposal-documents", &stubStorage{})
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/documents/analyze", nil)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}

func TestUploadDocumentContent(t *testing.T) {
	store := newStubStore()
	storage := &stubStorage{}
	document := domain.NewDocument("prop_123", domain.UploadRequest{
		DocumentType: "id_front",
		FileName:     "rg.jpg",
		ContentType:  "image/jpeg",
	}, "http://localhost:9000/proposal-documents", time.Now().UTC())
	_ = store.Create(context.Background(), document)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "rg.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	srv := NewServer(store, "http://localhost:9000/proposal-documents", storage)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/documents/"+document.ID+"/content", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}
	if !storage.uploaded {
		t.Fatal("expected upload to storage")
	}
}
