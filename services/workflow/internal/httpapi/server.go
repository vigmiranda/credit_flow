package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"creditflow/services/workflow/internal/backend"
	"creditflow/services/workflow/internal/queue"
)

const headerCorrelationID = "X-Correlation-Id"

type gateway interface {
	Get(ctx context.Context, path, correlationID string, out any) error
	Post(ctx context.Context, path, correlationID string, payload any, out any) error
	Patch(ctx context.Context, path, correlationID string, payload any, out any) error
}

type server struct {
	proposals     gateway
	customers     gateway
	documents     gateway
	credit        gateway
	fraud         gateway
	notifications gateway
	queue         queue.Queue
	delay         time.Duration
	maxRetries    int
	metrics       queueMetrics
}

type queueMetrics interface {
	RecordQueueEnqueued()
	RecordQueueProcessed()
	RecordQueueRetried()
	RecordQueueDeadLettered()
	SetQueueDepth(value int64)
	SetDeadLetterDepth(value int64)
}

type workflowResponse struct {
	ProposalID  string                   `json:"proposal_id"`
	FinalStatus string                   `json:"final_status"`
	QueueStatus string                   `json:"queue_status,omitempty"`
	Attempt     int                      `json:"attempt,omitempty"`
	Results     []backend.AnalysisResult `json:"results"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(proposals gateway, customers gateway, documents gateway, credit gateway, fraud gateway, notifications gateway, workflowQueue queue.Queue, delay time.Duration, maxRetries int, metrics queueMetrics) *server {
	if maxRetries < 0 {
		maxRetries = 0
	}

	return &server{
		proposals:     proposals,
		customers:     customers,
		documents:     documents,
		credit:        credit,
		fraud:         fraud,
		notifications: notifications,
		queue:         workflowQueue,
		delay:         delay,
		maxRetries:    maxRetries,
		metrics:       metrics,
	}
}

func (s *server) StartWorkers(ctx context.Context, workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}

	for index := 0; index < workerCount; index++ {
		go s.workerLoop(ctx)
	}
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
	if len(segments) != 2 || segments[0] == "" || segments[1] != "run-analyses" || r.Method != http.MethodPost {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
		return
	}

	job := queue.Job{
		ProposalID:    segments[0],
		CorrelationID: correlationID,
		Attempt:       0,
		EnqueuedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.queue.Enqueue(r.Context(), job); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "queue_error", "falha ao enfileirar workflow", nil)
		return
	}
	s.recordQueueEnqueued(r.Context())

	writeJSON(w, http.StatusAccepted, workflowResponse{
		ProposalID:  segments[0],
		FinalStatus: "queued",
		QueueStatus: "queued",
		Attempt:     0,
		Results:     []backend.AnalysisResult{},
	})
}

func (s *server) workerLoop(ctx context.Context) {
	for {
		job, err := s.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			time.Sleep(500 * time.Millisecond)
			continue
		}

		runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		err = s.processJob(runCtx, job)
		cancel()
		if err == nil {
			s.recordQueueProcessed(ctx)
			continue
		}

		if job.Attempt >= s.maxRetries {
			s.handleWorkflowFailure(ctx, job, err)
			continue
		}

		job.Attempt++
		job.EnqueuedAt = time.Now().UTC().Format(time.RFC3339)
		job.LastError = err.Error()
		_ = s.queue.Enqueue(ctx, job)
		s.recordQueueRetried(ctx)
	}
}

func (s *server) processJob(ctx context.Context, job queue.Job) error {
	_, err := s.runWorkflow(ctx, job.ProposalID, nonEmptyCorrelationID(job.CorrelationID))
	return err
}

func (s *server) runWorkflow(ctx context.Context, proposalID, correlationID string) (workflowResponse, error) {
	var response workflowResponse
	response.ProposalID = proposalID

	var customer backend.Customer
	if err := s.customers.Get(ctx, "/internal/proposals/"+proposalID+"/customer", correlationID, &customer); err != nil {
		return response, err
	}

	if err := s.updateProposalStatus(ctx, proposalID, "document_analysis_in_progress", correlationID); err != nil {
		return response, err
	}
	s.sendNotification(ctx, proposalID, customer.Email, "document_analysis_in_progress", "Sua proposta entrou em analise documental.", correlationID)
	time.Sleep(s.delay)

	var documentAnalysis backend.AnalysisResult
	if err := s.documents.Post(ctx, "/internal/proposals/"+proposalID+"/documents/analyze", correlationID, nil, &documentAnalysis); err != nil {
		return response, err
	}
	if err := s.storeAnalysisResult(ctx, proposalID, correlationID, documentAnalysis); err != nil {
		return response, err
	}
	response.Results = append(response.Results, documentAnalysis)
	if final, done := finalizeFromResult(documentAnalysis); done {
		response.FinalStatus = final
		_ = s.updateProposalStatus(ctx, proposalID, final, correlationID)
		s.sendNotification(ctx, proposalID, customer.Email, final, statusMessage(final), correlationID)
		return response, nil
	}

	if err := s.updateProposalStatus(ctx, proposalID, "credit_analysis_in_progress", correlationID); err != nil {
		return response, err
	}
	s.sendNotification(ctx, proposalID, customer.Email, "credit_analysis_in_progress", "Sua proposta entrou em analise de credito.", correlationID)
	time.Sleep(s.delay)

	var creditAnalysis backend.AnalysisResult
	if err := s.credit.Post(ctx, "/internal/proposals/"+proposalID+"/credit-analysis", correlationID, map[string]any{
		"customer": customer,
	}, &creditAnalysis); err != nil {
		return response, err
	}
	if err := s.storeAnalysisResult(ctx, proposalID, correlationID, creditAnalysis); err != nil {
		return response, err
	}
	response.Results = append(response.Results, creditAnalysis)
	if final, done := finalizeFromResult(creditAnalysis); done {
		response.FinalStatus = final
		_ = s.updateProposalStatus(ctx, proposalID, final, correlationID)
		s.sendNotification(ctx, proposalID, customer.Email, final, statusMessage(final), correlationID)
		return response, nil
	}

	if err := s.updateProposalStatus(ctx, proposalID, "fraud_analysis_in_progress", correlationID); err != nil {
		return response, err
	}
	s.sendNotification(ctx, proposalID, customer.Email, "fraud_analysis_in_progress", "Sua proposta entrou em analise antifraude.", correlationID)
	time.Sleep(s.delay)

	var fraudAnalysis backend.AnalysisResult
	if err := s.fraud.Post(ctx, "/internal/proposals/"+proposalID+"/fraud-analysis", correlationID, map[string]any{
		"customer": customer,
	}, &fraudAnalysis); err != nil {
		return response, err
	}
	if err := s.storeAnalysisResult(ctx, proposalID, correlationID, fraudAnalysis); err != nil {
		return response, err
	}
	response.Results = append(response.Results, fraudAnalysis)

	finalStatus := "approved"
	if fraudAnalysis.Result == "manual_review" {
		finalStatus = "manual_review"
	}
	if fraudAnalysis.Result == "rejected" {
		finalStatus = "rejected"
	}

	if err := s.updateProposalStatus(ctx, proposalID, finalStatus, correlationID); err != nil {
		return response, err
	}
	s.sendNotification(ctx, proposalID, customer.Email, finalStatus, statusMessage(finalStatus), correlationID)

	response.FinalStatus = finalStatus
	return response, nil
}

func (s *server) handleWorkflowFailure(ctx context.Context, job queue.Job, workflowErr error) {
	correlationID := nonEmptyCorrelationID(job.CorrelationID)
	job.LastError = workflowErr.Error()
	_ = s.queue.DeadLetter(ctx, job)
	s.recordQueueDeadLettered(ctx)
	_ = s.updateProposalStatus(ctx, job.ProposalID, "manual_review", correlationID)
	if email, ok := s.fetchCustomerEmail(ctx, job.ProposalID, correlationID); ok {
		s.sendNotification(
			ctx,
			job.ProposalID,
			email,
			"manual_review",
			"Sua proposta foi encaminhada para revisao manual apos uma falha tecnica no processamento.",
			correlationID,
		)
	}

	_ = workflowErr
}

func (s *server) recordQueueEnqueued(ctx context.Context) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordQueueEnqueued()
	s.refreshQueueDepth(ctx)
}

func (s *server) recordQueueProcessed(ctx context.Context) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordQueueProcessed()
	s.refreshQueueDepth(ctx)
}

func (s *server) recordQueueRetried(ctx context.Context) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordQueueRetried()
	s.refreshQueueDepth(ctx)
}

func (s *server) recordQueueDeadLettered(ctx context.Context) {
	if s.metrics == nil {
		return
	}

	s.metrics.RecordQueueDeadLettered()
	s.refreshQueueDepth(ctx)
}

func (s *server) refreshQueueDepth(ctx context.Context) {
	if s.metrics == nil {
		return
	}

	if depth, err := s.queue.Length(ctx); err == nil {
		s.metrics.SetQueueDepth(depth)
	}
	if depth, err := s.queue.DeadLetterLength(ctx); err == nil {
		s.metrics.SetDeadLetterDepth(depth)
	}
}

func (s *server) updateProposalStatus(ctx context.Context, proposalID, status, correlationID string) error {
	return s.proposals.Patch(ctx, "/internal/proposals/"+proposalID+"/status", correlationID, map[string]string{
		"status": status,
	}, nil)
}

func (s *server) storeAnalysisResult(ctx context.Context, proposalID, correlationID string, result backend.AnalysisResult) error {
	return s.proposals.Post(ctx, "/internal/proposals/"+proposalID+"/analysis-results", correlationID, map[string]any{
		"analysis_type": result.AnalysisType,
		"result":        result.Result,
		"provider":      result.Provider,
		"score":         result.Score,
		"reason":        result.Reason,
	}, nil)
}

func finalizeFromResult(result backend.AnalysisResult) (string, bool) {
	switch result.Result {
	case "awaiting_additional_documents":
		return "awaiting_additional_documents", true
	case "manual_review":
		return "manual_review", true
	case "rejected":
		return "rejected", true
	default:
		return "", false
	}
}

func (s *server) fetchCustomerEmail(ctx context.Context, proposalID, correlationID string) (string, bool) {
	var customer backend.Customer
	if err := s.customers.Get(ctx, "/internal/proposals/"+proposalID+"/customer", correlationID, &customer); err != nil {
		return "", false
	}

	if strings.TrimSpace(customer.Email) == "" {
		return "", false
	}

	return customer.Email, true
}

func (s *server) sendNotification(ctx context.Context, proposalID, recipient, status, message, correlationID string) {
	if strings.TrimSpace(recipient) == "" {
		return
	}

	_ = s.notifications.Post(ctx, "/internal/proposals/"+proposalID+"/notifications", correlationID, map[string]any{
		"channel":        "email",
		"template":       "proposal_status_changed",
		"recipient":      recipient,
		"message":        message,
		"trigger_status": status,
	}, nil)
}

func statusMessage(status string) string {
	switch status {
	case "approved":
		return "Sua proposta foi aprovada."
	case "rejected":
		return "Sua proposta foi reprovada."
	case "manual_review":
		return "Sua proposta foi direcionada para revisao manual."
	case "awaiting_additional_documents":
		return "Sua proposta precisa de documentos complementares."
	default:
		return "Sua proposta mudou de status."
	}
}

func nonEmptyCorrelationID(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}

	return "corr_" + time.Now().UTC().Format("20060102150405")
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
