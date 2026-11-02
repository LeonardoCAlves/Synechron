package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OutboxMessage struct {
	ID               string
	EventType        string
	AggregateID      string
	AggregateVersion int64
	Payload          []byte
	Attempts         int
}

type OutboxStats struct {
	Pending          int64
	OldestPendingAge time.Duration
	DeadLettered     int64
}

type OutboxStore struct {
	db *sql.DB
}

func NewOutboxStore(db *sql.DB) *OutboxStore {
	return &OutboxStore{db: db}
}

func (store *OutboxStore) ClaimPending(ctx context.Context, limit int, lease time.Duration) ([]OutboxMessage, error) {
	rows, err := store.db.QueryContext(ctx, `
		WITH pending AS (
			SELECT candidate.id
			FROM outbox_events AS candidate
			WHERE candidate.published_at IS NULL
			  AND candidate.dead_lettered_at IS NULL
			  AND candidate.next_attempt_at <= now()
			  AND (candidate.lease_until IS NULL OR candidate.lease_until <= now())
			  AND NOT EXISTS (
				SELECT 1
				FROM outbox_events AS earlier
				WHERE earlier.aggregate_id = candidate.aggregate_id
				  AND earlier.published_at IS NULL
				  AND (earlier.aggregate_version, earlier.created_at, earlier.id) <
				      (candidate.aggregate_version, candidate.created_at, candidate.id)
			  )
			ORDER BY candidate.created_at, candidate.id
			  LIMIT $1
			  FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events AS event
		SET lease_until = now() + ($2 * interval '1 millisecond')
		FROM pending
		WHERE event.id = pending.id
		RETURNING event.id, event.event_type, event.aggregate_id,
			event.aggregate_version, event.payload, event.attempts`,
		limit, lease.Milliseconds())
	if err != nil {
		return nil, fmt.Errorf("claim pending outbox events: %w", err)
	}
	defer rows.Close()

	messages := make([]OutboxMessage, 0)
	for rows.Next() {
		var message OutboxMessage
		if err := rows.Scan(
			&message.ID, &message.EventType, &message.AggregateID,
			&message.AggregateVersion, &message.Payload, &message.Attempts,
		); err != nil {
			return nil, fmt.Errorf("scan pending outbox event: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending outbox events: %w", err)
	}
	return messages, nil
}

func (store *OutboxStore) MarkPublished(ctx context.Context, id string) error {
	result, err := store.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET published_at = now(), lease_until = NULL, last_error = NULL
		WHERE id = $1 AND published_at IS NULL AND dead_lettered_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	if err := requireOneRow(result); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return nil
}

func (store *OutboxStore) Stats(ctx context.Context) (OutboxStats, error) {
	var stats OutboxStats
	var oldestSeconds float64
	err := store.db.QueryRowContext(ctx, `
		SELECT
			count(*) FILTER (WHERE published_at IS NULL AND dead_lettered_at IS NULL),
			COALESCE(EXTRACT(EPOCH FROM (
				now() - min(created_at) FILTER (WHERE published_at IS NULL AND dead_lettered_at IS NULL)
			)), 0),
			count(*) FILTER (WHERE dead_lettered_at IS NOT NULL AND published_at IS NULL)
		FROM outbox_events`).Scan(&stats.Pending, &oldestSeconds, &stats.DeadLettered)
	if err != nil {
		return OutboxStats{}, fmt.Errorf("read outbox statistics: %w", err)
	}
	stats.OldestPendingAge = time.Duration(oldestSeconds * float64(time.Second))
	return stats, nil
}

func (store *OutboxStore) RecordFailure(ctx context.Context, id, message string, retryAfter time.Duration, maxAttempts int) (bool, error) {
	result, err := store.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
		    last_error = left($2, 1000),
		    next_attempt_at = now() + ($3 * interval '1 millisecond'),
		    lease_until = NULL,
		    dead_lettered_at = CASE WHEN attempts + 1 >= $4 THEN now() ELSE NULL END
		WHERE id = $1 AND published_at IS NULL AND dead_lettered_at IS NULL`,
		id, message, retryAfter.Milliseconds(), maxAttempts)
	if err != nil {
		return false, fmt.Errorf("record outbox delivery failure: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check outbox failure update: %w", err)
	}
	if affected != 1 {
		return false, fmt.Errorf("outbox event %q is no longer pending", id)
	}
	var deadLettered bool
	if err := store.db.QueryRowContext(ctx, `SELECT dead_lettered_at IS NOT NULL FROM outbox_events WHERE id = $1`, id).Scan(&deadLettered); err != nil {
		return false, fmt.Errorf("read outbox dead-letter status: %w", err)
	}
	return deadLettered, nil
}

func requireOneRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("expected one affected row, got %d", affected)
	}
	return nil
}
