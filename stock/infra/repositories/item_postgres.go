package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type ItemRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewItemRepositoryPostgres(pool *pgxpool.Pool) *ItemRepositoryPostgres {
	return &ItemRepositoryPostgres{pool: pool}
}

func (r *ItemRepositoryPostgres) GetItem(itemId int32) (*item.Item, error) {
	ctx := context.Background()

	var it item.Item
	err := r.pool.QueryRow(ctx,
		`SELECT id, price, initial_stock FROM items WHERE id = $1`, itemId).
		Scan(&it.Id, &it.Price, &it.InitialStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, item_id, total_fee, quantity, status FROM reservations WHERE item_id = $1`, itemId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var res item.Reservation
		if err := rows.Scan(&res.Id, &res.ItemId, &res.TotalFee, &res.Quantity, &res.Status); err != nil {
			return nil, err
		}
		it.Reservations = append(it.Reservations, res)
	}

	return &it, nil
}

func (r *ItemRepositoryPostgres) Reserve(reservationItem *item.Item, quantity int32) (*item.Reservation, error) {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var initialStock int32
	err = tx.QueryRow(ctx,
		`SELECT initial_stock FROM items WHERE id = $1 FOR UPDATE`, reservationItem.Id).
		Scan(&initialStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	var reserved int32
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0) FROM reservations WHERE item_id = $1 AND status != 'canceled'`,
		reservationItem.Id).Scan(&reserved)
	if err != nil {
		return nil, err
	}

	if initialStock-reserved < quantity {
		return nil, ErrInsufficientStock
	}

	totalFee := float64(quantity) * reservationItem.Price
	var reservation item.Reservation
	err = tx.QueryRow(ctx,
		`INSERT INTO reservations (item_id, total_fee, quantity, status)
		 VALUES ($1, $2, $3, 'reserved')
		 RETURNING id, item_id, total_fee, quantity, status`,
		reservationItem.Id, totalFee, quantity).
		Scan(&reservation.Id, &reservation.ItemId, &reservation.TotalFee, &reservation.Quantity, &reservation.Status)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &reservation, nil
}

func (r *ItemRepositoryPostgres) ReleaseReservation(reservationId int32) error {
	ctx := context.Background()
	result, err := r.pool.Exec(ctx,
		`UPDATE reservations SET status = 'canceled' WHERE id = $1`, reservationId)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("reservation not found")
	}
	return nil
}

func (r *ItemRepositoryPostgres) CompleteReservation(reservationId int32) error {
	ctx := context.Background()
	result, err := r.pool.Exec(ctx,
		`UPDATE reservations SET status = 'completed' WHERE id = $1`, reservationId)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		fmt.Printf("reservation %d not found", reservationId)
		return errors.New("reservation not found")
	}
	return nil
}
