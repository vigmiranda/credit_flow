package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var ErrCustomerNotFound = errors.New("customer not found")

type Customer struct {
	ID            string    `json:"customer_id"`
	ProposalID    string    `json:"proposal_id"`
	FullName      string    `json:"full_name"`
	CPF           string    `json:"cpf"`
	BirthDate     string    `json:"birth_date"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	MonthlyIncome float64   `json:"monthly_income"`
	Address       string    `json:"address,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewCustomer(proposalID string, input CustomerInput, now time.Time) Customer {
	return Customer{
		ID:            "cus_" + randomToken(12),
		ProposalID:    proposalID,
		FullName:      strings.TrimSpace(input.FullName),
		CPF:           onlyDigits(input.CPF),
		BirthDate:     strings.TrimSpace(input.BirthDate),
		Email:         strings.TrimSpace(input.Email),
		Phone:         strings.TrimSpace(input.Phone),
		MonthlyIncome: input.MonthlyIncome,
		Address:       strings.TrimSpace(input.Address),
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
}

func (c Customer) WithUpdatedData(input CustomerInput, now time.Time) Customer {
	c.FullName = strings.TrimSpace(input.FullName)
	c.CPF = onlyDigits(input.CPF)
	c.BirthDate = strings.TrimSpace(input.BirthDate)
	c.Email = strings.TrimSpace(input.Email)
	c.Phone = strings.TrimSpace(input.Phone)
	c.MonthlyIncome = input.MonthlyIncome
	c.Address = strings.TrimSpace(input.Address)
	c.UpdatedAt = now.UTC()
	return c
}

type CustomerInput struct {
	FullName      string
	CPF           string
	BirthDate     string
	Email         string
	Phone         string
	MonthlyIncome float64
	Address       string
}

func (i CustomerInput) Validate() map[string]any {
	var fields []string

	if strings.TrimSpace(i.FullName) == "" {
		fields = append(fields, "full_name")
	}
	if len(onlyDigits(i.CPF)) != 11 {
		fields = append(fields, "cpf")
	}
	if strings.TrimSpace(i.BirthDate) == "" {
		fields = append(fields, "birth_date")
	}
	if !strings.Contains(strings.TrimSpace(i.Email), "@") {
		fields = append(fields, "email")
	}
	if strings.TrimSpace(i.Phone) == "" {
		fields = append(fields, "phone")
	}
	if i.MonthlyIncome <= 0 {
		fields = append(fields, "monthly_income")
	}

	if len(fields) == 0 {
		return nil
	}

	return map[string]any{"invalid_fields": fields}
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func randomToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}
