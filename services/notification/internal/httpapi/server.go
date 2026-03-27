package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"creditflow/services/notification/internal/domain"
	"creditflow/services/notification/internal/security"
)

const headerCorrelationID = "X-Correlation-Id"

type NotificationStore interface {
	Create(ctx context.Context, notification domain.Notification) error
	ListByProposalID(ctx context.Context, proposalID string) ([]domain.Notification, error)
}

type server struct {
	store NotificationStore
}

type createNotificationRequest struct {
	Channel       string `json:"channel"`
	Template      string `json:"template"`
	Recipient     string `json:"recipient"`
	Message       string `json:"message"`
	TriggerStatus string `json:"trigger_status"`
}

type errorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewServer(store NotificationStore) http.Handler {
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
	if len(segments) != 2 || segments[0] == "" || segments[1] != "notifications" {
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
		return
	}

	proposalID := segments[0]

	switch r.Method {
	case http.MethodPost:
		s.createNotification(w, r, correlationID, proposalID)
	case http.MethodGet:
		s.listNotifications(w, r, correlationID, proposalID)
	default:
		writeError(w, http.StatusNotFound, correlationID, "not_found", "rota nao encontrada", nil)
	}
}

func (s *server) createNotification(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	var payload createNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "payload invalido", nil)
		return
	}

	if strings.TrimSpace(payload.Channel) == "" || strings.TrimSpace(payload.Template) == "" || strings.TrimSpace(payload.Recipient) == "" {
		writeError(w, http.StatusBadRequest, correlationID, "invalid_request", "campos obrigatorios ausentes", map[string]any{
			"required_fields": []string{"channel", "template", "recipient"},
		})
		return
	}

	notification := domain.NewNotification(
		proposalID,
		payload.Channel,
		payload.Template,
		payload.Recipient,
		payload.Message,
		payload.TriggerStatus,
		time.Now().UTC(),
	)
	if err := s.store.Create(r.Context(), notification); err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao registrar notificacao", nil)
		return
	}

	notification.Recipient = security.MaskRecipient(notification.Channel, notification.Recipient)
	writeJSON(w, http.StatusAccepted, notification)
}

func (s *server) listNotifications(w http.ResponseWriter, r *http.Request, correlationID, proposalID string) {
	notifications, err := s.store.ListByProposalID(r.Context(), proposalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, correlationID, "internal_error", "falha ao listar notificacoes", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"proposal_id":   proposalID,
		"notifications": notifications,
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
