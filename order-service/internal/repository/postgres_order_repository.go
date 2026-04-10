package repository

import (
	"context"
	"database/sql"
	"errors"
	"order-service/internal/domain"
)

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) EnsureSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
		id TEXT PRIMARY KEY,
		customer_id TEXT NOT NULL,
		item_name TEXT NOT NULL,
		amount BIGINT NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		idempotency_key TEXT UNIQUE
	);`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO orders (id, customer_id, item_name, amount, status, created_at, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		order.ID,
		order.CustomerID,
		order.ItemName,
		order.Amount,
		order.Status,
		order.CreatedAt,
		nullIfEmpty(order.IdempotencyKey),
	)
	return err
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE id = $1`,
		id,
	)

	var order domain.Order
	if err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
		&order.IdempotencyKey,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *PostgresOrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at, COALESCE(idempotency_key, '')
		 FROM orders WHERE idempotency_key = $1`,
		key,
	)

	var order domain.Order
	if err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
		&order.IdempotencyKey,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	res, err := r.db.ExecContext(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}