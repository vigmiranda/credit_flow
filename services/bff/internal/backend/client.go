package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const headerCorrelationID = "X-Correlation-Id"

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type Proposal struct {
	ProposalID    string `json:"proposal_id"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Customer struct {
	CustomerID    string  `json:"customer_id"`
	ProposalID    string  `json:"proposal_id"`
	FullName      string  `json:"full_name"`
	CPF           string  `json:"cpf"`
	BirthDate     string  `json:"birth_date"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	MonthlyIncome float64 `json:"monthly_income"`
	Address       string  `json:"address,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Document struct {
	DocumentID  string  `json:"document_id"`
	ProposalID  string  `json:"proposal_id"`
	Type        string  `json:"document_type"`
	FileName    string  `json:"file_name"`
	ContentType string  `json:"content_type"`
	FileKey     string  `json:"file_key"`
	Status      string  `json:"status"`
	UploadURL   string  `json:"upload_url"`
	UploadedAt  *string `json:"uploaded_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type DocumentList struct {
	ProposalID string     `json:"proposal_id"`
	Documents  []Document `json:"documents"`
}

type AnalysisResult struct {
	AnalysisID   string `json:"analysis_id,omitempty"`
	ProposalID   string `json:"proposal_id"`
	AnalysisType string `json:"analysis_type"`
	Result       string `json:"result"`
	Provider     string `json:"provider"`
	Score        int    `json:"score"`
	Reason       string `json:"reason"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type AnalysisResultList struct {
	ProposalID      string           `json:"proposal_id"`
	AnalysisResults []AnalysisResult `json:"analysis_results"`
}

type StatusHistoryEntry struct {
	StatusHistoryID string `json:"status_history_id"`
	ProposalID      string `json:"proposal_id"`
	Status          string `json:"status"`
	Source          string `json:"source"`
	CreatedAt       string `json:"created_at"`
}

type StatusHistoryList struct {
	ProposalID    string               `json:"proposal_id"`
	StatusHistory []StatusHistoryEntry `json:"status_history"`
}

type Notification struct {
	NotificationID string `json:"notification_id"`
	ProposalID     string `json:"proposal_id"`
	Channel        string `json:"channel"`
	Template       string `json:"template"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
	Status         string `json:"status"`
	TriggerStatus  string `json:"trigger_status"`
	SentAt         string `json:"sent_at"`
	CreatedAt      string `json:"created_at"`
}

type NotificationList struct {
	ProposalID    string         `json:"proposal_id"`
	Notifications []Notification `json:"notifications"`
}

type ErrorResponse struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return c.request(ctx, http.MethodPost, path, correlationID, payload, out)
}

func (c *Client) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return c.request(ctx, http.MethodGet, path, correlationID, nil, out)
}

func (c *Client) Patch(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return c.request(ctx, http.MethodPatch, path, correlationID, payload, out)
}

func (c *Client) request(ctx context.Context, method, path, correlationID string, payload any, out any) (int, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set(headerCorrelationID, correlationID)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return resp.StatusCode, fmt.Errorf("backend status %d", resp.StatusCode)
		}
		if apiErr.Message == "" {
			apiErr.Message = "backend error"
		}
		return resp.StatusCode, fmt.Errorf("%s", apiErr.Message)
	}

	if out == nil {
		return resp.StatusCode, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return resp.StatusCode, err
	}

	return resp.StatusCode, nil
}
