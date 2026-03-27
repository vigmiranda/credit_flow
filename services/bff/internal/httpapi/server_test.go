package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/bff/internal/backend"
)

type stubProposalGateway struct {
	postFunc  func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc   func(ctx context.Context, path, correlationID string, out any) (int, error)
	patchFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
}

func (s stubProposalGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubProposalGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
}

func (s stubProposalGateway) Patch(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.patchFunc(ctx, path, correlationID, payload, out)
}

type stubCustomerGateway struct {
	postFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc  func(ctx context.Context, path, correlationID string, out any) (int, error)
}

func (s stubCustomerGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubCustomerGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
}

type stubDocumentGateway struct {
	postFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc  func(ctx context.Context, path, correlationID string, out any) (int, error)
}

func (s stubDocumentGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubDocumentGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
}

type stubWorkflowGateway struct {
	postFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
}

func (s stubWorkflowGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

type stubNotificationGateway struct {
	postFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc  func(ctx context.Context, path, correlationID string, out any) (int, error)
}

func (s stubNotificationGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubNotificationGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
}

func TestGetProposalAggregatesCustomerAndDocuments(t *testing.T) {
	srv := NewServer(
		stubProposalGateway{
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				switch path {
				case "/internal/proposals/prop_123":
					proposal := out.(*backend.Proposal)
					*proposal = backend.Proposal{
						ProposalID: "prop_123",
						Protocol:   "P-001",
						Status:     "documents_pending",
						CreatedAt:  "2026-03-27T10:00:00Z",
						UpdatedAt:  "2026-03-27T10:05:00Z",
					}
					return http.StatusOK, nil
				case "/internal/proposals/prop_123/analysis-results":
					results := out.(*backend.AnalysisResultList)
					*results = backend.AnalysisResultList{
						ProposalID: "prop_123",
						AnalysisResults: []backend.AnalysisResult{
							{AnalysisType: "document", Result: "approved", Provider: "doc", Score: 700},
						},
					}
					return http.StatusOK, nil
				case "/internal/proposals/prop_123/status-history":
					history := out.(*backend.StatusHistoryList)
					*history = backend.StatusHistoryList{
						ProposalID: "prop_123",
						StatusHistory: []backend.StatusHistoryEntry{
							{Status: "created", Source: "proposal_service", CreatedAt: "2026-03-27T10:00:00Z"},
						},
					}
					return http.StatusOK, nil
				default:
					t.Fatalf("unexpected proposal path %s", path)
				}
				return 0, nil
			},
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			patchFunc: func(context.Context, string, string, any, any) (int, error) {
				return 0, errors.New("unexpected")
			},
		},
		stubCustomerGateway{
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				if path != "/internal/proposals/prop_123/customer" {
					t.Fatalf("unexpected customer path %s", path)
				}
				customer := out.(*backend.Customer)
				*customer = backend.Customer{
					CustomerID: "cus_123",
					ProposalID: "prop_123",
					FullName:   "Maria Silva",
					CPF:        "12345678901",
					Email:      "maria@example.com",
					Phone:      "11999999999",
				}
				return http.StatusOK, nil
			},
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubDocumentGateway{
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				if path != "/internal/proposals/prop_123/documents" {
					t.Fatalf("unexpected document path %s", path)
				}
				documents := out.(*backend.DocumentList)
				*documents = backend.DocumentList{
					ProposalID: "prop_123",
					Documents: []backend.Document{
						{DocumentID: "doc_123", ProposalID: "prop_123", Type: "id_front", Status: "uploaded"},
					},
				}
				return http.StatusOK, nil
			},
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubWorkflowGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubNotificationGateway{
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				if path != "/internal/proposals/prop_123/notifications" {
					t.Fatalf("unexpected notifications path %s", path)
				}
				notifications := out.(*backend.NotificationList)
				*notifications = backend.NotificationList{
					ProposalID: "prop_123",
					Notifications: []backend.Notification{
						{
							NotificationID: "ntf_123",
							TriggerStatus:  "documents_pending",
							Message:        "envie os documentos",
							Recipient:      "maria@example.com",
						},
					},
				}
				return http.StatusOK, nil
			},
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proposals/prop_123", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var proposal proposalResponse
	if err := json.NewDecoder(resp.Body).Decode(&proposal); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if proposal.Customer == nil || proposal.Customer.CustomerID != "cus_123" {
		t.Fatal("expected aggregated customer in response")
	}
	if proposal.Customer.CPF != "*******8901" {
		t.Fatalf("expected masked cpf, got %s", proposal.Customer.CPF)
	}
	if proposal.Customer.Email != "m****@example.com" {
		t.Fatalf("expected masked email, got %s", proposal.Customer.Email)
	}
	if proposal.Customer.Phone != "*******9999" {
		t.Fatalf("expected masked phone, got %s", proposal.Customer.Phone)
	}
	if len(proposal.Documents) != 1 || proposal.Documents[0].DocumentID != "doc_123" {
		t.Fatal("expected aggregated documents in response")
	}
	if len(proposal.AnalysisResults) != 1 || proposal.AnalysisResults[0].AnalysisType != "document" {
		t.Fatal("expected aggregated analysis results in response")
	}
	if len(proposal.StatusHistory) != 1 || proposal.StatusHistory[0].Status != "created" {
		t.Fatal("expected aggregated status history in response")
	}
	if len(proposal.Notifications) != 1 || proposal.Notifications[0].NotificationID != "ntf_123" {
		t.Fatal("expected aggregated notifications in response")
	}
	if proposal.Notifications[0].Recipient != "m****@example.com" {
		t.Fatalf("expected masked notification recipient, got %s", proposal.Notifications[0].Recipient)
	}
}

func TestUpsertCustomerUpdatesProposalStatus(t *testing.T) {
	patchCalled := false
	notificationCalled := false
	srv := NewServer(
		stubProposalGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
			patchFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				patchCalled = true
				if path != "/internal/proposals/prop_123/status" {
					t.Fatalf("unexpected patch path %s", path)
				}
				body := payload.(map[string]string)
				if body["status"] != "documents_pending" {
					t.Fatalf("unexpected status %s", body["status"])
				}
				return http.StatusOK, nil
			},
		},
		stubCustomerGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				if path != "/internal/proposals/prop_123/customer" {
					t.Fatalf("unexpected customer path %s", path)
				}
				customer := out.(*backend.Customer)
				request := payload.(customerRequest)
				customer.CustomerID = "cus_123"
				customer.FullName = request.FullName
				return http.StatusAccepted, nil
			},
			getFunc: func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubDocumentGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubWorkflowGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubNotificationGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				notificationCalled = true
				if path != "/internal/proposals/prop_123/notifications" {
					t.Fatalf("unexpected notification path %s", path)
				}
				body := payload.(map[string]any)
				if body["trigger_status"] != "documents_pending" {
					t.Fatalf("unexpected trigger status %v", body["trigger_status"])
				}
				return http.StatusAccepted, nil
			},
			getFunc: func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
		},
	)

	body := bytes.NewBufferString(`{"full_name":"Maria Silva","cpf":"12345678901","birth_date":"1990-01-01","email":"maria@example.com","phone":"11999999999","monthly_income":5000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop_123/customer", body)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}
	if !patchCalled {
		t.Fatal("expected proposal status update after customer upsert")
	}
	if !notificationCalled {
		t.Fatal("expected notification after customer upsert")
	}
}

func TestMarkDocumentReceivedTriggersWorkflow(t *testing.T) {
	workflowTriggered := make(chan struct{}, 1)
	notificationCalled := false
	srv := NewServer(
		stubProposalGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
			patchFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				if path != "/internal/proposals/prop_123/status" {
					t.Fatalf("unexpected patch path %s", path)
				}
				return http.StatusOK, nil
			},
		},
		stubCustomerGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				if path != "/internal/proposals/prop_123/customer" {
					t.Fatalf("unexpected customer lookup path %s", path)
				}
				customer := out.(*backend.Customer)
				customer.Email = "maria@example.com"
				return http.StatusOK, nil
			},
		},
		stubDocumentGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				if path != "/internal/proposals/prop_123/documents/doc_123/received" {
					t.Fatalf("unexpected document path %s", path)
				}
				document := out.(*backend.Document)
				document.DocumentID = "doc_123"
				document.Status = "uploaded"
				return http.StatusAccepted, nil
			},
			getFunc: func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				if path != "/internal/proposals/prop_123/run-analyses" {
					t.Fatalf("unexpected workflow path %s", path)
				}
				workflowTriggered <- struct{}{}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				notificationCalled = true
				if path != "/internal/proposals/prop_123/notifications" {
					t.Fatalf("unexpected notification path %s", path)
				}
				return http.StatusAccepted, nil
			},
			getFunc: func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
		},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop_123/documents/doc_123/received", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	select {
	case <-workflowTriggered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected workflow trigger after document confirmation")
	}
	if !notificationCalled {
		t.Fatal("expected notification after document confirmation")
	}
}
