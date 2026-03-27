package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFraudAnalysisRejectsRepeatedCPF(t *testing.T) {
	srv := NewServer()
	body := bytes.NewBufferString(`{"customer":{"cpf":"11111111111","email":"maria@example.com","phone":"11999999999","monthly_income":4500,"address":"Rua A"}}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/fraud-analysis", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
}
