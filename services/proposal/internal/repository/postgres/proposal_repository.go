package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"creditflow/services/proposal/internal/domain"
)

type ProposalRepository struct {
	db *sql.DB
}

func NewProposalRepository(db *sql.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

func (r *ProposalRepository) EnsureSchema(ctx context.Context) error {
	const query = `
	CREATE TABLE IF NOT EXISTS proposals (
		id TEXT PRIMARY KEY,
		protocol TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL,
		correlation_id TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);

	CREATE TABLE IF NOT EXISTS proposal_analysis_results (
		id TEXT PRIMARY KEY,
		proposal_id TEXT NOT NULL,
		analysis_type TEXT NOT NULL,
		result TEXT NOT NULL,
		provider TEXT NOT NULL,
		score INTEGER NOT NULL,
		reason TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	);
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *ProposalRepository) Create(ctx context.Context, proposal domain.Proposal) error {
	const query = `
	INSERT INTO proposals (id, protocol, status, correlation_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6);
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		proposal.ID,
		proposal.Protocol,
		proposal.Status,
		proposal.CorrelationID,
		proposal.CreatedAt,
		proposal.UpdatedAt,
	)
	return err
}

func (r *ProposalRepository) GetByID(ctx context.Context, proposalID string) (domain.Proposal, error) {
	const query = `
	SELECT id, protocol, status, correlation_id, created_at, updated_at
	FROM proposals
	WHERE id = $1;
	`

	var proposal domain.Proposal
	err := r.db.QueryRowContext(ctx, query, proposalID).Scan(
		&proposal.ID,
		&proposal.Protocol,
		&proposal.Status,
		&proposal.CorrelationID,
		&proposal.CreatedAt,
		&proposal.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Proposal{}, domain.ErrProposalNotFound
		}

		return domain.Proposal{}, err
	}

	return proposal, nil
}

func (r *ProposalRepository) UpdateStatus(ctx context.Context, proposalID, status string, updatedAt time.Time) (domain.Proposal, error) {
	const query = `
	UPDATE proposals
	SET status = $2, updated_at = $3
	WHERE id = $1
	RETURNING id, protocol, status, correlation_id, created_at, updated_at;
	`

	var proposal domain.Proposal
	err := r.db.QueryRowContext(ctx, query, proposalID, status, updatedAt).Scan(
		&proposal.ID,
		&proposal.Protocol,
		&proposal.Status,
		&proposal.CorrelationID,
		&proposal.CreatedAt,
		&proposal.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Proposal{}, domain.ErrProposalNotFound
		}

		return domain.Proposal{}, err
	}

	return proposal, nil
}

func (r *ProposalRepository) CreateAnalysisResult(ctx context.Context, result domain.AnalysisResult) error {
	const deleteQuery = `
	DELETE FROM proposal_analysis_results
	WHERE proposal_id = $1 AND analysis_type = $2;
	`
	if _, err := r.db.ExecContext(ctx, deleteQuery, result.ProposalID, result.AnalysisType); err != nil {
		return err
	}

	const insertQuery = `
	INSERT INTO proposal_analysis_results (id, proposal_id, analysis_type, result, provider, score, reason, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`

	_, err := r.db.ExecContext(
		ctx,
		insertQuery,
		result.ID,
		result.ProposalID,
		result.AnalysisType,
		result.Result,
		result.Provider,
		result.Score,
		result.Reason,
		result.CreatedAt,
	)
	return err
}

func (r *ProposalRepository) ListAnalysisResults(ctx context.Context, proposalID string) ([]domain.AnalysisResult, error) {
	const query = `
	SELECT id, proposal_id, analysis_type, result, provider, score, reason, created_at
	FROM proposal_analysis_results
	WHERE proposal_id = $1
	ORDER BY created_at ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.AnalysisResult
	for rows.Next() {
		var result domain.AnalysisResult
		if err := rows.Scan(
			&result.ID,
			&result.ProposalID,
			&result.AnalysisType,
			&result.Result,
			&result.Provider,
			&result.Score,
			&result.Reason,
			&result.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}
