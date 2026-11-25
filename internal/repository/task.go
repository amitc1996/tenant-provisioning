package repository

import (
	"context"
	"errors"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/jackc/pgx/v5"
)

func (q *Queries) CreateTask(
	ctx context.Context,
	task *domain.Task,
) error {

	query := `
		INSERT INTO tasks (
			id,
			tenant_id,
			type,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := q.db.Exec(
		ctx,
		query,
		task.ID,
		task.TenantID,
		task.Type,
		task.Status,
		task.CreatedAt,
		task.UpdatedAt,
	)

	return err
}

func (q *Queries) GetTaskByID(
	ctx context.Context,
	id string,
) (*domain.Task, error) {

	query := `
		SELECT
			id::text,
			tenant_id::text,
			type,
			status,
			created_at,
			updated_at
		FROM tasks
		WHERE id = $1
	`

	return scanTask(q.db.QueryRow(ctx, query, id))
}

func scanTask(row scanner) (*domain.Task, error) {
	var task domain.Task

	err := row.Scan(
		&task.ID,
		&task.TenantID,
		&task.Type,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTaskNotFound
	}

	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (q *Queries) ListTasks(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.Task, error) {

	query := `
		SELECT
			id::text,
			tenant_id::text,
			type,
			status,
			created_at,
			updated_at
		FROM tasks
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.db.Query(
		ctx,
		query,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tasks := make([]domain.Task, 0)

	for rows.Next() {
		task, err := scanTask(rows)

		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (q *Queries) GetTaskForUpdate(ctx context.Context, id string) (*domain.Task, error) {
	query := `
		SELECT
			id::text,
			tenant_id::text,
			type,
			status,
			created_at,
			updated_at
		FROM tasks
		WHERE id = $1
		FOR UPDATE
	`

	return scanTask(q.db.QueryRow(ctx, query, id))
}

func (q *Queries) UpdateTaskStatus(
	ctx context.Context,
	task *domain.Task,
	expectedStatus domain.TaskStatus,
) error {

	query := `
		UPDATE tasks
		SET
			status = $2,
			updated_at = $3
		WHERE
			id = $1
			AND status = $4
	`
	result, err := q.db.Exec(
		ctx,
		query,
		task.ID,
		task.Status,
		task.UpdatedAt,
		expectedStatus,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrInvalidTaskTransition
	}

	return nil
}
