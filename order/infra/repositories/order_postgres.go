package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreatePendingTx inserts a new pending order inside the caller's transaction.
// Returns the generated order UUID, or empty string if the idempotency_key already exists.
func (r *OrderRepository) CreatePendingTx(ctx context.Context, tx *sql.Tx, idempotencyKey string, itemId int32, quantity int32) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		INSERT INTO orders (idempotency_key, item_id, quantity, status)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`, idempotencyKey, itemId, quantity).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // already exists, idempotent
		}
		return "", fmt.Errorf("insert order: %w", err)
	}
	return id, nil
}

// UpdateStatus updates the order status for the given idempotency_key, only if current status is 'pending'.
func (r *OrderRepository) UpdateStatus(ctx context.Context, idempotencyKey string, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = $1, updated_at = NOW()
		WHERE idempotency_key = $2 AND status = 'pending'
	`, status, idempotencyKey)
	return err
}
