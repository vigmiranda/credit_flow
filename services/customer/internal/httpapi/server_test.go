package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"creditflow/services/customer/internal/domain"
)

type stubStore struct {
	items map[string]domain.Customer
}

func newStubStore() *stubStore {
	return &stubStore{items: map[string]domain.Customer{}}
}

func (s *stubStore) UpsertByProposalID(_ context.Context, customer domain.Customer) (domain.Customer, error) {
	if current, ok := s.items[customer.ProposalID]; ok {
		customer = current.WithUpdatedData(domain.CustomerInput{
			FullName:      customer.FullName,
			CPF:           customer.CPF,
			BirthDate:     customer.BirthDate,
			Email:         customer.Email,
			Phone:         customer.Phone,
			MonthlyIncome: customer.MonthlyIncome,
			Address:       customer.Address,
		}, time.Now().UTC())
	}

	s.items[customer.ProposalID] = customer
	return customer, nil
}

func (s *stubStore) GetByProposalID(_ context.Context, proposalID string) (domain.Customer, error) {
	customer, ok := s.items[proposalID]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	return customer, nil
}

func TestUpsertCustomer(t *testing.T) {
	srv := NewServer(newStubStore())
	body := bytes.NewBufferString(`{"full_name":"Maria Silva","cpf":"12345678901","birth_date":"1990-01-01","email":"maria@example.com","phone":"11999999999","monthly_income":4500}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/customer", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, resp.Code)
	}

	var customer domain.Customer
	if err := json.NewDecoder(resp.Body).Decode(&customer); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if customer.ProposalID != "prop_123" {
		t.Fatalf("expected proposal id prop_123, got %s", customer.ProposalID)
	}
}

func TestUpsertCustomerRejectsInvalidCPF(t *testing.T) {
	srv := NewServer(newStubStore())
	body := bytes.NewBufferString(`{"full_name":"Maria Silva","cpf":"123","birth_date":"1990-01-01","email":"maria@example.com","phone":"11999999999","monthly_income":4500}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/proposals/prop_123/customer", body)
	resp := httptest.NewRecorder()

	srv.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
}
