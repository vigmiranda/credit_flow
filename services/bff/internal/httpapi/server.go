package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
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
	Get(ctx context.Context, path, correlationID string, out any) (int, error)
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
	policies      map[string]webhookPolicy
	replay        WebhookReplayStore
	audit         WebhookAuditStore
	rateLimit     WebhookRateLimitStore
	metrics       webhookMetricsRecorder
}

type WebhookPolicyConfig struct {
	StorageMaxAge           time.Duration
	CreditMaxAge            time.Duration
	FraudMaxAge             time.Duration
	StorageRateLimit        int
	CreditRateLimit         int
	FraudRateLimit          int
	StorageRateWindow       time.Duration
	CreditRateWindow        time.Duration
	FraudRateWindow         time.Duration
	AllowedStorageProviders []string
	AllowedCreditProviders  []string
	AllowedFraudProviders   []string
}

type webhookPolicy struct {
	maxAge           time.Duration
	rateLimit        int
	rateWindow       time.Duration
	allowedProviders map[string]struct{}
}

type webhookMetricsRecorder interface {
	RecordWebhook(callbackType, provider, outcome string)
	RecordWebhookCleanup(removed int)
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
	WebhookAudit    []WebhookAuditRecord         `json:"webhook_audit,omitempty"`
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

type workflowOperationsMetrics struct {
	Service          string            `json:"service"`
	TotalRequests    int64             `json:"total_requests"`
	TotalErrors      int64             `json:"total_errors"`
	InflightRequests int64             `json:"inflight_requests"`
	AverageMS        int64             `json:"average_ms"`
	Queue            workflowQueueView `json:"queue"`
	Paths            map[string]int64  `json:"paths"`
}

type workflowQueueView struct {
	Enqueued   int64 `json:"enqueued"`
	Processed  int64 `json:"processed"`
	Retried    int64 `json:"retried"`
	DeadLetter int64 `json:"dead_letter"`
	Depth      int64 `json:"depth"`
	DLQDepth   int64 `json:"dlq_depth"`
}

type callbackAuditSummary struct {
	RecentCount        int `json:"recent_count"`
	Processed          int `json:"processed"`
	DuplicateIgnored   int `json:"duplicate_ignored"`
	Rejected           int `json:"rejected"`
	RateLimited        int `json:"rate_limited"`
	InvalidProvider    int `json:"invalid_provider"`
	ReplayReleased     int `json:"replay_released"`
	ExpiringWithinHour int `json:"expiring_within_hour"`
}

type operationsAlert struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type operationsOverviewResponse struct {
	GeneratedAt          string                     `json:"generated_at"`
	WorkflowMetrics      *workflowOperationsMetrics `json:"workflow_metrics,omitempty"`
	WorkflowDeadLetters  deadLetterListResponse     `json:"workflow_dead_letters"`
	CallbackAuditSummary callbackAuditSummary       `json:"callback_audit_summary"`
	Alerts               []operationsAlert          `json:"alerts"`
}

type deadLetterListResponse struct {
	Count int `json:"count"`
}

func NewServer(proposals proposalGateway, customers customerGateway, documents documentGateway, workflow workflowGateway, notifications notificationGateway, webhookSecret string, webhookMaxAge time.Duration) http.Handler {
	return NewServerWithDependencies(
		proposals,
		customers,
		documents,
		workflow,
		notifications,
		webhookSecret,
		webhookMaxAge,
		NewMemoryWebhookReplayStore(),
		NewMemoryWebhookAuditStore(),
		nil,
	)
}

func NewServerWithDependencies(proposals proposalGateway, customers customerGateway, documents documentGateway, workflow workflowGateway, notifications notificationGateway, webhookSecret string, webhookMaxAge time.Duration, replayStore WebhookReplayStore, auditStore WebhookAuditStore, metrics webhookMetricsRecorder) http.Handler {
	return NewServerWithPolicyConfig(
		proposals,
		customers,
		documents,
		workflow,
		notifications,
		webhookSecret,
		webhookMaxAge,
		replayStore,
		auditStore,
		NewMemoryWebhookRateLimitStore(),
		metrics,
		WebhookPolicyConfig{},
	)
}

func NewServerWithPolicyConfig(proposals proposalGateway, customers customerGateway, documents documentGateway, workflow workflowGateway, notifications notificationGateway, webhookSecret string, webhookMaxAge time.Duration, replayStore WebhookReplayStore, auditStore WebhookAuditStore, rateLimitStore WebhookRateLimitStore, metrics webhookMetricsRecorder, policyConfig WebhookPolicyConfig) http.Handler {
	if replayStore == nil {
		replayStore = NewMemoryWebhookReplayStore()
	}
	if auditStore == nil {
		auditStore = NewMemoryWebhookAuditStore()
	}
	if rateLimitStore == nil {
		rateLimitStore = NewMemoryWebhookRateLimitStore()
	}

	return &server{
		proposals:     proposals,
		customers:     customers,
		documents:     documents,
		workflow:      workflow,
		notifications: notifications,
		webhookSecret: webhookSecret,
		webhookMaxAge: webhookMaxAge,
		policies:      newWebhookPolicies(webhookMaxAge, policyConfig),
		replay:        replayStore,
		audit:         auditStore,
		rateLimit:     rateLimitStore,
		metrics:       metrics,
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
	case r.Method == http.MethodGet && r.URL.Path == "/internal/webhooks/audit":
		s.listWebhookAudit(w, r, correlationID)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/internal/operations/overview":
		s.getOperationsOverview(w, r, correlationID)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/internal/webhooks/audit/cleanup":
		s.cleanupWebhookAudit(w, r, correlationID)
		return
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/internal/webhooks/audit/") && strings.HasSuffix(r.URL.Path, "/replay-release"):
		s.releaseWebhookReplay(w, r, correlationID)
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
	if s.audit != nil {
		if records, err := s.audit.List(r.Context(), WebhookAuditFilter{
			ProposalID: proposalID,
			Limit:      100,
		}); err == nil {
			response.WebhookAudit = records
		}
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
	audit := newWebhookAuditRecord("storage", correlationID, r, rawBody, s.resolveWebhookMaxAge("storage"))

	replayStatus, release, err := s.validateWebhookRequest(r, rawBody, "storage")
	if err != nil {
		s.auditWebhook(r.Context(), audit.failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric("storage", audit.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	if replayStatus == "duplicate_ignored" {
		s.auditWebhook(r.Context(), audit.withReplayStatus(replayStatus).processed())
		s.recordWebhookMetric("storage", "", replayStatus)
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
	if err := s.validateWebhookProvider("storage", payload.Provider); err != nil {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, payload.DocumentID, payload.Provider, payload.EventType).failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric("storage", payload.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	rateLimit, err := s.applyWebhookRateLimit(r.Context(), "storage", payload.Provider)
	if err != nil {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, payload.DocumentID, payload.Provider, payload.EventType).failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric("storage", payload.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	s.writeWebhookRateLimitHeaders(w, rateLimit)
	if !rateLimit.Allowed {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, payload.DocumentID, payload.Provider, payload.EventType).failed(validationErrorCode(errWebhookRateLimited), validationErrorMessage(errWebhookRateLimited)))
		s.recordWebhookMetric("storage", payload.Provider, validationErrorCode(errWebhookRateLimited))
		s.writeWebhookRateLimitError(w, correlationID, rateLimit)
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
	s.auditWebhook(r.Context(), audit.withReplayStatus(replayStatus).withPayload(payload.ProposalID, payload.DocumentID, payload.Provider, payload.EventType).processed())
	s.recordWebhookMetric("storage", payload.Provider, replayStatus)

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
	audit := newWebhookAuditRecord(analysisType, correlationID, r, rawBody, s.resolveWebhookMaxAge(analysisType))

	replayStatus, release, err := s.validateWebhookRequest(r, rawBody, analysisType)
	if err != nil {
		s.auditWebhook(r.Context(), audit.failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric(analysisType, audit.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	if replayStatus == "duplicate_ignored" {
		s.auditWebhook(r.Context(), audit.withReplayStatus(replayStatus).processed())
		s.recordWebhookMetric(analysisType, "", replayStatus)
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
	if err := s.validateWebhookProvider(analysisType, payload.Provider); err != nil {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, "", payload.Provider, payload.EventType).failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric(analysisType, payload.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	rateLimit, err := s.applyWebhookRateLimit(r.Context(), analysisType, payload.Provider)
	if err != nil {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, "", payload.Provider, payload.EventType).failed(validationErrorCode(err), validationErrorMessage(err)))
		s.recordWebhookMetric(analysisType, payload.Provider, validationErrorCode(err))
		s.writeWebhookValidationError(w, correlationID, err)
		return
	}
	s.writeWebhookRateLimitHeaders(w, rateLimit)
	if !rateLimit.Allowed {
		s.auditWebhook(r.Context(), audit.withPayload(payload.ProposalID, "", payload.Provider, payload.EventType).failed(validationErrorCode(errWebhookRateLimited), validationErrorMessage(errWebhookRateLimited)))
		s.recordWebhookMetric(analysisType, payload.Provider, validationErrorCode(errWebhookRateLimited))
		s.writeWebhookRateLimitError(w, correlationID, rateLimit)
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
	s.auditWebhook(r.Context(), audit.withReplayStatus(replayStatus).withPayload(payload.ProposalID, "", payload.Provider, payload.EventType).processed())
	s.recordWebhookMetric(analysisType, payload.Provider, replayStatus)

	workflowResponse["status"] = "accepted"
	workflowResponse["replay_status"] = replayStatus
	writeJSON(w, http.StatusAccepted, workflowResponse)
}

func (s *server) listWebhookAudit(w http.ResponseWriter, r *http.Request, correlationID string) {
	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}

	records, err := s.audit.List(r.Context(), WebhookAuditFilter{
		EventID:      strings.TrimSpace(r.URL.Query().Get("event_id")),
		CallbackType: strings.TrimSpace(r.URL.Query().Get("callback_type")),
		ProposalID:   strings.TrimSpace(r.URL.Query().Get("proposal_id")),
		Limit:        limit,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar auditoria de webhooks", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(records),
		"records": records,
	})
}

func (s *server) getOperationsOverview(w http.ResponseWriter, r *http.Request, correlationID string) {
	response := operationsOverviewResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Alerts:      make([]operationsAlert, 0, 4),
	}

	if s.workflow != nil {
		var metrics workflowOperationsMetrics
		if _, err := s.workflow.Get(r.Context(), "/metrics", correlationID, &metrics); err == nil {
			response.WorkflowMetrics = &metrics
			response.Alerts = append(response.Alerts, buildWorkflowAlerts(metrics)...)
		}

		var deadLetters deadLetterListResponse
		if _, err := s.workflow.Get(r.Context(), "/internal/dlq", correlationID, &deadLetters); err == nil {
			response.WorkflowDeadLetters = deadLetters
			if deadLetters.Count > 0 {
				response.Alerts = append(response.Alerts, operationsAlert{
					Severity: "critical",
					Code:     "workflow_dlq_non_empty",
					Message:  "workflow possui itens na DLQ e requer inspecao operacional",
				})
			}
		}
	}

	if s.audit != nil {
		records, err := s.audit.List(r.Context(), WebhookAuditFilter{Limit: 100})
		if err == nil {
			response.CallbackAuditSummary = summarizeCallbackAudit(records)
			response.Alerts = append(response.Alerts, buildCallbackAlerts(response.CallbackAuditSummary)...)
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *server) cleanupWebhookAudit(w http.ResponseWriter, r *http.Request, correlationID string) {
	processedAt := time.Now().UTC().Format(time.RFC3339)
	if s.audit == nil {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":       "accepted",
			"removed":      0,
			"processed_at": processedAt,
		})
		return
	}

	removed, err := s.audit.CleanupExpired(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao limpar auditoria de webhooks", nil)
		return
	}
	s.recordWebhookCleanupMetric(removed)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":       "accepted",
		"removed":      removed,
		"processed_at": processedAt,
	})
}

func (s *server) releaseWebhookReplay(w http.ResponseWriter, r *http.Request, correlationID string) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/internal/webhooks/audit/")
	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) != 2 || segments[0] == "" || segments[1] != "replay-release" {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
		return
	}

	eventID := segments[0]
	record, ok, err := s.audit.Get(r.Context(), eventID)
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar auditoria de webhook", nil)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "evento de webhook nao encontrado", nil)
		return
	}

	if err := s.replay.Release(r.Context(), eventID); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao liberar replay do webhook", nil)
		return
	}
	if err := s.audit.MarkReplayReleased(r.Context(), eventID, time.Now().UTC()); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao atualizar auditoria do replay", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":        "accepted",
		"event_id":      eventID,
		"callback_type": record.CallbackType,
		"replay_status": "released",
	})
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

func (s *server) validateWebhookRequest(r *http.Request, body []byte, callbackType string) (string, func(bool), error) {
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

	maxAge := s.resolveWebhookMaxAge(callbackType)
	if maxAge > 0 {
		now := time.Now().UTC()
		if timestamp.After(now.Add(30*time.Second)) || now.Sub(timestamp) > maxAge {
			return "", nil, errStaleWebhookTimestamp
		}
	}

	expiresAt := timestamp.Add(maxAge)
	if maxAge <= 0 {
		expiresAt = time.Now().UTC().Add(5 * time.Minute)
	}
	duplicate, err := s.replay.Mark(r.Context(), eventID, expiresAt)
	if err != nil {
		return "", nil, errInvalidWebhookReplayStore
	}
	if duplicate {
		return "duplicate_ignored", func(bool) {}, nil
	}

	return "processed", func(success bool) {
		if !success {
			_ = s.replay.Release(r.Context(), eventID)
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
	errInvalidWebhookSignature   = errorString("invalid_signature")
	errInvalidWebhookTimestamp   = errorString("invalid_timestamp")
	errStaleWebhookTimestamp     = errorString("stale_timestamp")
	errMissingWebhookEventID     = errorString("missing_event_id")
	errWebhookProviderNotAllowed = errorString("invalid_provider")
	errWebhookRateLimited        = errorString("rate_limited")
	errInvalidWebhookReplayStore = errorString("replay_store_error")
	errInvalidWebhookRateStore   = errorString("rate_limit_store_error")
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
	case errWebhookProviderNotAllowed:
		writeError(w, http.StatusUnauthorized, correlationID, "invalid_provider", "provedor do webhook nao permitido para esta rota", nil)
	case errInvalidWebhookReplayStore:
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao persistir deduplicacao do webhook", nil)
	case errInvalidWebhookRateStore:
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao aplicar rate limit do webhook", nil)
	default:
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "webhook invalido", nil)
	}
}

func (s *server) writeWebhookRateLimitError(w http.ResponseWriter, correlationID string, result WebhookRateLimitResult) {
	if !result.ResetAt.IsZero() {
		retryAfter := int(time.Until(result.ResetAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}

	writeError(w, http.StatusTooManyRequests, correlationID, "rate_limited", "limite de callbacks excedido para a rota e parceiro", map[string]any{
		"limit":     result.Limit,
		"count":     result.Count,
		"remaining": result.Remaining,
		"reset_at":  result.ResetAt.UTC().Format(time.RFC3339),
	})
}

func (s *server) recordWebhookMetric(callbackType, provider, replayStatus string) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordWebhook(callbackType, provider, replayStatus)
}

func (s *server) recordWebhookCleanupMetric(removed int) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordWebhookCleanup(removed)
}

func (s *server) auditWebhook(ctx context.Context, record WebhookAuditRecord) {
	if s.audit == nil || strings.TrimSpace(record.EventID) == "" {
		return
	}
	if existing, ok, err := s.audit.Get(ctx, record.EventID); err == nil && ok {
		if record.ExpiresAt == "" {
			record.ExpiresAt = existing.ExpiresAt
		}
		if record.RetentionExpiresAt == "" {
			record.RetentionExpiresAt = existing.RetentionExpiresAt
		}
		if record.ReplayReleasedAt == "" {
			record.ReplayReleasedAt = existing.ReplayReleasedAt
		}
		if record.LastReplayAction == "" {
			record.LastReplayAction = existing.LastReplayAction
		}
	}
	_ = s.audit.Upsert(ctx, record)
}

func newWebhookPolicies(defaultMaxAge time.Duration, config WebhookPolicyConfig) map[string]webhookPolicy {
	return map[string]webhookPolicy{
		"storage": {
			maxAge:           firstNonZeroDuration(config.StorageMaxAge, defaultMaxAge),
			rateLimit:        config.StorageRateLimit,
			rateWindow:       config.StorageRateWindow,
			allowedProviders: newProviderSet(config.AllowedStorageProviders),
		},
		"credit": {
			maxAge:           firstNonZeroDuration(config.CreditMaxAge, defaultMaxAge),
			rateLimit:        config.CreditRateLimit,
			rateWindow:       config.CreditRateWindow,
			allowedProviders: newProviderSet(config.AllowedCreditProviders),
		},
		"fraud": {
			maxAge:           firstNonZeroDuration(config.FraudMaxAge, defaultMaxAge),
			rateLimit:        config.FraudRateLimit,
			rateWindow:       config.FraudRateWindow,
			allowedProviders: newProviderSet(config.AllowedFraudProviders),
		},
	}
}

func (s *server) resolveWebhookMaxAge(callbackType string) time.Duration {
	if policy, ok := s.policies[strings.TrimSpace(callbackType)]; ok && policy.maxAge > 0 {
		return policy.maxAge
	}
	return s.webhookMaxAge
}

func (s *server) validateWebhookProvider(callbackType, provider string) error {
	policy, ok := s.policies[strings.TrimSpace(callbackType)]
	if !ok || len(policy.allowedProviders) == 0 {
		return nil
	}
	if _, allowed := policy.allowedProviders[strings.TrimSpace(provider)]; allowed {
		return nil
	}
	return errWebhookProviderNotAllowed
}

func (s *server) applyWebhookRateLimit(ctx context.Context, callbackType, provider string) (WebhookRateLimitResult, error) {
	policy, ok := s.policies[strings.TrimSpace(callbackType)]
	if !ok || policy.rateLimit <= 0 || policy.rateWindow <= 0 || s.rateLimit == nil {
		return WebhookRateLimitResult{Allowed: true}, nil
	}

	result, err := s.rateLimit.Increment(ctx, callbackType, provider, policy.rateLimit, policy.rateWindow)
	if err != nil {
		return WebhookRateLimitResult{}, errInvalidWebhookRateStore
	}

	return result, nil
}

func (s *server) writeWebhookRateLimitHeaders(w http.ResponseWriter, result WebhookRateLimitResult) {
	if result.Limit > 0 {
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	}
	if result.Remaining >= 0 {
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	}
	if !result.ResetAt.IsZero() {
		w.Header().Set("X-RateLimit-Reset", result.ResetAt.UTC().Format(time.RFC3339))
	}
}

func newProviderSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}

	providers := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		providers[trimmed] = struct{}{}
	}

	if len(providers) == 0 {
		return nil
	}

	return providers
}

func firstNonZeroDuration(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func summarizeCallbackAudit(records []WebhookAuditRecord) callbackAuditSummary {
	now := time.Now().UTC()
	summary := callbackAuditSummary{
		RecentCount: len(records),
	}

	for _, record := range records {
		switch strings.TrimSpace(record.ReplayStatus) {
		case "processed":
			summary.Processed++
		case "duplicate_ignored":
			summary.DuplicateIgnored++
		}

		switch strings.TrimSpace(record.ErrorCode) {
		case "rate_limited":
			summary.RateLimited++
		case "invalid_provider":
			summary.InvalidProvider++
		}

		if strings.TrimSpace(record.ProcessingStatus) == "rejected" {
			summary.Rejected++
		}
		if strings.TrimSpace(record.LastReplayAction) == "released" {
			summary.ReplayReleased++
		}
		if expiresAt, err := time.Parse(time.RFC3339, record.RetentionExpiresAt); err == nil {
			if expiresAt.After(now) && expiresAt.Sub(now) <= time.Hour {
				summary.ExpiringWithinHour++
			}
		}
	}

	return summary
}

func buildWorkflowAlerts(metrics workflowOperationsMetrics) []operationsAlert {
	alerts := make([]operationsAlert, 0, 3)
	if metrics.TotalErrors > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "warning",
			Code:     "workflow_errors_detected",
			Message:  "workflow registrou erros e precisa de acompanhamento",
		})
	}
	if metrics.Queue.Retried > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "warning",
			Code:     "workflow_retries_detected",
			Message:  "workflow executou retries e pode indicar degradacao externa",
		})
	}
	if metrics.Queue.DLQDepth > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "critical",
			Code:     "workflow_dlq_depth_positive",
			Message:  "workflow possui profundidade positiva de DLQ",
		})
	}
	return alerts
}

func buildCallbackAlerts(summary callbackAuditSummary) []operationsAlert {
	alerts := make([]operationsAlert, 0, 4)
	if summary.RateLimited > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "warning",
			Code:     "callback_rate_limited",
			Message:  "callbacks recentes foram bloqueados por rate limit",
		})
	}
	if summary.InvalidProvider > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "warning",
			Code:     "callback_invalid_provider",
			Message:  "callbacks recentes foram rejeitados por provider fora da allowlist",
		})
	}
	if summary.ExpiringWithinHour > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "info",
			Code:     "callback_audit_expiring",
			Message:  "existem registros de auditoria proximos da expiracao operacional",
		})
	}
	if summary.ReplayReleased > 0 {
		alerts = append(alerts, operationsAlert{
			Severity: "info",
			Code:     "callback_manual_replay_release",
			Message:  "houve liberacoes manuais de replay na amostra recente",
		})
	}
	return alerts
}
