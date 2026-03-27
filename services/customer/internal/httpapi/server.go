package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"creditflow/services/customer/internal/domain"
)

const headerCorrelationID = "X-Correlation-Id"

type CustomerStore interface {
	UpsertByProposalID(ctx context.Context, customer domain.Customer) (domain.Customer, error)
	GetByProposalID(ctx context.Context, proposalID string) (domain.Customer, error)
}

type server struct {
	store CustomerStore
}

type customerRequest struct {
	FullName      string  `json:"full_name"`
	CPF           string  `json:"cpf"`
	BirthDate     string  `json:"birth_date"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	MonthlyIncome float64 `json:"monthly_income"`
	Address       string  `json:"address"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(store CustomerStore) http.Handler {
	return &server{store: store}
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
	if len(segments) != 2 || segments[0] == "" || segments[1] != "customer" {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
		return
	}

	proposalID := segments[0]

	switch r.Method {
	case http.MethodPost:
		s.upsertCustomer(w, r, correlationID, proposalID)
	case http.MethodGet:
		s.getCustomer(w, r, correlationID, proposalID)
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) upsertCustomer(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload customerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	input := domain.CustomerInput{
		FullName:      payload.FullName,
		CPF:           payload.CPF,
		BirthDate:     payload.BirthDate,
		Email:         payload.Email,
		Phone:         payload.Phone,
		MonthlyIncome: payload.MonthlyIncome,
		Address:       payload.Address,
	}

	if details := input.Validate(); details != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "dados do cliente invalidos", details)
		return
	}

	customer := domain.NewCustomer(proposalID, input, time.Now().UTC())
	saved, err := s.store.UpsertByProposalID(r.Context(), customer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao salvar cliente", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, saved)
}

func (s *server) getCustomer(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	customer, err := s.store.GetByProposalID(r.Context(), proposalID)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			writeError(w, http.StatusNotFound, correlationID, "not_found", "cliente nao encontrado", nil)
			return
		}

		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao consultar cliente", nil)
		return
	}

	writeJSON(w, http.StatusOK, customer)
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
