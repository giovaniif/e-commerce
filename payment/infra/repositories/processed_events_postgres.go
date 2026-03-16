package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type ProcessedEventsRepository struct {
	db *sql.DB
}

func NewProcessedEventsRepository(db *sql.DB) *ProcessedEventsRepository {
	return &ProcessedEventsRepository{db: db}
}

func (r *ProcessedEventsRepository) IsProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`, eventID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *ProcessedEventsRepository) MarkProcessedTx(ctx context.Context, tx *sql.Tx, eventID uuid.UUID) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID,
	)
	return err
}

// ParseEventID parses a string UUID, returning an error if invalid.
func ParseEventID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, errors.New("invalid event_id: " + s)
	}
	return id, nil
}
