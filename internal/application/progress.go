package application

import (
	"context"
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
)

type ProgressInput struct {
	EventID string
	TaskID  string
	Status  domain.TaskStatus
}

func (s *Service) ApplyTaskProgress(
	ctx context.Context,
	input ProgressInput,
) error {

	return s.repo.WithinTx(
		ctx,
		func(q *repository.Queries) error {

			// Lock task to serialize concurrent progress updates.
			task, err := q.GetTaskForUpdate(
				ctx,
				input.TaskID,
			)

			if err != nil {
				return err
			}

			// Persist the event for idempotency.
			inserted, err := q.CreateInboxEvent(
				ctx,
				repository.InboxEvent{
					ID:     input.EventID,
					TaskID: input.TaskID,
					Status: input.Status,
				},
			)

			if err != nil {
				return err
			}

			// Duplicate event: no state mutation.
			if !inserted {
				return nil
			}

			// Never modify terminal tasks.
			if task.IsTerminal() {
				return q.MarkPendingInboxProcessed(
					ctx,
					task.ID,
				)
			}

			// Apply in_progress events.
			if input.Status == domain.TaskStatusInProgress {

				if err := applyProgress(
					ctx,
					q,
					task,
					input.Status,
				); err != nil {
					return err
				}

				if err := q.MarkInboxProcessed(
					ctx,
					input.EventID,
				); err != nil {
					// return err
				}
			}

			// Terminal events received before in_progress
			// remain safely pending in the inbox.
			if task.Status == domain.TaskStatusAccepted {
				return nil
			}

			// Apply any pending terminal event.
			return applyPendingTerminal(
				ctx,
				q,
				task,
			)
		},
	)
}

func applyProgress(
	ctx context.Context,
	q *repository.Queries,
	task *domain.Task,
	status domain.TaskStatus,
) error {

	oldStatus := task.Status

	now := time.Now().UTC()

	changed, err := task.ApplyProgress(
		status,
		now,
	)

	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	if err := q.UpdateTaskStatus(
		ctx,
		task,
		oldStatus,
	); err != nil {
		return err
	}

	// Non-terminal progress does not mutate the tenant.
	if !task.IsTerminal() {
		return nil
	}

	// Apply terminal task outcome to the tenant.
	tenant, err := q.GetTenantForUpdate(
		ctx,
		task.TenantID,
	)

	if err != nil {
		return err
	}

	oldVersion := tenant.Version
	oldTenantStatus := tenant.Status

	if err := tenant.ApplyTaskOutcome(
		*task,
		now,
	); err != nil {
		return err
	}

	return q.UpdateTenant(
		ctx,
		tenant,
		oldVersion,
		oldTenantStatus,
	)
}

func applyPendingTerminal(
	ctx context.Context,
	q *repository.Queries,
	task *domain.Task,
) error {

	if task.Status != domain.TaskStatusInProgress {
		return nil
	}

	event, err := q.GetPendingTerminalEvent(
		ctx,
		task.ID,
	)

	if err != nil {
		return err
	}

	if event == nil {
		return nil
	}

	if err := applyProgress(
		ctx,
		q,
		task,
		event.Status,
	); err != nil {
		return err
	}

	return q.MarkPendingInboxProcessed(
		ctx,
		task.ID,
	)
}
