package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type StatusHistoryEntry struct {
	ID         string    `json:"status_history_id"`
	ProposalID string    `json:"proposal_id"`
	Status     string    `json:"status"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewStatusHistoryEntry(proposalID, status, source string, now time.Time) StatusHistoryEntry {
	return StatusHistoryEntry{
		ID:         "hst_" + randomStatusHistoryToken(12),
		ProposalID: proposalID,
		Status:     status,
		Source:     source,
		CreatedAt:  now.UTC(),
	}
}

func randomStatusHistoryToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}
