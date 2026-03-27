package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	AnalysisTypeDocument = "document"
	AnalysisTypeCredit   = "credit"
	AnalysisTypeFraud    = "fraud"
)

const (
	AnalysisResultApproved     = "approved"
	AnalysisResultRejected     = "rejected"
	AnalysisResultManualReview = "manual_review"
	AnalysisResultPendingDocs  = "awaiting_additional_documents"
)

type AnalysisResult struct {
	ID           string    `json:"analysis_id"`
	ProposalID   string    `json:"proposal_id"`
	AnalysisType string    `json:"analysis_type"`
	Result       string    `json:"result"`
	Provider     string    `json:"provider"`
	Score        int       `json:"score"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewAnalysisResult(proposalID, analysisType, result, provider, reason string, score int, now time.Time) AnalysisResult {
	return AnalysisResult{
		ID:           "anl_" + randomAnalysisToken(12),
		ProposalID:   proposalID,
		AnalysisType: analysisType,
		Result:       result,
		Provider:     provider,
		Score:        score,
		Reason:       reason,
		CreatedAt:    now.UTC(),
	}
}

func IsValidAnalysisType(value string) bool {
	switch value {
	case AnalysisTypeDocument, AnalysisTypeCredit, AnalysisTypeFraud:
		return true
	default:
		return false
	}
}

func IsValidAnalysisResult(value string) bool {
	switch value {
	case AnalysisResultApproved, AnalysisResultRejected, AnalysisResultManualReview, AnalysisResultPendingDocs:
		return true
	default:
		return false
	}
}

func randomAnalysisToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}
