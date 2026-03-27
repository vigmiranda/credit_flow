package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"creditflow/services/bff/internal/backend"
)

const (
	headerCorrelationID = "X-Correlation-Id"
)

type proposalGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	Get(ctx context.Context, path, correlationID string, out any) (int, error)
	Patch(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
}

type customerGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	Get(ctx context.Context, path, correlationID string, out any) (int, error)
}

type documentGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	Get(ctx context.Context, path, correlationID string, out any) (int, error)
}

type workflowGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
}

type server struct {
	proposals proposalGateway
	customers customerGateway
	documents documentGateway
	workflow  workflowGateway
}

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
	ProposalID      string                   `json:"proposal_id"`
	Protocol        string                   `json:"protocol"`
	Status          string                   `json:"status"`
	Customer        *backend.Customer        `json:"customer,omitempty"`
	Documents       []backend.Document       `json:"documents,omitempty"`
	AnalysisResults []backend.AnalysisResult `json:"analysis_results,omitempty"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`
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
	Status     string `json:"status,omitempty"`
}

func NewServer(proposals proposalGateway, customers customerGateway, documents documentGateway, workflow workflowGateway) http.Handler {
	return &server{
		proposals: proposals,
		customers: customers,
		documents: documents,
		workflow:  workflow,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	w.Header().Set(headerCorrelationID, correlationID)
	applyCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/proposals":
		s.createProposal(w, r, correlationID)
		return
	case strings.HasPrefix(r.URL.Path, "/api/v1/proposals/"):
		s.handleProposalRoutes(w, r, correlationID)
		return
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) createProposal(w http.ResponseWriter, r *http.Request, correlationID string) {
	var proposal backend.Proposal
	if _, err := s.proposals.Post(r.Context(), "/internal/proposals", correlationID, nil, &proposal); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao criar proposta", nil)
		return
	}

	var updated backend.Proposal
	if _, err := s.proposals.Patch(r.Context(), "/internal/proposals/"+proposal.ProposalID+"/status", correlationID, map[string]string{
		"status": "customer_data_pending",
	}, &updated); err == nil {
		proposal = updated
	}

	writeJSON(w, http.StatusCreated, createProposalResponse{
		ProposalID: proposal.ProposalID,
		Protocol:   proposal.Protocol,
		Status:     proposal.Status,
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
		s.getProposal(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 2 && segments[1] == "status" && r.Method == http.MethodGet {
		s.getProposalStatus(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 2 && segments[1] == "customer" && r.Method == http.MethodPost {
		s.upsertCustomer(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 3 && segments[1] == "documents" && segments[2] == "upload-url" && r.Method == http.MethodPost {
		s.createDocumentUploadURL(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 2 && segments[1] == "documents" && r.Method == http.MethodGet {
		s.listDocuments(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 4 && segments[1] == "documents" && segments[3] == "received" && r.Method == http.MethodPost {
		s.markDocumentReceived(w, r, correlationID, proposalID, segments[2])
		return
	}

	writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
}

func (s *server) getProposal(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var proposal backend.Proposal
	if _, err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID, correlationID, &proposal); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar proposta", nil)
		return
	}

	response := proposalResponse{
		ProposalID: proposal.ProposalID,
		Protocol:   proposal.Protocol,
		Status:     proposal.Status,
		CreatedAt:  proposal.CreatedAt,
		UpdatedAt:  proposal.UpdatedAt,
	}

	var customer backend.Customer
	if _, err := s.customers.Get(r.Context(), "/internal/proposals/"+proposalID+"/customer", correlationID, &customer); err == nil {
		response.Customer = &customer
	}

	var documents backend.DocumentList
	if _, err := s.documents.Get(r.Context(), "/internal/proposals/"+proposalID+"/documents", correlationID, &documents); err == nil {
		response.Documents = documents.Documents
	}

	var analysisResults backend.AnalysisResultList
	if _, err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID+"/analysis-results", correlationID, &analysisResults); err == nil {
		response.AnalysisResults = analysisResults.AnalysisResults
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *server) getProposalStatus(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var proposal backend.Proposal
	if _, err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID, correlationID, &proposal); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar status", nil)
		return
	}

	writeJSON(w, http.StatusOK, proposalStatusResponse{
		ProposalID:    proposal.ProposalID,
		Status:        proposal.Status,
		LastUpdatedAt: proposal.UpdatedAt,
	})
}

func (s *server) upsertCustomer(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
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

	var customer backend.Customer
	if _, err := s.customers.Post(r.Context(), "/internal/proposals/"+proposalID+"/customer", correlationID, payload, &customer); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao salvar cliente", nil)
		return
	}

	var proposal backend.Proposal
	_, _ = s.proposals.Patch(r.Context(), "/internal/proposals/"+proposalID+"/status", correlationID, map[string]string{
		"status": "documents_pending",
	}, &proposal)

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

	var document backend.Document
	if _, err := s.documents.Post(r.Context(), "/internal/proposals/"+proposalID+"/documents/upload-url", correlationID, payload, &document); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao gerar upload", nil)
		return
	}

	writeJSON(w, http.StatusOK, documentUploadResponse{
		DocumentID: document.DocumentID,
		UploadURL:  document.UploadURL,
		FileKey:    document.FileKey,
		Status:     document.Status,
	})
}

func (s *server) listDocuments(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var documents backend.DocumentList
	if _, err := s.documents.Get(r.Context(), "/internal/proposals/"+proposalID+"/documents", correlationID, &documents); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao listar documentos", nil)
		return
	}

	writeJSON(w, http.StatusOK, documents)
}

func (s *server) markDocumentReceived(w http.ResponseWriter, r *http.Request, correlationID, proposalID, documentID string) {
	var document backend.Document
	if _, err := s.documents.Post(r.Context(), "/internal/proposals/"+proposalID+"/documents/"+documentID+"/received", correlationID, nil, &document); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao confirmar envio do documento", nil)
		return
	}

	var proposal backend.Proposal
	_, _ = s.proposals.Patch(r.Context(), "/internal/proposals/"+proposalID+"/status", correlationID, map[string]string{
		"status": "documents_received",
	}, &proposal)

	go s.triggerWorkflow(proposalID, correlationID)

	writeJSON(w, http.StatusAccepted, document)
}

func (s *server) triggerWorkflow(proposalID, correlationID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, _ = s.workflow.Post(ctx, "/internal/proposals/"+proposalID+"/run-analyses", correlationID, nil, nil)
}

func getOrCreateCorrelationID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get(headerCorrelationID)); value != "" {
		return value
	}

	return "corr_" + time.Now().UTC().Format("20060102150405")
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

func applyCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Correlation-Id, Idempotency-Key")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
}
