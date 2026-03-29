package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"mime/multipart"
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
	postFunc   func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc    func(ctx context.Context, path, correlationID string, out any) (int, error)
	uploadFunc func(ctx context.Context, path, correlationID, fieldName, fileName, contentType string, content []byte, out any) (int, error)
}

func (s stubDocumentGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubDocumentGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
}

func (s stubDocumentGateway) UploadMultipart(ctx context.Context, path, correlationID, fieldName, fileName, contentType string, content []byte, out any) (int, error) {
	return s.uploadFunc(ctx, path, correlationID, fieldName, fileName, contentType, content, out)
}

type stubWorkflowGateway struct {
	postFunc func(ctx context.Context, path, correlationID string, payload any, out any) (int, error)
	getFunc  func(ctx context.Context, path, correlationID string, out any) (int, error)
}

func (s stubWorkflowGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) (int, error) {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubWorkflowGateway) Get(ctx context.Context, path, correlationID string, out any) (int, error) {
	return s.getFunc(ctx, path, correlationID, out)
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
			uploadFunc: func(context.Context, string, string, string, string, string, []byte, any) (int, error) {
				return 0, errors.New("unexpected")
			},
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
		"",
		time.Minute,
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
			uploadFunc: func(context.Context, string, string, string, string, string, []byte, any) (int, error) {
				return 0, errors.New("unexpected")
			},
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
		"",
		time.Minute,
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
			uploadFunc: func(context.Context, string, string, string, string, string, []byte, any) (int, error) {
				return 0, errors.New("unexpected")
			},
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
		"",
		time.Minute,
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

func TestUploadDocumentContentTriggersWorkflow(t *testing.T) {
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
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("unexpected") },
			uploadFunc: func(_ context.Context, path, _ string, fieldName, fileName, contentType string, content []byte, out any) (int, error) {
				if path != "/internal/proposals/prop_123/documents/doc_123/content" {
					t.Fatalf("unexpected upload path %s", path)
				}
				if fieldName != "file" || fileName != "rg.jpg" || len(content) == 0 {
					t.Fatal("unexpected upload metadata")
				}
				if contentType != "" && contentType != "application/octet-stream" {
					t.Fatalf("unexpected content type %s", contentType)
				}
				document := out.(*backend.Document)
				document.DocumentID = "doc_123"
				document.Status = "uploaded"
				return http.StatusAccepted, nil
			},
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
		"",
		time.Minute,
	)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "rg.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("write form content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/proposals/prop_123/documents/doc_123/content", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	select {
	case <-workflowTriggered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected workflow trigger after content upload")
	}
	if !notificationCalled {
		t.Fatal("expected notification after document upload")
	}
}

func TestStorageWebhookTriggersWorkflow(t *testing.T) {
	workflowTriggered := make(chan struct{}, 1)
	notificationCalled := false
	secret := "local-webhook-secret"
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
			uploadFunc: func(context.Context, string, string, string, string, string, []byte, any) (int, error) {
				return 0, errors.New("unexpected")
			},
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
		secret,
		time.Minute,
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","document_id":"doc_123","provider":"minio","event_type":"object_created"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/storage/document-uploaded", body)
	req.Header.Set(headerWebhookSig, testWebhookSignature(secret, body.Bytes()))
	setWebhookHeaders(req, "evt_storage_001", time.Now().UTC())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	select {
	case <-workflowTriggered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected workflow trigger after storage webhook")
	}
	if !notificationCalled {
		t.Fatal("expected notification after storage webhook")
	}
}

func TestStorageWebhookRejectsInvalidSignature(t *testing.T) {
	srv := NewServer(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{},
		stubNotificationGateway{},
		"local-webhook-secret",
		time.Minute,
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","document_id":"doc_123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/storage/document-uploaded", body)
	req.Header.Set(headerWebhookSig, "sha256=deadbeef")
	setWebhookHeaders(req, "evt_storage_001", time.Now().UTC())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestAnalysisWebhookTriggersWorkflowExternalCallback(t *testing.T) {
	srv := NewServer(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				if path != "/internal/proposals/prop_123/external-analyses/credit" {
					t.Fatalf("unexpected workflow path %s", path)
				}
				body := payload.(map[string]any)
				if body["result"] != "approved" {
					t.Fatalf("unexpected result %v", body["result"])
				}
				response := out.(*map[string]any)
				*response = map[string]any{
					"proposal_id":     "prop_123",
					"analysis_type":   "credit",
					"proposal_status": "fraud_analysis_in_progress",
				}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved","score":710}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
	req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
	setWebhookHeaders(req, "evt_credit_001", time.Now().UTC())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["replay_status"] != "processed" {
		t.Fatalf("expected replay_status processed, got %v", response["replay_status"])
	}
}

func TestAnalysisWebhookIgnoresReplay(t *testing.T) {
	callCount := 0
	srv := NewServer(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				callCount++
				response := out.(*map[string]any)
				*response = map[string]any{
					"proposal_id": "prop_123",
				}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
	)

	send := func() *httptest.ResponseRecorder {
		body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
		req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
		setWebhookHeaders(req, "evt_credit_001", time.Now().UTC())
		resp := httptest.NewRecorder()
		srv.ServeHTTP(resp, req)
		return resp
	}

	first := send()
	second := send()

	if first.Code != http.StatusAccepted || second.Code != http.StatusAccepted {
		t.Fatalf("expected accepted responses, got %d and %d", first.Code, second.Code)
	}
	if callCount != 1 {
		t.Fatalf("expected a single workflow call, got %d", callCount)
	}
}

func TestAnalysisWebhookRejectsStaleTimestamp(t *testing.T) {
	srv := NewServer(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
	req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
	setWebhookHeaders(req, "evt_credit_001", time.Now().UTC().Add(-10*time.Minute))
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestWebhookAuditListReturnsPersistedRecords(t *testing.T) {
	replayStore := NewMemoryWebhookReplayStore()
	auditStore := NewMemoryWebhookAuditStore()
	srv := NewServerWithDependencies(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				response := out.(*map[string]any)
				*response = map[string]any{
					"proposal_id": "prop_123",
				}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
		replayStore,
		auditStore,
		nil,
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
	req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
	setWebhookHeaders(req, "evt_credit_audit", time.Now().UTC())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/internal/webhooks/audit?event_id=evt_credit_audit", nil)
	listResp := httptest.NewRecorder()
	srv.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listResp.Code)
	}

	var payload struct {
		Count   int                  `json:"count"`
		Records []WebhookAuditRecord `json:"records"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Count != 1 || len(payload.Records) != 1 {
		t.Fatalf("expected 1 audit record, got count=%d len=%d", payload.Count, len(payload.Records))
	}
	if payload.Records[0].ReplayStatus != "processed" {
		t.Fatalf("expected replay status processed, got %s", payload.Records[0].ReplayStatus)
	}
}

func TestWebhookReplayReleaseAllowsManualReplay(t *testing.T) {
	replayStore := NewMemoryWebhookReplayStore()
	auditStore := NewMemoryWebhookAuditStore()
	callCount := 0
	srv := NewServerWithDependencies(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				callCount++
				response := out.(*map[string]any)
				*response = map[string]any{
					"proposal_id": "prop_123",
				}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
		replayStore,
		auditStore,
		nil,
	)

	send := func() int {
		body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
		req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
		setWebhookHeaders(req, "evt_credit_release", time.Now().UTC())
		resp := httptest.NewRecorder()
		srv.ServeHTTP(resp, req)
		return resp.Code
	}

	if status := send(); status != http.StatusAccepted {
		t.Fatalf("expected first webhook accepted, got %d", status)
	}
	if status := send(); status != http.StatusAccepted {
		t.Fatalf("expected duplicate webhook accepted, got %d", status)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 workflow call after duplicate, got %d", callCount)
	}

	releaseReq := httptest.NewRequest(http.MethodPost, "/internal/webhooks/audit/evt_credit_release/replay-release", nil)
	releaseResp := httptest.NewRecorder()
	srv.ServeHTTP(releaseResp, releaseReq)

	if releaseResp.Code != http.StatusAccepted {
		t.Fatalf("expected release status %d, got %d", http.StatusAccepted, releaseResp.Code)
	}

	if status := send(); status != http.StatusAccepted {
		t.Fatalf("expected replayed webhook accepted, got %d", status)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 workflow calls after replay release, got %d", callCount)
	}

	record, ok, err := auditStore.Get(context.Background(), "evt_credit_release")
	if err != nil || !ok {
		t.Fatalf("expected audit record after replay release, ok=%v err=%v", ok, err)
	}
	if record.ReplayReleasedAt == "" {
		t.Fatal("expected replay released timestamp to be stored")
	}
}

func TestGetProposalIncludesWebhookAudit(t *testing.T) {
	auditStore := NewMemoryWebhookAuditStore()
	if err := auditStore.Upsert(context.Background(), WebhookAuditRecord{
		EventID:            "evt_credit_audit",
		CallbackType:       "credit",
		ProposalID:         "prop_123",
		Provider:           "partner-credit",
		EventType:          "analysis_completed",
		ReplayStatus:       "processed",
		ProcessingStatus:   "processed",
		ReceivedAt:         "2026-03-29T12:00:00Z",
		ProcessedAt:        "2026-03-29T12:00:01Z",
		RetentionExpiresAt: "2026-04-05T12:00:00Z",
	}); err != nil {
		t.Fatalf("seed audit store: %v", err)
	}

	srv := NewServerWithDependencies(
		stubProposalGateway{
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				if path != "/internal/proposals/prop_123" {
					return 0, errors.New("unexpected proposal lookup")
				}
				proposal := out.(*backend.Proposal)
				*proposal = backend.Proposal{
					ProposalID: "prop_123",
					Protocol:   "P-001",
					Status:     "credit_analysis_in_progress",
					CreatedAt:  "2026-03-29T11:00:00Z",
					UpdatedAt:  "2026-03-29T12:00:00Z",
				}
				return http.StatusOK, nil
			},
			postFunc:  func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			patchFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubCustomerGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("not found") },
		},
		stubDocumentGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("not found") },
			uploadFunc: func(context.Context, string, string, string, string, string, []byte, any) (int, error) {
				return 0, errors.New("unexpected")
			},
		},
		stubWorkflowGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
		},
		stubNotificationGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc:  func(context.Context, string, string, any) (int, error) { return 0, errors.New("not found") },
		},
		"",
		time.Minute,
		NewMemoryWebhookReplayStore(),
		auditStore,
		nil,
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

	if len(proposal.WebhookAudit) != 1 {
		t.Fatalf("expected 1 webhook audit record, got %d", len(proposal.WebhookAudit))
	}
	if proposal.WebhookAudit[0].EventID != "evt_credit_audit" {
		t.Fatalf("unexpected audit event id %s", proposal.WebhookAudit[0].EventID)
	}
}

func TestAnalysisWebhookRejectsProviderOutsidePolicy(t *testing.T) {
	auditStore := NewMemoryWebhookAuditStore()
	srv := NewServerWithPolicyConfig(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) {
				t.Fatal("workflow should not be called for unauthorized provider")
				return 0, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
		NewMemoryWebhookReplayStore(),
		auditStore,
		NewMemoryWebhookRateLimitStore(),
		nil,
		WebhookPolicyConfig{
			AllowedCreditProviders: []string{"partner-credit"},
		},
	)

	body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"rogue-credit","result":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
	req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
	setWebhookHeaders(req, "evt_credit_policy", time.Now().UTC())
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.Code)
	}

	record, ok, err := auditStore.Get(context.Background(), "evt_credit_policy")
	if err != nil || !ok {
		t.Fatalf("expected audit record for rejected provider, ok=%v err=%v", ok, err)
	}
	if record.ErrorCode != "invalid_provider" {
		t.Fatalf("expected invalid_provider audit code, got %s", record.ErrorCode)
	}
}

func TestAnalysisWebhookRejectsWhenRateLimitExceeded(t *testing.T) {
	auditStore := NewMemoryWebhookAuditStore()
	callCount := 0
	srv := NewServerWithPolicyConfig(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(_ context.Context, path, _ string, payload any, out any) (int, error) {
				callCount++
				response := out.(*map[string]any)
				*response = map[string]any{"proposal_id": "prop_123"}
				return http.StatusAccepted, nil
			},
		},
		stubNotificationGateway{},
		"local-webhook-secret",
		5*time.Minute,
		NewMemoryWebhookReplayStore(),
		auditStore,
		NewMemoryWebhookRateLimitStore(),
		nil,
		WebhookPolicyConfig{
			CreditRateLimit:        1,
			CreditRateWindow:       time.Minute,
			AllowedCreditProviders: []string{"partner-credit"},
		},
	)

	send := func(eventID string) *httptest.ResponseRecorder {
		body := bytes.NewBufferString(`{"proposal_id":"prop_123","provider":"partner-credit","result":"approved"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/partners/credit-analysis", body)
		req.Header.Set(headerWebhookSig, testWebhookSignature("local-webhook-secret", body.Bytes()))
		setWebhookHeaders(req, eventID, time.Now().UTC())
		resp := httptest.NewRecorder()
		srv.ServeHTTP(resp, req)
		return resp
	}

	first := send("evt_credit_rate_001")
	second := send("evt_credit_rate_002")

	if first.Code != http.StatusAccepted {
		t.Fatalf("expected first webhook accepted, got %d", first.Code)
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second webhook rate limited, got %d", second.Code)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 workflow call before rate limit, got %d", callCount)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on rate-limited response")
	}

	record, ok, err := auditStore.Get(context.Background(), "evt_credit_rate_002")
	if err != nil || !ok {
		t.Fatalf("expected audit record for rate-limited event, ok=%v err=%v", ok, err)
	}
	if record.ErrorCode != "rate_limited" {
		t.Fatalf("expected rate_limited audit code, got %s", record.ErrorCode)
	}
}

func TestWebhookAuditCleanupRemovesExpiredRecords(t *testing.T) {
	auditStore := NewMemoryWebhookAuditStore()
	if err := auditStore.Upsert(context.Background(), WebhookAuditRecord{
		EventID:            "evt_expired",
		CallbackType:       "storage",
		ProposalID:         "prop_123",
		ReceivedAt:         "2026-03-29T10:00:00Z",
		RetentionExpiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed expired audit record: %v", err)
	}
	if err := auditStore.Upsert(context.Background(), WebhookAuditRecord{
		EventID:            "evt_live",
		CallbackType:       "credit",
		ProposalID:         "prop_123",
		ReceivedAt:         "2026-03-29T10:00:00Z",
		RetentionExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed live audit record: %v", err)
	}

	srv := NewServerWithDependencies(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{},
		stubNotificationGateway{},
		"",
		time.Minute,
		NewMemoryWebhookReplayStore(),
		auditStore,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/internal/webhooks/audit/cleanup", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	var payload struct {
		Removed int `json:"removed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Removed != 1 {
		t.Fatalf("expected 1 removed record, got %d", payload.Removed)
	}

	records, err := auditStore.List(context.Background(), WebhookAuditFilter{ProposalID: "prop_123", Limit: 10})
	if err != nil {
		t.Fatalf("list audit records: %v", err)
	}
	if len(records) != 1 || records[0].EventID != "evt_live" {
		t.Fatalf("expected only live record after cleanup, got %+v", records)
	}
}

func TestOperationsOverviewAggregatesWorkflowAndAuditSignals(t *testing.T) {
	auditStore := NewMemoryWebhookAuditStore()
	if err := auditStore.Upsert(context.Background(), WebhookAuditRecord{
		EventID:            "evt_rate_limited",
		CallbackType:       "credit",
		Provider:           "partner-credit",
		ProcessingStatus:   "rejected",
		ErrorCode:          "rate_limited",
		ReceivedAt:         time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
		RetentionExpiresAt: time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed audit store: %v", err)
	}

	srv := NewServerWithDependencies(
		stubProposalGateway{},
		stubCustomerGateway{},
		stubDocumentGateway{},
		stubWorkflowGateway{
			postFunc: func(context.Context, string, string, any, any) (int, error) { return 0, errors.New("unexpected") },
			getFunc: func(_ context.Context, path, _ string, out any) (int, error) {
				switch path {
				case "/metrics":
					metrics := out.(*workflowOperationsMetrics)
					*metrics = workflowOperationsMetrics{
						Service:       "workflow",
						TotalRequests: 10,
						TotalErrors:   2,
						AverageMS:     120,
						Queue: workflowQueueView{
							Enqueued:   5,
							Processed:  4,
							Retried:    1,
							DeadLetter: 1,
							Depth:      0,
							DLQDepth:   1,
						},
					}
					return http.StatusOK, nil
				case "/internal/dlq":
					payload := out.(*deadLetterListResponse)
					*payload = deadLetterListResponse{Count: 1}
					return http.StatusOK, nil
				default:
					t.Fatalf("unexpected workflow get path %s", path)
				}
				return 0, nil
			},
		},
		stubNotificationGateway{},
		"",
		time.Minute,
		NewMemoryWebhookReplayStore(),
		auditStore,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/internal/operations/overview", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var payload operationsOverviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.WorkflowMetrics == nil || payload.WorkflowMetrics.Queue.DLQDepth != 1 {
		t.Fatal("expected workflow metrics with dlq depth")
	}
	if payload.WorkflowDeadLetters.Count != 1 {
		t.Fatalf("expected 1 dead letter, got %d", payload.WorkflowDeadLetters.Count)
	}
	if payload.CallbackAuditSummary.RateLimited != 1 {
		t.Fatalf("expected rate_limited summary 1, got %d", payload.CallbackAuditSummary.RateLimited)
	}
	if len(payload.Alerts) < 3 {
		t.Fatalf("expected aggregated alerts, got %d", len(payload.Alerts))
	}
}

func testWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func setWebhookHeaders(req *http.Request, eventID string, timestamp time.Time) {
	req.Header.Set(headerWebhookEvent, eventID)
	req.Header.Set(headerWebhookTime, timestamp.Format(time.RFC3339))
}
