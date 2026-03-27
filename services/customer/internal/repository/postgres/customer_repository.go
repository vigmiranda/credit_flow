package postgres

import (
	"context"
	"database/sql"
	"errors"

	"creditflow/services/customer/internal/domain"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) EnsureSchema(ctx context.Context) error {
	const query = `
	CREATE TABLE IF NOT EXISTS customers (
		id TEXT PRIMARY KEY,
		proposal_id TEXT NOT NULL UNIQUE,
		full_name TEXT NOT NULL,
		cpf TEXT NOT NULL,
		birth_date TEXT NOT NULL,
		email TEXT NOT NULL,
		phone TEXT NOT NULL,
		monthly_income DOUBLE PRECISION NOT NULL,
		address TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	);
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *CustomerRepository) UpsertByProposalID(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	const query = `
	INSERT INTO customers (
		id,
		proposal_id,
		full_name,
		cpf,
		birth_date,
		email,
		phone,
		monthly_income,
		address,
		created_at,
		updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (proposal_id) DO UPDATE SET
		full_name = EXCLUDED.full_name,
		cpf = EXCLUDED.cpf,
		birth_date = EXCLUDED.birth_date,
		email = EXCLUDED.email,
		phone = EXCLUDED.phone,
		monthly_income = EXCLUDED.monthly_income,
		address = EXCLUDED.address,
		updated_at = EXCLUDED.updated_at
	RETURNING id, proposal_id, full_name, cpf, birth_date, email, phone, monthly_income, address, created_at, updated_at;
	`

	var saved domain.Customer
	err := r.db.QueryRowContext(
		ctx,
		query,
		customer.ID,
		customer.ProposalID,
		customer.FullName,
		customer.CPF,
		customer.BirthDate,
		customer.Email,
		customer.Phone,
		customer.MonthlyIncome,
		customer.Address,
		customer.CreatedAt,
		customer.UpdatedAt,
	).Scan(
		&saved.ID,
		&saved.ProposalID,
		&saved.FullName,
		&saved.CPF,
		&saved.BirthDate,
		&saved.Email,
		&saved.Phone,
		&saved.MonthlyIncome,
		&saved.Address,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	return saved, err
}

func (r *CustomerRepository) GetByProposalID(ctx context.Context, proposalID string) (domain.Customer, error) {
	const query = `
	SELECT id, proposal_id, full_name, cpf, birth_date, email, phone, monthly_income, address, created_at, updated_at
	FROM customers
	WHERE proposal_id = $1;
	`

	var customer domain.Customer
	err := r.db.QueryRowContext(ctx, query, proposalID).Scan(
		&customer.ID,
		&customer.ProposalID,
		&customer.FullName,
		&customer.CPF,
		&customer.BirthDate,
		&customer.Email,
		&customer.Phone,
		&customer.MonthlyIncome,
		&customer.Address,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, err
	}

	return customer, nil
}
