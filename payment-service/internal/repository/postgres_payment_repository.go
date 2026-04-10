package repository

import (
	"context"
	"database/sql"
	"errors"
	"payment-service/internal/domain"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) EnsureSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS payments (
		id TEXT PRIMARY KEY,
		order_id TEXT UNIQUE NOT NULL,
		transaction_id TEXT NOT NULL,
		amount BIGINT NOT NULL,
		status TEXT NOT NULL
	);`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO payments (id, order_id, transaction_id, amount, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		payment.ID, payment.OrderID, payment.TransactionID, payment.Amount, payment.Status,
	)
	return err
}

func (r *PostgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, transaction_id, amount, status FROM payments WHERE order_id = $1`, orderID,
	)
	var p domain.Payment
	if err := row.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}
