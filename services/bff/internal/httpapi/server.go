package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	headerCorrelationID = "X-Correlation-Id"
)

type server struct{}

type healthResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

type createProposalResponse struct {
	ProposalID string `json:"proposal_id"`
	Protocol   string `json:"protocol"`
	Status     string `json:"status"`
}

type proposalResponse struct {
	ProposalID string `json:"proposal_id"`
	Protocol   string `json:"protocol"`
	Status     string `json:"status"`
	CustomerID string `json:"customer_id,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type proposalStatusResponse struct {
	ProposalID    string `json:"proposal_id"`
	Status        string `json:"status"`
	LastUpdatedAt string `json:"last_updated_at"`
}

type customerRequest struct {
	FullName      string  `json:"full_name"`
	CPF           string  `json:"cpf"`
	BirthDate     string  `json:"birth_date"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	MonthlyIncome float64 `json:"monthly_income"`
}

type acceptedResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type documentUploadRequest struct {
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	ContentType  string `json:"content_type"`
}

type documentUploadResponse struct {
	DocumentID string `json:"document_id"`
	UploadURL  string `json:"upload_url"`
	FileKey    string `json:"file_key"`
}

func NewServer() http.Handler {
	return &server{}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	w.Header().Set(headerCorrelationID, correlationID)

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/proposals":
		s.createProposal(w, r)
		return
	case strings.HasPrefix(r.URL.Path, "/api/v1/proposals/"):
		s.handleProposalRoutes(w, r, correlationID)
		return
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) createProposal(w http.ResponseWriter, _ *http.Request) {
	proposalID := "prop_" + randomToken(8)
	protocol := "P-" + time.Now().UTC().Format("20060102150405")

	writeJSON(w, http.StatusCreated, createProposalResponse{
		ProposalID: proposalID,
		Protocol:   protocol,
		Status:     "created",
	})
}

func (s *server) handleProposalRoutes(w http.ResponseWriter, r *http.Request, correlationID string) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/proposals/")
	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "proposal_id obrigatorio", nil)
		return
	}

	proposalID := segments[0]

	if len(segments) == 1 && r.Method == http.MethodGet {
		now := time.Now().UTC().Format(time.RFC3339)
		writeJSON(w, http.StatusOK, proposalResponse{
			ProposalID: proposalID,
			Protocol:   "P-DEMO-0001",
			Status:     "documents_pending",
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		return
	}

	if len(segments) == 2 && segments[1] == "status" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, proposalStatusResponse{
			ProposalID:    proposalID,
			Status:        "documents_pending",
			LastUpdatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if len(segments) == 2 && segments[1] == "customer" && r.Method == http.MethodPost {
		s.upsertCustomer(w, r, correlationID)
		return
	}

	if len(segments) == 3 && segments[1] == "documents" && segments[2] == "upload-url" && r.Method == http.MethodPost {
		s.createDocumentUploadURL(w, r, correlationID, proposalID)
		return
	}

	writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
}

func (s *server) upsertCustomer(w http.ResponseWriter, r *http.Request, correlationID string) {
	var payload customerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if payload.FullName == "" || payload.CPF == "" || payload.Email == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "campos obrigatorios ausentes", map[string]any{
			"required_fields": []string{"full_name", "cpf", "email"},
		})
		return
	}

	writeJSON(w, http.StatusAccepted, acceptedResponse{
		Status:  "accepted",
		Message: "dados do cliente recebidos",
	})
}

func (s *server) createDocumentUploadURL(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload documentUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if payload.DocumentType == "" || payload.FileName == "" || payload.ContentType == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "campos obrigatorios ausentes", map[string]any{
			"required_fields": []string{"document_type", "file_name", "content_type"},
		})
		return
	}

	documentID := "doc_" + randomToken(8)
	fileKey := fmt.Sprintf("%s/%s/%s", proposalID, payload.DocumentType, payload.FileName)

	writeJSON(w, http.StatusOK, documentUploadResponse{
		DocumentID: documentID,
		UploadURL:  "http://localhost:4566/mock-upload/" + fileKey,
		FileKey:    fileKey,
	})
}

func getOrCreateCorrelationID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get(headerCorrelationID)); value != "" {
		return value
	}

	return "corr_" + randomToken(8)
}

func randomToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}

func writeError(w http.ResponseWriter, status int, correlationID, code, message string, details map[string]any) {
	writeJSONWithStatus(w, status, errorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
		Details:       details,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	writeJSONWithStatus(w, status, payload)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"code":"internal_error","message":"falha ao serializar resposta"}`, http.StatusInternalServerError)
	}
}
