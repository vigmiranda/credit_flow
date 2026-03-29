package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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
	proposals      gateway
	customers      gateway
	documents      gateway
	credit         gateway
	fraud          gateway
	notifications  gateway
	queue          queue.Queue
	delay          time.Duration
	maxRetries     int
	metrics        queueMetrics
	externalCredit bool
	externalFraud  bool
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

type deadLetterResponse struct {
	Count int         `json:"count"`
	Jobs  []queue.Job `json:"jobs"`
}

type reprocessDLQRequest struct {
	ProposalID string `json:"proposal_id,omitempty"`
}

type reprocessDLQResponse struct {
	Status           string `json:"status"`
	ProposalID       string `json:"proposal_id,omitempty"`
	ReprocessedCount int    `json:"reprocessed_count"`
}

type externalAnalysisRequest struct {
	Provider   string `json:"provider"`
	EventType  string `json:"event_type,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
	Result     string `json:"result"`
	Score      int    `json:"score,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type externalAnalysisResponse struct {
	Status         string `json:"status"`
	ProposalID     string `json:"proposal_id"`
	AnalysisType   string `json:"analysis_type"`
	Result         string `json:"result"`
	ProposalStatus string `json:"proposal_status"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(proposals gateway, customers gateway, documents gateway, credit gateway, fraud gateway, notifications gateway, workflowQueue queue.Queue, delay time.Duration, maxRetries int, metrics queueMetrics, externalCredit bool, externalFraud bool) *server {
	if maxRetries < 0 {
		maxRetries = 0
	}

	return &server{
		proposals:      proposals,
		customers:      customers,
		documents:      documents,
		credit:         credit,
		fraud:          fraud,
		notifications:  notifications,
		queue:          workflowQueue,
		delay:          delay,
		maxRetries:     maxRetries,
		metrics:        metrics,
		externalCredit: externalCredit,
		externalFraud:  externalFraud,
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
	case r.Method == http.MethodGet && r.URL.Path == "/internal/dlq":
		s.listDeadLetters(w, r, correlationID)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/internal/dlq/reprocess":
		s.reprocessDeadLetters(w, r, correlationID)
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
	if len(segments) == 2 && segments[0] != "" && segments[1] == "run-analyses" && r.Method == http.MethodPost {
		s.enqueueWorkflow(w, r, correlationID, segments[0])
		return
	}

	if len(segments) == 3 && segments[0] != "" && segments[1] == "external-analyses" && r.Method == http.MethodPost {
		s.applyExternalAnalysis(w, r, correlationID, segments[0], segments[2])
		return
	}

	writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
}

func (s *server) enqueueWorkflow(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	job := queue.Job{
		ProposalID:    proposalID,
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
		ProposalID:  proposalID,
		FinalStatus: "queued",
		QueueStatus: "queued",
		Attempt:     0,
		Results:     []backend.AnalysisResult{},
	})
}

func (s *server) listDeadLetters(w http.ResponseWriter, r *http.Request, correlationID string) {
	jobs, err := s.queue.ListDeadLetters(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "queue_error", "falha ao consultar DLQ", nil)
		return
	}

	writeJSON(w, http.StatusOK, deadLetterResponse{
		Count: len(jobs),
		Jobs:  jobs,
	})
}

func (s *server) reprocessDeadLetters(w http.ResponseWriter, r *http.Request, correlationID string) {
	var payload reprocessDLQRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
			return
		}
	}

	count, err := s.queue.RequeueDeadLetters(r.Context(), strings.TrimSpace(payload.ProposalID))
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "queue_error", "falha ao reprocessar DLQ", nil)
		return
	}
	s.refreshQueueDepth(r.Context())

	writeJSON(w, http.StatusAccepted, reprocessDLQResponse{
		Status:           "accepted",
		ProposalID:       strings.TrimSpace(payload.ProposalID),
		ReprocessedCount: count,
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
	if s.externalCredit {
		response.FinalStatus = "credit_analysis_in_progress"
		return response, nil
	}
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

	if err := s.moveToFraudStage(ctx, proposalID, customer.Email, correlationID); err != nil {
		return response, err
	}
	if s.externalFraud {
		response.FinalStatus = "fraud_analysis_in_progress"
		return response, nil
	}
	time.Sleep(s.delay)

	fraudAnalysis, err := s.executeFraudAnalysis(ctx, proposalID, customer, correlationID)
	if err != nil {
		return response, err
	}
	response.Results = append(response.Results, fraudAnalysis)

	finalStatus := finalStatusFromFraudResult(fraudAnalysis.Result)
	if err := s.updateProposalStatus(ctx, proposalID, finalStatus, correlationID); err != nil {
		return response, err
	}
	s.sendNotification(ctx, proposalID, customer.Email, finalStatus, statusMessage(finalStatus), correlationID)

	response.FinalStatus = finalStatus
	return response, nil
}

func (s *server) applyExternalAnalysis(w http.ResponseWriter, r *http.Request, correlationID, proposalID, analysisType string) {
	if analysisType != "credit" && analysisType != "fraud" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "analysis_type invalido", map[string]any{
			"analysis_type": analysisType,
		})
		return
	}

	var payload externalAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}
	if !isValidExternalAnalysisResult(payload.Result) {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "result invalido", map[string]any{
			"result": payload.Result,
		})
		return
	}

	var proposal backend.Proposal
	if err := s.proposals.Get(r.Context(), "/internal/proposals/"+proposalID, correlationID, &proposal); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar proposta para callback externo", nil)
		return
	}

	var customer backend.Customer
	if err := s.customers.Get(r.Context(), "/internal/proposals/"+proposalID+"/customer", correlationID, &customer); err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", "falha ao consultar cliente para callback externo", nil)
		return
	}

	var proposalStatus string
	var err error
	switch analysisType {
	case "credit":
		proposalStatus, err = s.applyExternalCreditResult(r.Context(), proposal, customer, correlationID, payload)
	case "fraud":
		proposalStatus, err = s.applyExternalFraudResult(r.Context(), proposal, customer, correlationID, payload)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_state", err.Error(), map[string]any{
			"proposal_status": proposal.Status,
			"analysis_type":   analysisType,
		})
		return
	}

	writeJSON(w, http.StatusAccepted, externalAnalysisResponse{
		Status:         "accepted",
		ProposalID:     proposalID,
		AnalysisType:   analysisType,
		Result:         payload.Result,
		ProposalStatus: proposalStatus,
	})
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

func (s *server) applyExternalCreditResult(ctx context.Context, proposal backend.Proposal, customer backend.Customer, correlationID string, payload externalAnalysisRequest) (string, error) {
	if proposal.Status != "credit_analysis_in_progress" {
		return "", errors.New("proposta nao esta aguardando callback de credito")
	}

	creditAnalysis := backend.AnalysisResult{
		ProposalID:   proposal.ProposalID,
		AnalysisType: "credit",
		Result:       payload.Result,
		Provider:     payload.Provider,
		Score:        payload.Score,
		Reason:       payload.Reason,
	}
	if err := s.storeAnalysisResult(ctx, proposal.ProposalID, correlationID, creditAnalysis); err != nil {
		return "", errors.New("falha ao registrar callback externo de credito")
	}
	if final, done := finalizeFromResult(creditAnalysis); done {
		if err := s.updateProposalStatus(ctx, proposal.ProposalID, final, correlationID); err != nil {
			return "", errors.New("falha ao atualizar status da proposta")
		}
		s.sendNotification(ctx, proposal.ProposalID, customer.Email, final, statusMessage(final), correlationID)
		return final, nil
	}

	if err := s.moveToFraudStage(ctx, proposal.ProposalID, customer.Email, correlationID); err != nil {
		return "", errors.New("falha ao mover proposta para antifraude")
	}
	if s.externalFraud {
		return "fraud_analysis_in_progress", nil
	}

	fraudAnalysis, err := s.executeFraudAnalysis(ctx, proposal.ProposalID, customer, correlationID)
	if err != nil {
		return "", errors.New("falha ao executar antifraude apos callback de credito")
	}

	finalStatus := finalStatusFromFraudResult(fraudAnalysis.Result)
	if err := s.updateProposalStatus(ctx, proposal.ProposalID, finalStatus, correlationID); err != nil {
		return "", errors.New("falha ao atualizar status final da proposta")
	}
	s.sendNotification(ctx, proposal.ProposalID, customer.Email, finalStatus, statusMessage(finalStatus), correlationID)
	return finalStatus, nil
}

func (s *server) applyExternalFraudResult(ctx context.Context, proposal backend.Proposal, customer backend.Customer, correlationID string, payload externalAnalysisRequest) (string, error) {
	if proposal.Status != "fraud_analysis_in_progress" {
		return "", errors.New("proposta nao esta aguardando callback de antifraude")
	}

	fraudAnalysis := backend.AnalysisResult{
		ProposalID:   proposal.ProposalID,
		AnalysisType: "fraud",
		Result:       payload.Result,
		Provider:     payload.Provider,
		Score:        payload.Score,
		Reason:       payload.Reason,
	}
	if err := s.storeAnalysisResult(ctx, proposal.ProposalID, correlationID, fraudAnalysis); err != nil {
		return "", errors.New("falha ao registrar callback externo de antifraude")
	}

	finalStatus := finalStatusFromFraudResult(fraudAnalysis.Result)
	if err := s.updateProposalStatus(ctx, proposal.ProposalID, finalStatus, correlationID); err != nil {
		return "", errors.New("falha ao atualizar status final da proposta")
	}
	s.sendNotification(ctx, proposal.ProposalID, customer.Email, finalStatus, statusMessage(finalStatus), correlationID)
	return finalStatus, nil
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

func (s *server) moveToFraudStage(ctx context.Context, proposalID, recipient, correlationID string) error {
	if err := s.updateProposalStatus(ctx, proposalID, "fraud_analysis_in_progress", correlationID); err != nil {
		return err
	}
	s.sendNotification(ctx, proposalID, recipient, "fraud_analysis_in_progress", "Sua proposta entrou em analise antifraude.", correlationID)
	return nil
}

func (s *server) executeFraudAnalysis(ctx context.Context, proposalID string, customer backend.Customer, correlationID string) (backend.AnalysisResult, error) {
	var fraudAnalysis backend.AnalysisResult
	if err := s.fraud.Post(ctx, "/internal/proposals/"+proposalID+"/fraud-analysis", correlationID, map[string]any{
		"customer": customer,
	}, &fraudAnalysis); err != nil {
		return fraudAnalysis, err
	}
	if err := s.storeAnalysisResult(ctx, proposalID, correlationID, fraudAnalysis); err != nil {
		return fraudAnalysis, err
	}
	return fraudAnalysis, nil
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

func isValidExternalAnalysisResult(result string) bool {
	switch result {
	case "approved", "manual_review", "rejected":
		return true
	default:
		return false
	}
}

func finalStatusFromFraudResult(result string) string {
	switch result {
	case "manual_review":
		return "manual_review"
	case "rejected":
		return "rejected"
	default:
		return "approved"
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
