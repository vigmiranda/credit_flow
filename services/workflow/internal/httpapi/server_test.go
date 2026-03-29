package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/workflow/internal/backend"
	"creditflow/services/workflow/internal/queue"
)

type stubGateway struct {
	getFunc   func(ctx context.Context, path, correlationID string, out any) error
	postFunc  func(ctx context.Context, path, correlationID string, payload any, out any) error
	patchFunc func(ctx context.Context, path, correlationID string, payload any, out any) error
}

func (s stubGateway) Get(ctx context.Context, path, correlationID string, out any) error {
	return s.getFunc(ctx, path, correlationID, out)
}

func (s stubGateway) Post(ctx context.Context, path, correlationID string, payload any, out any) error {
	return s.postFunc(ctx, path, correlationID, payload, out)
}

func (s stubGateway) Patch(ctx context.Context, path, correlationID string, payload any, out any) error {
	return s.patchFunc(ctx, path, correlationID, payload, out)
}

func TestRunAnalysesEnqueuesWorkflow(t *testing.T) {
	workflowQueue := queue.NewMemoryQueue(1)
	srv := NewServer(
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		workflowQueue,
		0,
		2,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/run-analyses", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	var response workflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.FinalStatus != "queued" {
		t.Fatalf("expected queued response, got %s", response.FinalStatus)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	job, err := workflowQueue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue workflow job: %v", err)
	}
	if job.ProposalID != "prop_123" {
		t.Fatalf("expected proposal id prop_123, got %s", job.ProposalID)
	}
}

func TestProcessJobApprovesProposal(t *testing.T) {
	proposalPostCount := 0
	proposalPatchCount := 0
	proposalGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, _ any) error {
			if path != "/internal/proposals/prop_123/analysis-results" {
				t.Fatalf("unexpected proposal post path %s", path)
			}
			proposalPostCount++
			return nil
		},
		patchFunc: func(_ context.Context, path, _ string, _ any, _ any) error {
			if path != "/internal/proposals/prop_123/status" {
				t.Fatalf("unexpected proposal patch path %s", path)
			}
			proposalPatchCount++
			return nil
		},
	}

	customerGateway := stubGateway{
		getFunc: func(_ context.Context, path, _ string, out any) error {
			if path != "/internal/proposals/prop_123/customer" {
				t.Fatalf("unexpected customer path %s", path)
			}
			customer := out.(*backend.Customer)
			customer.CPF = "12345678901"
			customer.Email = "maria@example.com"
			customer.Phone = "11999999999"
			customer.MonthlyIncome = 5000
			customer.Address = "Rua A"
			return nil
		},
		postFunc:  func(context.Context, string, string, any, any) error { return nil },
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	documentGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, out any) error {
			if path != "/internal/proposals/prop_123/documents/analyze" {
				t.Fatalf("unexpected document path %s", path)
			}
			result := out.(*backend.AnalysisResult)
			*result = backend.AnalysisResult{ProposalID: "prop_123", AnalysisType: "document", Result: "approved", Provider: "doc", Score: 700}
			return nil
		},
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	creditGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, out any) error {
			if path != "/internal/proposals/prop_123/credit-analysis" {
				t.Fatalf("unexpected credit path %s", path)
			}
			result := out.(*backend.AnalysisResult)
			*result = backend.AnalysisResult{ProposalID: "prop_123", AnalysisType: "credit", Result: "approved", Provider: "credit", Score: 710}
			return nil
		},
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	fraudGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, out any) error {
			if path != "/internal/proposals/prop_123/fraud-analysis" {
				t.Fatalf("unexpected fraud path %s", path)
			}
			result := out.(*backend.AnalysisResult)
			*result = backend.AnalysisResult{ProposalID: "prop_123", AnalysisType: "fraud", Result: "approved", Provider: "fraud", Score: 120}
			return nil
		},
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	notificationPostCount := 0
	notificationGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, _ any) error {
			if path != "/internal/proposals/prop_123/notifications" {
				t.Fatalf("unexpected notification path %s", path)
			}
			notificationPostCount++
			return nil
		},
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	srv := NewServer(
		proposalGateway,
		customerGateway,
		documentGateway,
		creditGateway,
		fraudGateway,
		notificationGateway,
		queue.NewMemoryQueue(1),
		0,
		2,
		nil,
	)

	err := srv.processJob(context.Background(), queue.Job{
		ProposalID:    "prop_123",
		CorrelationID: "corr_test",
	})
	if err != nil {
		t.Fatalf("process job: %v", err)
	}
	if proposalPostCount != 3 {
		t.Fatalf("expected 3 analysis result writes, got %d", proposalPostCount)
	}
	if proposalPatchCount != 4 {
		t.Fatalf("expected 4 status transitions, got %d", proposalPatchCount)
	}
	if notificationPostCount != 4 {
		t.Fatalf("expected 4 notifications, got %d", notificationPostCount)
	}
}

func TestListDeadLettersReturnsQueuedFailures(t *testing.T) {
	workflowQueue := queue.NewMemoryQueue(2)
	if err := workflowQueue.DeadLetter(context.Background(), queue.Job{
		ProposalID:    "prop_123",
		CorrelationID: "corr_123",
		Attempt:       2,
		LastError:     "credit timeout",
	}); err != nil {
		t.Fatalf("dead letter: %v", err)
	}

	srv := NewServer(
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		workflowQueue,
		0,
		2,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/internal/dlq", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var response deadLetterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Count != 1 {
		t.Fatalf("expected 1 dead letter, got %d", response.Count)
	}
	if len(response.Jobs) != 1 || response.Jobs[0].ProposalID != "prop_123" {
		t.Fatal("expected DLQ payload with proposal prop_123")
	}
}

func TestReprocessDeadLettersMovesJobBackToQueue(t *testing.T) {
	workflowQueue := queue.NewMemoryQueue(2)
	if err := workflowQueue.DeadLetter(context.Background(), queue.Job{
		ProposalID:    "prop_123",
		CorrelationID: "corr_123",
		Attempt:       2,
		LastError:     "credit timeout",
	}); err != nil {
		t.Fatalf("dead letter: %v", err)
	}

	srv := NewServer(
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		stubGateway{},
		workflowQueue,
		0,
		2,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/internal/dlq/reprocess", nil)
	resp := httptest.NewRecorder()
	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	var response reprocessDLQResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ReprocessedCount != 1 {
		t.Fatalf("expected 1 reprocessed job, got %d", response.ReprocessedCount)
	}

	if depth, err := workflowQueue.DeadLetterLength(context.Background()); err != nil || depth != 0 {
		t.Fatalf("expected empty DLQ, got depth=%d err=%v", depth, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	job, err := workflowQueue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue reprocessed job: %v", err)
	}
	if job.ProposalID != "prop_123" {
		t.Fatalf("expected proposal id prop_123, got %s", job.ProposalID)
	}
	if job.Attempt != 0 || job.LastError != "" {
		t.Fatalf("expected reset job state, got attempt=%d last_error=%q", job.Attempt, job.LastError)
	}
}
