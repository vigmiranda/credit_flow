package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"creditflow/services/proposal/internal/domain"
)

const headerCorrelationID = "X-Correlation-Id"

type ProposalStore interface {
	Create(ctx context.Context, proposal domain.Proposal) error
	GetByID(ctx context.Context, proposalID string) (domain.Proposal, error)
	UpdateStatus(ctx context.Context, proposalID, status string, updatedAt time.Time) (domain.Proposal, error)
	CreateAnalysisResult(ctx context.Context, result domain.AnalysisResult) error
	ListAnalysisResults(ctx context.Context, proposalID string) ([]domain.AnalysisResult, error)
}

type server struct {
	store ProposalStore
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type createAnalysisResultRequest struct {
	AnalysisType string `json:"analysis_type"`
	Result       string `json:"result"`
	Provider     string `json:"provider"`
	Score        int    `json:"score"`
	Reason       string `json:"reason"`
}

func NewServer(store ProposalStore) http.Handler {
	return &server{store: store}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	correlationID := getOrCreateCorrelationID(r)
	w.Header().Set(headerCorrelationID, correlationID)

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	case r.Method == http.MethodPost && r.URL.Path == "/internal/proposals":
		s.createProposal(w, r, correlationID)
		return
	case strings.HasPrefix(r.URL.Path, "/internal/proposals/"):
		s.handleProposalRoutes(w, r, correlationID)
		return
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) createProposal(w http.ResponseWriter, r *http.Request, correlationID string) {
	proposal := domain.NewProposal(correlationID, time.Now())
	if err := s.store.Create(r.Context(), proposal); err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao criar proposta", nil)
		return
	}

	writeJSON(w, http.StatusCreated, proposal)
}

func (s *server) handleProposalRoutes(w http.ResponseWriter, r *http.Request, correlationID string) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/internal/proposals/")
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

	if len(segments) == 2 && segments[1] == "status" && r.Method == http.MethodPatch {
		s.updateProposalStatus(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 2 && segments[1] == "analysis-results" && r.Method == http.MethodPost {
		s.createAnalysisResult(w, r, correlationID, proposalID)
		return
	}

	if len(segments) == 2 && segments[1] == "analysis-results" && r.Method == http.MethodGet {
		s.listAnalysisResults(w, r, correlationID, proposalID)
		return
	}

	writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
}

func (s *server) getProposal(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	proposal, err := s.store.GetByID(r.Context(), proposalID)
	if err != nil {
		if errors.Is(err, domain.ErrProposalNotFound) {
			writeError(w, http.StatusNotFound, correlationID, "not_found", "proposta nao encontrada", nil)
			return
		}

		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao consultar proposta", nil)
		return
	}

	writeJSON(w, http.StatusOK, proposal)
}

func (s *server) updateProposalStatus(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if !domain.IsValidStatus(payload.Status) {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "status invalido", map[string]any{
			"status": payload.Status,
		})
		return
	}

	proposal, err := s.store.UpdateStatus(r.Context(), proposalID, payload.Status, time.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrProposalNotFound) {
			writeError(w, http.StatusNotFound, correlationID, "not_found", "proposta nao encontrada", nil)
			return
		}

		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao atualizar status", nil)
		return
	}

	writeJSON(w, http.StatusOK, proposal)
}

func (s *server) createAnalysisResult(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload createAnalysisResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if !domain.IsValidAnalysisType(payload.AnalysisType) {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "analysis_type invalido", map[string]any{
			"analysis_type": payload.AnalysisType,
		})
		return
	}

	if !domain.IsValidAnalysisResult(payload.Result) {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "result invalido", map[string]any{
			"result": payload.Result,
		})
		return
	}

	result := domain.NewAnalysisResult(
		proposalID,
		payload.AnalysisType,
		payload.Result,
		payload.Provider,
		payload.Reason,
		payload.Score,
		time.Now().UTC(),
	)
	if err := s.store.CreateAnalysisResult(r.Context(), result); err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao registrar analise", nil)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *server) listAnalysisResults(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	results, err := s.store.ListAnalysisResults(r.Context(), proposalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao listar analises", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"proposal_id":      proposalID,
		"analysis_results": results,
	})
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
