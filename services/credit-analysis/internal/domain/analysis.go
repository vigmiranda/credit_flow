package domain

import (
	"strings"
	"time"
)

type Customer struct {
	CPF           string  `json:"cpf"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	MonthlyIncome float64 `json:"monthly_income"`
	Address       string  `json:"address"`
}

type AnalysisResult struct {
	ProposalID   string    `json:"proposal_id"`
	AnalysisType string    `json:"analysis_type"`
	Result       string    `json:"result"`
	Provider     string    `json:"provider"`
	Score        int       `json:"score"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

func Analyze(proposalID string, customer Customer, now time.Time) AnalysisResult {
	result := AnalysisResult{
		ProposalID:   proposalID,
		AnalysisType: "credit",
		Provider:     "mock-credit-engine",
		Score:        720,
		Reason:       "score aprovado na simulacao",
		CreatedAt:    now.UTC(),
	}

	switch {
	case customer.MonthlyIncome < 2000:
		result.Result = "rejected"
		result.Score = 380
		result.Reason = "renda insuficiente para o produto simulado"
	case strings.Contains(strings.ToLower(customer.Email), "manual"), strings.HasSuffix(onlyDigits(customer.CPF), "00"):
		result.Result = "manual_review"
		result.Score = 540
		result.Reason = "analise de credito enviada para revisao manual"
	default:
		result.Result = "approved"
		result.Score = min(900, 500+int(customer.MonthlyIncome/15))
	}

	return result
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
