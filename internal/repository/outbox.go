package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type OutboxEvent struct {
	TaskID   string
	Payload  []byte
	Attempts int
}

func (q *Queries) CreateOutboxEvent(
	ctx context.Context,
	taskID string,
) error {

	query := `
		INSERT INTO outbox_events (
			task_id,
			payload
		)
		SELECT
			id,
			jsonb_build_object(
				'id', id,
				'tenant_id', tenant_id,
				'type', type,
				'status', status
			)
		FROM tasks
		WHERE id = $1
	`

	result, err := q.db.Exec(ctx, query, taskID)

	if err != nil {
		return err
	}

	if result.RowsAffected() != 1 {
		return errors.New("outbox task not found")
	}

	return nil
}

func (q *Queries) GetNextOutboxEvent(ctx context.Context) (*OutboxEvent, error) {

	query := `
		SELECT
			task_id::text,
			payload,
			attempts
		FROM outbox_events
		WHERE published_at IS NULL
		  AND next_attempt_at <= NOW()
		ORDER BY next_attempt_at, created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var event OutboxEvent

	err := q.db.QueryRow(ctx, query).Scan(
		&event.TaskID,
		&event.Payload,
		&event.Attempts,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (q *Queries) MarkOutboxPublished(ctx context.Context, taskID string) error {
	query := `
		UPDATE outbox_events
		SET published_at = NOW()
		WHERE task_id = $1
		  AND published_at IS NULL
	`
	_, err := q.db.Exec(ctx, query, taskID)

	return err
}

func (q *Queries) RetryOutboxEvent(
	ctx context.Context,
	taskID string,
	delaySeconds int,
) error {

	query := `
		UPDATE outbox_events
		SET
			attempts = attempts + 1,
			next_attempt_at = NOW() +
				($2::integer * INTERVAL '1 second')
		WHERE task_id = $1
		  AND published_at IS NULL
	`

	_, err := q.db.Exec(
		ctx,
		query,
		taskID,
		delaySeconds,
	)

	return err
}
