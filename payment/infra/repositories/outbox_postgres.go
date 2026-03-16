package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
)

type OutboxEvent struct {
	AggregateType string
	AggregateID   string
	Type          string
	Payload       json.RawMessage
	Traceparent   string
}

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) InsertTx(ctx context.Context, tx *sql.Tx, event OutboxEvent) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox (aggregate_type, aggregate_id, type, payload, traceparent)
		VALUES ($1, $2, $3, $4, $5)
	`, event.AggregateType, event.AggregateID, event.Type, event.Payload, event.Traceparent)
	return err
}
