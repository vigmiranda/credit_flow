package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"creditflow/services/document/internal/domain"
)

const headerCorrelationID = "X-Correlation-Id"

type DocumentStore interface {
	Create(ctx context.Context, document domain.Document) error
	ListByProposalID(ctx context.Context, proposalID string) ([]domain.Document, error)
	GetByProposalIDAndDocumentID(ctx context.Context, proposalID, documentID string) (domain.Document, error)
	MarkUploaded(ctx context.Context, proposalID, documentID string, uploadedAt time.Time) (domain.Document, error)
}

type ObjectStorage interface {
	Upload(ctx context.Context, fileKey, contentType string, body io.Reader, size int64) error
}

type server struct {
	store         DocumentStore
	uploadBaseURL string
	storage       ObjectStorage
}

type uploadRequest struct {
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	ContentType  string `json:"content_type"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(store DocumentStore, uploadBaseURL string, storage ObjectStorage) http.Handler {
	return &server{store: store, uploadBaseURL: uploadBaseURL, storage: storage}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	w.Header().Set(headerCorrelationID, correlationID)

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	case strings.HasPrefix(r.URL.Path, "/internal/proposals/"):
		s.handleProposalRoutes(w, r, correlationID)
		return
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) handleProposalRoutes(w http.ResponseWriter, r *http.Request, correlationID string) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/internal/proposals/")
	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) < 2 || segments[0] == "" || segments[1] != "documents" {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
		return
	}

	proposalID := segments[0]

	if len(segments) == 2 && r.Method == http.MethodGet {
		s.listDocuments(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 3 && segments[2] == "upload-url" && r.Method == http.MethodPost {
		s.createUploadURL(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 4 && segments[3] == "received" && r.Method == http.MethodPost {
		s.markDocumentReceived(w, r, correlationID, proposalID, segments[2])
		return
	}

	if len(segments) == 4 && segments[3] == "content" && r.Method == http.MethodPost {
		s.uploadDocumentContent(w, r, correlationID, proposalID, segments[2])
		return
	}

	if len(segments) == 3 && segments[2] == "analyze" && r.Method == http.MethodPost {
		s.analyzeDocuments(w, r, correlationID, proposalID)
		return
	}

	writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
}

func (s *server) createUploadURL(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	request := domain.UploadRequest{
		DocumentType: payload.DocumentType,
		FileName:     payload.FileName,
		ContentType:  payload.ContentType,
	}
	if details := request.Validate(); details != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "dados do documento invalidos", details)
		return
	}

	document := domain.NewDocument(proposalID, request, s.uploadBaseURL, time.Now().UTC())
	if err := s.store.Create(r.Context(), document); err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao gerar upload", nil)
		return
	}

	writeJSON(w, http.StatusOK, document)
}

func (s *server) uploadDocumentContent(w http.ResponseWriter, r *http.Request, correlationID, proposalID, documentID string) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "arquivo invalido", nil)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "arquivo obrigatorio", map[string]any{
			"required_fields": []string{"file"},
		})
		return
	}
	defer file.Close()

	document, err := s.store.GetByProposalIDAndDocumentID(r.Context(), proposalID, documentID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeError(w, http.StatusNotFound, correlationID, "not_found", "documento nao encontrado", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao carregar documento", nil)
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao ler arquivo", nil)
		return
	}

	contentType := document.ContentType
	if files := r.MultipartForm.File["file"]; len(files) > 0 {
		if headerValue := strings.TrimSpace(files[0].Header.Get("Content-Type")); headerValue != "" {
			contentType = headerValue
		}
	}
	if headerValue := strings.TrimSpace(r.Header.Get("X-Upload-Content-Type")); headerValue != "" {
		contentType = headerValue
	}

	if err := s.storage.Upload(r.Context(), document.FileKey, contentType, bytes.NewReader(content), int64(len(content))); err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao enviar arquivo para storage", nil)
		return
	}

	updated, err := s.store.MarkUploaded(r.Context(), proposalID, documentID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao atualizar documento", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, updated)
}

func (s *server) listDocuments(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	documents, err := s.store.ListByProposalID(r.Context(), proposalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao listar documentos", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"proposal_id": proposalID,
		"documents":   documents,
	})
}

func (s *server) markDocumentReceived(w http.ResponseWriter, r *http.Request, correlationID, proposalID, documentID string) {
	document, err := s.store.MarkUploaded(r.Context(), proposalID, documentID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			writeError(w, http.StatusNotFound, correlationID, "not_found", "documento nao encontrado", nil)
			return
		}

		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao atualizar documento", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, document)
}

func (s *server) analyzeDocuments(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	documents, err := s.store.ListByProposalID(r.Context(), proposalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao carregar documentos", nil)
		return
	}

	writeJSON(w, http.StatusOK, domain.AnalyzeDocuments(proposalID, documents, time.Now().UTC()))
}

func getOrCreateCorrelationID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get(headerCorrelationID)); value != "" {
		return value
	}

	return "corr_" + time.Now().UTC().Format("20060102150405")
}

func writeError(w http.ResponseWriter, status int, correlationID, code, message string, details map[string]any) {
	writeJSON(w, status, errorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
		Details:       details,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
