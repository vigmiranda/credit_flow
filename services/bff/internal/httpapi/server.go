package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"creditflow/services/bff/internal/backend"
)

const (
	headerCorrelationID = "X-Correlation-Id"
	headerWebhookSig    = "X-Webhook-Signature"
	headerWebhookEvent  = "X-Webhook-Event-Id"
	headerWebhookTime   = "X-Webhook-Timestamp"
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
	UploadMultipart(ctx context.Context, path, correlationID, fieldName, fileName, contentType string, content []byte, out any) (int, error)
}

type workflowGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
}

type notificationGateway interface {
	Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	Get(ctx context.Context, path, correlationID string, out any) (int, error)
}

type server struct {
	proposals     proposalGateway
	customers     customerGateway
	documents     documentGateway
	workflow      workflowGateway
	notifications notificationGateway
	webhookSecret string
	webhookMaxAge time.Duration
	replay        *webhookReplayProtector
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
	ProposalID      string                       `json:"proposal_id"`
	Protocol        string                       `json:"protocol"`
	Status          string                       `json:"status"`
	Customer        *backend.Customer            `json:"customer,omitempty"`
	Documents       []backend.Document           `json:"documents,omitempty"`
	AnalysisResults []backend.AnalysisResult     `json:"analysis_results,omitempty"`
	StatusHistory   []backend.StatusHistoryEntry `json:"status_history,omitempty"`
	Notifications   []backend.Notification       `json:"notifications,omitempty"`
	CreatedAt       string                       `json:"created_at"`
	UpdatedAt       string                       `json:"updated_at"`
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
	StorageURL string `json:"storage_url,omitempty"`
	Status     string `json:"status,omitempty"`
}

type storageWebhookRequest struct {
	ProposalID string `json:"proposal_id"`
	DocumentID string `json:"document_id"`
	Provider   string `json:"provider,omitempty"`
	EventType  string `json:"event_type,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type analysisWebhookRequest struct {
	ProposalID string `json:"proposal_id"`
	Provider   string `json:"provider,omitempty"`
	EventType  string `json:"event_type,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
	Result     string `json:"result"`
	Score      int    `json:"score,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func NewServer(proposals proposalGateway, customers customerGateway, documents documentGateway, workflow workflowGateway, notifications notificationGateway, webhookSecret string, webhookMaxAge time.Duration) http.Handler {
	return &server{
		proposals:     proposals,
		customers:     customers,
		documents:     documents,
		workflow:      workflow,
		notifications: notifications,
		webhookSecret: webhookSecret,
		webhookMaxAge: webhookMaxAge,
		replay:        newWebhookReplayProtector(),
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
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhooks/storage/document-uploaded":
		s.handleStorageDocumentUploaded(w, r, correlationID)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhooks/partners/credit-analysis":
		s.handleAnalysisWebhook(w, r, correlationID, "credit")
		return
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/webhooks/partners/fraud-analysis":
		s.handleAnalysisWebhook(w, r, correlationID, "fraud")
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

	if len(segments) == 4 && segments[1] == "documents" && segments[3] == "content" && r.Method == http.MethodPost {
		s.uploadDocumentContent(w, r, correlationID, proposalID, segments[2])
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
		response.Customer = maskCustomer(&customer)
	}

	var documents backend.DocumentList
	if _, err := s.documents.Get(r.Context(), "/internal/proposals/"+proposalID+"/documents", correlationID, &documents); err == nil {
		response.Documents = documents.Documents
	}

	var analysisResults backend.AnalysisResultList
	if _, err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID+"/analysis-results", correlationID, &analysisResults); err == nil {
		response.AnalysisResults = analysisResults.AnalysisResults
	}

	var statusHistory backend.StatusHistoryList
	if _, err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID+"/status-history", correlationID, &statusHistory); err == nil {
		response.StatusHistory = statusHistory.StatusHistory
	}

	var notifications backend.NotificationList
	if _, err := s.notifications.Get(r.Context(), "/internal/proposals/"+proposalID+"/notifications", correlationID, &notifications); err == nil {
		response.Notifications = maskNotifications(notifications.Notifications)
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
	s.sendNotification(r.Context(), proposalID, payload.Email, "documents_pending", "Recebemos seus dados. Agora envie os documentos da proposta.", correlationID)

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
		UploadURL:  "/api/v1/proposals/" + proposalID + "/documents/" + document.DocumentID + "/content",
		FileKey:    document.FileKey,
		StorageURL: document.StorageURL,
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
	s.completeDocumentFlow(r.Context(), proposalID, correlationID, &proposal)

	writeJSON(w, http.StatusAccepted, document)
}

func (s *server) uploadDocumentContent(w http.ResponseWriter, r *http.Request, correlationID, proposalID, documentID string) {
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "arquivo invalido", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "arquivo obrigatorio", map[string]any{
			"required_fields": []string{"file"},
		})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao ler arquivo", nil)
		return
	}

	contentType := header.Header.Get("Content-Type")
	var document backend.Document
	if _, err := s.documents.UploadMultipart(
		r.Context(),
		"/internal/proposals/"+proposalID+"/documents/"+documentID+"/content",
		correlationID,
		"file",
		header.Filename,
		contentType,
		content,
		&document,
	); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao enviar documento", nil)
		return
	}

	var proposal backend.Proposal
	s.completeDocumentFlow(r.Context(), proposalID, correlationID, &proposal)

	writeJSON(w, http.StatusAccepted, document)
}

func (s *server) handleStorageDocumentUploaded(w http.ResponseWriter, r *http.Request, correlationID string) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	replayStatus, release, err := s.validateWebhookRequest(r, rawBody)
	if err != nil {
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	if replayStatus == "duplicate_ignored" {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":        "accepted",
			"replay_status": replayStatus,
		})
		return
	}
	processed := false
	defer func() {
		release(processed)
	}()

	var payload storageWebhookRequest
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if strings.TrimSpace(payload.ProposalID) == "" || strings.TrimSpace(payload.DocumentID) == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "campos obrigatorios ausentes", map[string]any{
			"required_fields": []string{"proposal_id", "document_id"},
		})
		return
	}

	var document backend.Document
	if _, err := s.documents.Post(
		r.Context(),
		"/internal/proposals/"+payload.ProposalID+"/documents/"+payload.DocumentID+"/received",
		correlationID,
		nil,
		&document,
	); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao confirmar documento por webhook", nil)
		return
	}

	var proposal backend.Proposal
	s.completeDocumentFlow(r.Context(), payload.ProposalID, correlationID, &proposal)
	processed = true

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":        "accepted",
		"replay_status": replayStatus,
		"proposal_id":   payload.ProposalID,
		"document_id":   payload.DocumentID,
		"provider":      payload.Provider,
		"event_type":    payload.EventType,
	})
}

func (s *server) handleAnalysisWebhook(w http.ResponseWriter, r *http.Request, correlationID, analysisType string) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	replayStatus, release, err := s.validateWebhookRequest(r, rawBody)
	if err != nil {
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	if replayStatus == "duplicate_ignored" {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":        "accepted",
			"analysis_type": analysisType,
			"replay_status": replayStatus,
		})
		return
	}
	processed := false
	defer func() {
		release(processed)
	}()

	var payload analysisWebhookRequest
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}
	if strings.TrimSpace(payload.ProposalID) == "" || strings.TrimSpace(payload.Result) == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "campos obrigatorios ausentes", map[string]any{
			"required_fields": []string{"proposal_id", "result"},
		})
		return
	}

	var workflowResponse map[string]any
	if _, err := s.workflow.Post(
		r.Context(),
		"/internal/proposals/"+payload.ProposalID+"/external-analyses/"+analysisType,
		correlationID,
		map[string]any{
			"provider":    payload.Provider,
			"event_type":  payload.EventType,
			"occurred_at": payload.OccurredAt,
			"result":      payload.Result,
			"score":       payload.Score,
			"reason":      payload.Reason,
		},
		&workflowResponse,
	); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao aplicar callback externo de analise", nil)
		return
	}
	processed = true

	workflowResponse["status"] = "accepted"
	workflowResponse["replay_status"] = replayStatus
	writeJSON(w, http.StatusAccepted, workflowResponse)
}

func (s *server) triggerWorkflow(proposalID, correlationID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, _ = s.workflow.Post(ctx, "/internal/proposals/"+proposalID+"/run-analyses", correlationID, nil, nil)
}

func (s *server) completeDocumentFlow(ctx context.Context, proposalID, correlationID string, proposal *backend.Proposal) {
	if proposal != nil {
		_, _ = s.proposals.Patch(ctx, "/internal/proposals/"+proposalID+"/status", correlationID, map[string]string{
			"status": "documents_received",
		}, proposal)
	} else {
		_, _ = s.proposals.Patch(ctx, "/internal/proposals/"+proposalID+"/status", correlationID, map[string]string{
			"status": "documents_received",
		}, nil)
	}
	if email, ok := s.fetchCustomerEmail(ctx, proposalID, correlationID); ok {
		s.sendNotification(ctx, proposalID, email, "documents_received", "Recebemos seus documentos e vamos iniciar as analises.", correlationID)
	}

	go s.triggerWorkflow(proposalID, correlationID)
}

func (s *server) fetchCustomerEmail(ctx context.Context, proposalID, correlationID string) (string, bool) {
	var customer backend.Customer
	if _, err := s.customers.Get(ctx, "/internal/proposals/"+proposalID+"/customer", correlationID, &customer); err != nil {
		return "", false
	}

	if strings.TrimSpace(customer.Email) == "" {
		return "", false
	}

	return customer.Email, true
}

func (s *server) sendNotification(ctx context.Context, proposalID, recipient, triggerStatus, message, correlationID string) {
	if strings.TrimSpace(recipient) == "" {
		return
	}

	_, _ = s.notifications.Post(ctx, "/internal/proposals/"+proposalID+"/notifications", correlationID, map[string]any{
		"channel":        "email",
		"template":       "proposal_status_changed",
		"recipient":      recipient,
		"message":        message,
		"trigger_status": triggerStatus,
	}, nil)
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

func verifyWebhookSignature(secret string, body []byte, provided string) bool {
	if strings.TrimSpace(secret) == "" {
		return true
	}

	signature := strings.TrimSpace(provided)
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	decoded, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(decoded, expected)
}

func (s *server) validateWebhookRequest(r *http.Request, body []byte) (string, func(bool), error) {
	if !verifyWebhookSignature(s.webhookSecret, body, r.Header.Get(headerWebhookSig)) {
		return "", nil, errInvalidWebhookSignature
	}

	eventID := strings.TrimSpace(r.Header.Get(headerWebhookEvent))
	if eventID == "" {
		return "", nil, errMissingWebhookEventID
	}

	timestamp, err := parseWebhookTimestamp(r.Header.Get(headerWebhookTime))
	if err != nil {
		return "", nil, errInvalidWebhookTimestamp
	}

	if s.webhookMaxAge > 0 {
		now := time.Now().UTC()
		if timestamp.After(now.Add(30*time.Second)) || now.Sub(timestamp) > s.webhookMaxAge {
			return "", nil, errStaleWebhookTimestamp
		}
	}

	expiresAt := timestamp.Add(s.webhookMaxAge)
	if s.webhookMaxAge <= 0 {
		expiresAt = time.Now().UTC().Add(5 * time.Minute)
	}
	if s.replay.Mark(eventID, expiresAt) {
		return "duplicate_ignored", func(bool) {}, nil
	}

	return "processed", func(success bool) {
		if !success {
			s.replay.Release(eventID)
		}
	}, nil
}

func parseWebhookTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errInvalidWebhookTimestamp
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC(), nil
	}

	return time.Time{}, errInvalidWebhookTimestamp
}

var (
	errInvalidWebhookSignature = errorString("invalid_signature")
	errInvalidWebhookTimestamp = errorString("invalid_timestamp")
	errStaleWebhookTimestamp   = errorString("stale_timestamp")
	errMissingWebhookEventID   = errorString("missing_event_id")
)

type errorString string

func (e errorString) Error() string {
	return string(e)
}

func (s *server) writeWebhookValidationError(w http.ResponseWriter, correlationID string, err error) {
	switch err {
	case errInvalidWebhookSignature:
		writeError(w, http.StatusUnauthorized, correlationID, "invalid_signature", "assinatura do webhook invalida", nil)
	case errMissingWebhookEventID:
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "header X-Webhook-Event-Id obrigatorio", nil)
	case errInvalidWebhookTimestamp:
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "header X-Webhook-Timestamp invalido", nil)
	case errStaleWebhookTimestamp:
		writeError(w, http.StatusUnauthorized, correlationID, "stale_webhook", "timestamp do webhook fora da janela aceita", nil)
	default:
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "webhook invalido", nil)
	}
}
