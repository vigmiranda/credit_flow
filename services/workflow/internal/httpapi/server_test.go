package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/workflow/internal/backend"
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

func TestWorkflowApprovesProposal(t *testing.T) {
	proposalPostCount := 0
	proposalGateway := stubGateway{
		getFunc: func(context.Context, string, string, any) error { return nil },
		postFunc: func(_ context.Context, path, _ string, _ any, _ any) error {
			if path != "/internal/proposals/prop_123/analysis-results" {
				t.Fatalf("unexpected proposal post path %s", path)
			}
			proposalPostCount++
			return nil
		},
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
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

	notificationGateway := stubGateway{
		getFunc:   func(context.Context, string, string, any) error { return nil },
		postFunc:  func(context.Context, string, string, any, any) error { return nil },
		patchFunc: func(context.Context, string, string, any, any) error { return nil },
	}

	srv := NewServer(proposalGateway, customerGateway, documentGateway, creditGateway, fraudGateway, notificationGateway, 0*time.Millisecond)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/run-analyses", nil)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}
	if proposalPostCount != 3 {
		t.Fatalf("expected 3 analysis result writes, got %d", proposalPostCount)
	}
}
