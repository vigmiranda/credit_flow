package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var ErrProposalNotFound = errors.New("proposal not found")

const (
	StatusCreated                    = "created"
	StatusCustomerDataPending        = "customer_data_pending"
	StatusDocumentsPending           = "documents_pending"
	StatusDocumentsReceived          = "documents_received"
	StatusDocumentAnalysisInProgress = "document_analysis_in_progress"
	StatusCreditAnalysisInProgress   = "credit_analysis_in_progress"
	StatusFraudAnalysisInProgress    = "fraud_analysis_in_progress"
	StatusManualReview               = "manual_review"
	StatusApproved                   = "approved"
	StatusRejected                   = "rejected"
	StatusAwaitingAdditionalDocs     = "awaiting_additional_documents"
)

type Proposal struct {
	ID            string    `json:"proposal_id"`
	Protocol      string    `json:"protocol"`
	Status        string    `json:"status"`
	CorrelationID string    `json:"correlation_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewProposal(correlationID string, now time.Time) Proposal {
	return Proposal{
		ID:            "prop_" + randomToken(12),
		Protocol:      "P-" + now.UTC().Format("20060102150405"),
		Status:        StatusCreated,
		CorrelationID: correlationID,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusCreated,
		StatusCustomerDataPending,
		StatusDocumentsPending,
		StatusDocumentsReceived,
		StatusDocumentAnalysisInProgress,
		StatusCreditAnalysisInProgress,
		StatusFraudAnalysisInProgress,
		StatusManualReview,
		StatusApproved,
		StatusRejected,
		StatusAwaitingAdditionalDocs:
		return true
	default:
		return false
	}
}

func randomToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}
