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
		AnalysisType: "fraud",
		Provider:     "mock-fraud-engine",
		Score:        140,
		Reason:       "risco dentro do esperado na simulacao",
		CreatedAt:    now.UTC(),
	}

	switch {
	case strings.Contains(strings.ToLower(customer.Email), "fraude"), hasRepeatedDigits(customer.CPF):
		result.Result = "rejected"
		result.Score = 960
		result.Reason = "indicio forte de fraude na simulacao"
	case strings.HasSuffix(onlyDigits(customer.Phone), "0000"), strings.Contains(strings.ToLower(customer.Address), "manual"):
		result.Result = "manual_review"
		result.Score = 640
		result.Reason = "caso encaminhado para revisao manual"
	default:
		result.Result = "approved"
	}

	return result
}

func hasRepeatedDigits(value string) bool {
	digits := onlyDigits(value)
	if len(digits) < 11 {
		return false
	}

	first := digits[0]
	for i := 1; i < len(digits); i++ {
		if digits[i] != first {
			return false
		}
	}

	return true
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
