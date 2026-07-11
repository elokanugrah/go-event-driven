package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/elokanugrah/go-event-driven/internal/domain"
	"github.com/elokanugrah/go-event-driven/internal/usecase"
)

var _ usecase.OutboxRepository = (*PostgresOutboxRepository)(nil)

type PostgresOutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{db: db}
}

// getQuerier extracts a transaction from the context if it exists,
// otherwise it returns the base database connection.
func (r *PostgresOutboxRepository) getQuerier(ctx context.Context) querier {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return r.db
}

// Save stores a new outbox event in the database.
func (r *PostgresOutboxRepository) Save(ctx context.Context, event *domain.OutboxEvent) error {
	query := `INSERT INTO outbox_events (event_type, payload, status, created_at)
			  VALUES ($1, $2, $3, $4)
			  RETURNING id`

	now := time.Now()
	err := r.getQuerier(ctx).QueryRowContext(ctx, query,
		event.EventType,
		event.Payload,
		event.Status,
		now,
	).Scan(&event.ID)

	if err != nil {
		return fmt.Errorf("error saving outbox event: %w", err)
	}

	event.CreatedAt = now
	return nil
}

// FindPending retrieves pending outbox events. It uses FOR UPDATE SKIP LOCKED
// to allow concurrent processors to safely work on different outbox rows.
func (r *PostgresOutboxRepository) FindPending(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	query := `SELECT id, event_type, payload, status, created_at, processed_at
			  FROM outbox_events
			  WHERE status = 'pending'
			  ORDER BY id ASC
			  LIMIT $1
			  FOR UPDATE SKIP LOCKED`

	rows, err := r.getQuerier(ctx).QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		var processedAt sql.NullTime
		err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.Payload,
			&e.Status,
			&e.CreatedAt,
			&processedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning outbox event row: %w", err)
		}
		if processedAt.Valid {
			e.ProcessedAt = &processedAt.Time
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during outbox events rows iteration: %w", err)
	}

	return events, nil
}

// UpdateStatus changes the status of an outbox event and records the timestamp.
func (r *PostgresOutboxRepository) UpdateStatus(ctx context.Context, id int64, status domain.OutboxStatus) error {
	query := `UPDATE outbox_events
			  SET status = $1, processed_at = $2
			  WHERE id = $3`

	var processedAt *time.Time
	if status == domain.OutboxStatusProcessed {
		now := time.Now()
		processedAt = &now
	}

	result, err := r.getQuerier(ctx).ExecContext(ctx, query, status, processedAt, id)
	if err != nil {
		return fmt.Errorf("error updating outbox event status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected for outbox update: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("outbox event not found for update (id: %d)", id)
	}

	return nil
}
