package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	ChannelEmail = "email"
	StatusSent   = "sent"
)

type Notification struct {
	ID            string    `json:"notification_id"`
	ProposalID    string    `json:"proposal_id"`
	Channel       string    `json:"channel"`
	Template      string    `json:"template"`
	Recipient     string    `json:"recipient"`
	Message       string    `json:"message"`
	Status        string    `json:"status"`
	TriggerStatus string    `json:"trigger_status"`
	SentAt        time.Time `json:"sent_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewNotification(proposalID, channel, template, recipient, message, triggerStatus string, now time.Time) Notification {
	return Notification{
		ID:            "ntf_" + randomNotificationToken(12),
		ProposalID:    proposalID,
		Channel:       channel,
		Template:      template,
		Recipient:     recipient,
		Message:       message,
		Status:        StatusSent,
		TriggerStatus: triggerStatus,
		SentAt:        now.UTC(),
		CreatedAt:     now.UTC(),
	}
}

func randomNotificationToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}
