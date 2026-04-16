package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"github.com/giovaniif/e-commerce/stock/domain/item"
)

var (
	ErrItemNotFound      = errors.New("item not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type ItemRepositoryPostgres struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewItemRepositoryPostgres(db *sql.DB, rdb *redis.Client) *ItemRepositoryPostgres {
	return &ItemRepositoryPostgres{db: db, rdb: rdb}
}

// SeedStockCounters initialises Redis counters and price cache from the items table.
// Called once at startup so the atomic DECRBY path has a baseline and GetItem never hits Postgres.
func (r *ItemRepositoryPostgres) SeedStockCounters(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `SELECT id, price, initial_stock FROM items`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int32
		var price float64
		var initialStock int64
		if err := rows.Scan(&id, &price, &initialStock); err != nil {
			return err
		}
		stockKey := fmt.Sprintf("stock:item:%d", id)
		if err := r.rdb.Set(ctx, stockKey, initialStock, 0).Err(); err != nil {
			return fmt.Errorf("seed item %d stock: %w", id, err)
		}
		priceKey := fmt.Sprintf("stock:item:price:%d", id)
		if err := r.rdb.Set(ctx, priceKey, price, 0).Err(); err != nil {
			return fmt.Errorf("seed item %d price: %w", id, err)
		}
	}
	return nil
}

func (r *ItemRepositoryPostgres) GetItem(itemId int32) (*item.Item, error) {
	ctx := context.Background()
	priceKey := fmt.Sprintf("stock:item:price:%d", itemId)
	priceStr, err := r.rdb.Get(ctx, priceKey).Result()
	if err == nil {
		var price float64
		if _, scanErr := fmt.Sscanf(priceStr, "%f", &price); scanErr == nil {
			return &item.Item{Id: itemId, Price: price}, nil
		}
	}
	// Fallback to Postgres if cache miss (e.g. unknown item).
	row := r.db.QueryRow(`SELECT id, price, initial_stock FROM items WHERE id = $1`, itemId)
	var it item.Item
	if err := row.Scan(&it.Id, &it.Price, &it.InitialStock); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	return &it, nil
}

func (r *ItemRepositoryPostgres) Reserve(reservationItem *item.Item, quantity int32) (*item.Reservation, error) {
	ctx := context.Background()
	key := fmt.Sprintf("stock:item:%d", reservationItem.Id)

	// Atomically decrement — no row lock, no queue, sub-millisecond.
	remaining, err := r.rdb.DecrBy(ctx, key, int64(quantity)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis decr stock: %w", err)
	}
	if remaining < 0 {
		// Not enough stock — compensate and reject.
		r.rdb.IncrBy(ctx, key, int64(quantity))
		return nil, ErrInsufficientStock
	}

	// Postgres is now append-only — no transaction or FOR UPDATE needed.
	var reservationId int64
	if err := r.db.QueryRow(`SELECT nextval('reservation_id_seq')`).Scan(&reservationId); err != nil {
		r.rdb.IncrBy(ctx, key, int64(quantity))
		return nil, fmt.Errorf("next reservation id: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO stock_events (reservation_id, item_id, event_type, quantity)
		VALUES ($1, $2, 'reserved', $3)
	`, reservationId, reservationItem.Id, quantity)
	if err != nil {
		r.rdb.IncrBy(ctx, key, int64(quantity))
		return nil, fmt.Errorf("insert reserved event: %w", err)
	}

	return &item.Reservation{
		Id:       int32(reservationId),
		TotalFee: float64(quantity) * reservationItem.Price,
		Quantity: quantity,
		ItemId:   reservationItem.Id,
		Status:   "reserved",
	}, nil
}

func (r *ItemRepositoryPostgres) ReleaseReservation(reservationId int32) error {
	ctx := context.Background()

	var quantity int32
	var itemId int32
	err := r.db.QueryRow(`
		SELECT quantity, item_id FROM stock_events
		WHERE reservation_id = $1 AND event_type = 'reserved'
	`, reservationId).Scan(&quantity, &itemId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("reservation not found")
		}
		return fmt.Errorf("get reservation: %w", err)
	}

	// Return stock to Redis before writing the event.
	key := fmt.Sprintf("stock:item:%d", itemId)
	r.rdb.IncrBy(ctx, key, int64(quantity))

	_, err = r.db.Exec(`
		INSERT INTO stock_events (reservation_id, item_id, event_type, quantity)
		VALUES ($1, $2, 'released', $3)
	`, reservationId, itemId, quantity)
	return err
}

func (r *ItemRepositoryPostgres) CompleteReservation(reservationId int32) error {
	// Stock was already decremented on reserve and stays consumed — no Redis change.
	// Write the audit event asynchronously to keep the critical path fast.
	go func() {
		var quantity int32
		var itemId int32
		err := r.db.QueryRow(`
			SELECT quantity, item_id FROM stock_events
			WHERE reservation_id = $1 AND event_type = 'reserved'
		`, reservationId).Scan(&quantity, &itemId)
		if err != nil {
			return
		}
		r.db.Exec(`
			INSERT INTO stock_events (reservation_id, item_id, event_type, quantity)
			VALUES ($1, $2, 'completed', $3)
		`, reservationId, itemId, quantity)
	}()
	return nil
}
