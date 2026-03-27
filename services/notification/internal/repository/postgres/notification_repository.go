package postgres

import (
	"context"
	"database/sql"

	"creditflow/services/notification/internal/domain"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) EnsureSchema(ctx context.Context) error {
	const query = `
	CREATE TABLE IF NOT EXISTS notifications (
		id TEXT PRIMARY KEY,
		proposal_id TEXT NOT NULL,
		channel TEXT NOT NULL,
		template TEXT NOT NULL,
		recipient TEXT NOT NULL,
		message TEXT NOT NULL,
		status TEXT NOT NULL,
		trigger_status TEXT NOT NULL,
		sent_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	);
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *NotificationRepository) Create(ctx context.Context, notification domain.Notification) error {
	const query = `
	INSERT INTO notifications (
		id, proposal_id, channel, template, recipient, message, status, trigger_status, sent_at, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		notification.ID,
		notification.ProposalID,
		notification.Channel,
		notification.Template,
		notification.Recipient,
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
	SELECT id, proposal_id, channel, template, recipient, message, status, trigger_status, sent_at, created_at
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
		notifications = append(notifications, notification)
	}

	return notifications, rows.Err()
}
