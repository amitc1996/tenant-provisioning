package repository

import (
	"context"
	"errors"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/jackc/pgx/v5"
)

type InboxEvent struct {
	ID     string
	TaskID string
	Status domain.TaskStatus
}

func (q *Queries) CreateInboxEvent(
	ctx context.Context,
	event InboxEvent,
) (bool, error) {

	query := `
		INSERT INTO inbox_events (
			id,
			task_id,
			status
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`

	result, err := q.db.Exec(
		ctx,
		query,
		event.ID,
		event.TaskID,
		event.Status,
	)

	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

func (q *Queries) GetPendingTerminalEvent(
	ctx context.Context,
	taskID string,
) (*InboxEvent, error) {

	query := `
		SELECT
			id,
			task_id::text,
			status
		FROM inbox_events
		WHERE task_id = $1
		  AND processed_at IS NULL
		  AND status IN ('done', 'failed')
		ORDER BY received_order
		LIMIT 1
	`

	var event InboxEvent

	err := q.db.QueryRow(
		ctx,
		query,
		taskID,
	).Scan(
		&event.ID,
		&event.TaskID,
		&event.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (q *Queries) MarkInboxProcessed(
	ctx context.Context,
	eventID string,
) error {

	query := `
		UPDATE inbox_events
		SET processed_at = NOW()
		WHERE id = $1
		  AND processed_at IS NULL
	`

	_, err := q.db.Exec(ctx, query, eventID)

	return err
}

func (q *Queries) MarkPendingInboxProcessed(
	ctx context.Context,
	taskID string,
) error {

	query := `
		UPDATE inbox_events
		SET processed_at = NOW()
		WHERE task_id = $1
		  AND processed_at IS NULL
	`

	_, err := q.db.Exec(ctx, query, taskID)

	return err
}
