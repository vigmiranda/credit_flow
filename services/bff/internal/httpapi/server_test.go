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

func testWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func setWebhookHeaders(req *http.Request, eventID string, timestamp time.Time) {
	req.Header.Set(headerWebhookEvent, eventID)
	req.Header.Set(headerWebhookTime, timestamp.Format(time.RFC3339))
}
