package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"creditflow/services/document/internal/domain"
)

type DocumentRepository struct {
	db *sql.DB
}

func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

func (r *DocumentRepository) EnsureSchema(ctx context.Context) error {
	const query = `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		proposal_id TEXT NOT NULL,
		document_type TEXT NOT NULL,
		file_name TEXT NOT NULL,
		content_type TEXT NOT NULL,
		file_key TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL,
		upload_url TEXT NOT NULL,
		storage_url TEXT NOT NULL DEFAULT '',
		uploaded_at TIMESTAMPTZ NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	`

	if _, err := r.db.ExecContext(ctx, query); err != nil {
		return err
	}

	const alterQuery = `
	ALTER TABLE documents
		ADD COLUMN IF NOT EXISTS storage_url TEXT NOT NULL DEFAULT '';
	`

	_, err := r.db.ExecContext(ctx, alterQuery)
	return err
}

func (r *DocumentRepository) Create(ctx context.Context, document domain.Document) error {
	const query = `
	INSERT INTO documents (
		id,
		proposal_id,
		document_type,
		file_name,
		content_type,
		file_key,
		status,
		upload_url,
		storage_url,
		uploaded_at,
		created_at,
		updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		document.ID,
		document.ProposalID,
		document.Type,
		document.FileName,
		document.ContentType,
		document.FileKey,
		document.Status,
		document.UploadURL,
		document.StorageURL,
		document.UploadedAt,
		document.CreatedAt,
		document.UpdatedAt,
	)
	return err
}

func (r *DocumentRepository) ListByProposalID(ctx context.Context, proposalID string) ([]domain.Document, error) {
	const query = `
	SELECT id, proposal_id, document_type, file_name, content_type, file_key, status, upload_url, storage_url, uploaded_at, created_at, updated_at
	FROM documents
	WHERE proposal_id = $1
	ORDER BY created_at ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []domain.Document
	for rows.Next() {
		var document domain.Document
		var uploadedAt sql.NullTime

		if err := rows.Scan(
			&document.ID,
			&document.ProposalID,
			&document.Type,
			&document.FileName,
			&document.ContentType,
			&document.FileKey,
			&document.Status,
			&document.UploadURL,
			&document.StorageURL,
			&uploadedAt,
			&document.CreatedAt,
			&document.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if uploadedAt.Valid {
			timestamp := uploadedAt.Time
			document.UploadedAt = &timestamp
		}

		documents = append(documents, document)
	}

	return documents, rows.Err()
}

func (r *DocumentRepository) GetByProposalIDAndDocumentID(ctx context.Context, proposalID, documentID string) (domain.Document, error) {
	const query = `
	SELECT id, proposal_id, document_type, file_name, content_type, file_key, status, upload_url, storage_url, uploaded_at, created_at, updated_at
	FROM documents
	WHERE proposal_id = $1 AND id = $2;
	`

	var document domain.Document
	var uploadedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, proposalID, documentID).Scan(
		&document.ID,
		&document.ProposalID,
		&document.Type,
		&document.FileName,
		&document.ContentType,
		&document.FileKey,
		&document.Status,
		&document.UploadURL,
		&document.StorageURL,
		&uploadedAt,
		&document.CreatedAt,
		&document.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, err
	}

	if uploadedAt.Valid {
		timestamp := uploadedAt.Time
		document.UploadedAt = &timestamp
	}

	return document, nil
}

func (r *DocumentRepository) MarkUploaded(ctx context.Context, proposalID, documentID string, uploadedAt time.Time) (domain.Document, error) {
	const query = `
	UPDATE documents
	SET status = $3, uploaded_at = $4, updated_at = $4
	WHERE proposal_id = $1 AND id = $2
	RETURNING id, proposal_id, document_type, file_name, content_type, file_key, status, upload_url, storage_url, uploaded_at, created_at, updated_at;
	`

	var document domain.Document
	var uploaded sql.NullTime
	err := r.db.QueryRowContext(ctx, query, proposalID, documentID, domain.StatusUploaded, uploadedAt).Scan(
		&document.ID,
		&document.ProposalID,
		&document.Type,
		&document.FileName,
		&document.ContentType,
		&document.FileKey,
		&document.Status,
		&document.UploadURL,
		&document.StorageURL,
		&uploaded,
		&document.CreatedAt,
		&document.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, err
	}

	if uploaded.Valid {
		timestamp := uploaded.Time
		document.UploadedAt = &timestamp
	}

	return document, nil
}
