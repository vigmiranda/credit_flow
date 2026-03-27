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
					Email:      "maria@example.com",
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
	if len(proposal.Documents) != 1 || proposal.Documents[0].DocumentID != "doc_123" {
		t.Fatal("expected aggregated documents in response")
	}
	if len(proposal.AnalysisResults) != 1 || proposal.AnalysisResults[0].AnalysisType != "document" {
		t.Fatal("expected aggregated analysis results in response")
	}
}

func TestUpsertCustomerUpdatesProposalStatus(t *testing.T) {
	patchCalled := false
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
}

func TestMarkDocumentReceivedTriggersWorkflow(t *testing.T) {
	workflowTriggered := make(chan struct{}, 1)
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
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
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
}
