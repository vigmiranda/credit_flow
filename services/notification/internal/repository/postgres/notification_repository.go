package postgres

import (
	"context"
	"database/sql"

	"creditflow/services/notification/internal/domain"
	"creditflow/services/notification/internal/security"
)

type NotificationRepository struct {
	db     *sql.DB
	crypto *security.Cipher
}

func NewNotificationRepository(db *sql.DB, crypto *security.Cipher) *NotificationRepository {
	return &NotificationRepository{db: db, crypto: crypto}
}

func (r *NotificationRepository) EnsureSchema(ctx context.Context) error {
	const createTableQuery = `
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		proposal_id TEXT NOT NULL,
		channel TEXT NOT NULL,
		template TEXT NOT NULL,
		recipient TEXT NOT NULL,
		recipient_masked TEXT NOT NULL DEFAULT '',
		recipient_encrypted BYTEA,
		message TEXT NOT NULL,
		status TEXT NOT NULL,
		trigger_status TEXT NOT NULL,
		sent_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	);
	`

	if _, err := r.db.ExecContext(ctx, createTableQuery); err != nil {
		return err
	}

	const alterTableQuery = `
	ALTER TABLE notifications
		ADD COLUMN IF NOT EXISTS recipient_masked TEXT NOT NULL DEFAULT '',
		ADD COLUMN IF NOT EXISTS recipient_encrypted BYTEA;
	`

	_, err := r.db.ExecContext(ctx, alterTableQuery)
	return err
}

func (r *NotificationRepository) Create(ctx context.Context, notification domain.Notification) error {
	encryptedRecipient, err := r.crypto.Encrypt(notification.Recipient)
	if err != nil {
		return err
	}
	maskedRecipient := security.MaskRecipient(notification.Channel, notification.Recipient)

	const query = `
	INSERT INTO notifications (
		id, proposal_id, channel, template, recipient, recipient_masked, recipient_encrypted, message, status, trigger_status, sent_at, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		notification.ID,
		notification.ProposalID,
		notification.Channel,
		notification.Template,
		maskedRecipient,
		maskedRecipient,
		encryptedRecipient,
		notification.Message,
		notification.Status,
		notification.TriggerStatus,
		notification.SentAt,
		notification.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) ListByProposalID(ctx context.Context, proposalID string) ([]domain.Notification, error) {
	const query = `
	SELECT id, proposal_id, channel, template, COALESCE(NULLIF(recipient_masked, ''), recipient) AS recipient, message, status, trigger_status, sent_at, created_at
	FROM notifications
	WHERE proposal_id = $1
	ORDER BY created_at ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		var notification domain.Notification
		if err := rows.Scan(
			&notification.ID,
			&notification.ProposalID,
			&notification.Channel,
			&notification.Template,
			&notification.Recipient,
			&notification.Message,
			&notification.Status,
			&notification.TriggerStatus,
			&notification.SentAt,
			&notification.CreatedAt,
		); err != nil {
			return nil, err
		}
		notification.Recipient = security.MaskRecipient(notification.Channel, notification.Recipient)
		notifications = append(notifications, notification)
	}

	return notifications, rows.Err()
}
