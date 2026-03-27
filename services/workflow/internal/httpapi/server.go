package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"creditflow/services/workflow/internal/backend"
)

const headerCorrelationID = "X-Correlation-Id"

type gateway interface {
	Get(ctx context.Context, path, correlationID string, out any) error
	Post(ctx context.Context, path, correlationID string, payload any, out any) error
	Patch(ctx context.Context, path, correlationID string, payload any, out any) error
}

type server struct {
	proposals gateway
	customers gateway
	documents gateway
	credit    gateway
	fraud     gateway
	delay     time.Duration
}

type workflowResponse struct {
	ProposalID  string                   `json:"proposal_id"`
	FinalStatus string                   `json:"final_status"`
	Results     []backend.AnalysisResult `json:"results"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(proposals gateway, customers gateway, documents gateway, credit gateway, fraud gateway, delay time.Duration) http.Handler {
	return &server{
		proposals: proposals,
		customers: customers,
		documents: documents,
		credit:    credit,
		fraud:     fraud,
		delay:     delay,
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

	result, err := s.runWorkflow(r.Context(), segments[0], correlationID)
	if err != nil {
		writeError(w, http.StatusBadGateway, correlationID, "upstream_error", err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusAccepted, result)
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
		return response, nil
	}

	if err := s.updateProposalStatus(ctx, proposalID, "credit_analysis_in_progress", correlationID); err != nil {
		return response, err
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
		return response, nil
	}

	if err := s.updateProposalStatus(ctx, proposalID, "fraud_analysis_in_progress", correlationID); err != nil {
		return response, err
	}
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

	response.FinalStatus = finalStatus
	return response, nil
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
