package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
)

type CreateTenantInput struct {
	Slug string
	Name string
}

func (s *Service) CreateTenant(ctx context.Context, input CreateTenantInput) (*MutationResult, error) {

	now := time.Now().UTC()

	tenantID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	tenant, err := domain.NewTenant(
		tenantID.String(),
		input.Slug,
		input.Name,
		now,
	)

	if err != nil {
		return nil, err
	}

	task, err := newTask(
		tenant.ID,
		domain.TaskTypeDeploy,
		now,
	)

	if err != nil {
		return nil, err
	}

	err = s.repo.WithinTx(
		ctx,
		func(q *repository.Queries) error {

			if err := q.CreateTenant(ctx, tenant); err != nil {
				return err
			}

			return persistTask(ctx, q, task)
		},
	)
	if err != nil {
		return nil, err
	}

	return &MutationResult{
		Tenant: tenant,
		Task:   task,
	}, nil
}

func newTask(tenantID string, taskType domain.TaskType, now time.Time) (*domain.Task, error) {

	id, err := uuid.NewRandom()

	if err != nil {
		return nil, err
	}

	return domain.NewTask(
		id.String(),
		tenantID,
		taskType,
		now,
	)
}

func (s *Service) GetTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrTenantNotFound
	}
	return s.repo.GetTenantByID(
		ctx,
		tenantID.String(),
	)
}

func (s *Service) ListTenants(ctx context.Context, limit int, offset int) ([]domain.Tenant, error) {

	if limit < 1 || limit > 100 || offset < 0 {
		return nil, domain.ErrInvalidPagination
	}

	return s.repo.ListTenants(
		ctx,
		limit,
		offset,
	)
}

type UpdateTenantInput struct {
	Name    *string
	Slug    *string
	Version int
}

func (s *Service) UpdateTenant(ctx context.Context, id string, input UpdateTenantInput) (*MutationResult, error) {
	tenantID, err := uuid.Parse(id)

	if err != nil {
		return nil, domain.ErrTenantNotFound
	}

	if input.Version < 1 {
		return nil, domain.ErrInvalidVersion
	}

	if input.Name == nil && input.Slug == nil {
		return nil, domain.ErrEmptyTenantUpdate
	}

	var result *MutationResult

	err = s.repo.WithinTx(
		ctx,
		func(q *repository.Queries) error {

			tenant, err := q.GetTenantForUpdate(
				ctx,
				tenantID.String(),
			)

			if err != nil {
				return err
			}

			oldVersion := tenant.Version
			oldStatus := tenant.Status

			now := time.Now().UTC()

			if err := tenant.BeginUpdate(
				input.Version,
				now,
			); err != nil {
				return err
			}

			if input.Name != nil {

				if err := domain.ValidateName(
					*input.Name,
				); err != nil {
					return err
				}

				tenant.Name = *input.Name
			}

			if input.Slug != nil {

				if err := domain.ValidateSlug(
					*input.Slug,
				); err != nil {
					return err
				}

				tenant.Slug = *input.Slug
			}

			task, err := newTask(
				tenant.ID,
				domain.TaskTypeUpdate,
				now,
			)

			if err != nil {
				return err
			}

			if err := q.UpdateTenant(
				ctx,
				tenant,
				oldVersion,
				oldStatus,
			); err != nil {
				return err
			}

			if err := persistTask(
				ctx,
				q,
				task,
			); err != nil {
				return err
			}

			result = &MutationResult{
				Tenant: tenant,
				Task:   task,
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) DeleteTenant(ctx context.Context, id string) (*MutationResult, error) {
	tenantID, err := uuid.Parse(id)

	if err != nil {
		return nil, domain.ErrTenantNotFound
	}

	var result *MutationResult

	err = s.repo.WithinTx(
		ctx,
		func(q *repository.Queries) error {

			tenant, err := q.GetTenantForUpdate(
				ctx,
				tenantID.String(),
			)

			if err != nil {
				return err
			}

			oldVersion := tenant.Version
			oldStatus := tenant.Status

			now := time.Now().UTC()

			if err := tenant.BeginDestroy(now); err != nil {
				return err
			}

			task, err := newTask(
				tenant.ID,
				domain.TaskTypeDestroy,
				now,
			)

			if err != nil {
				return err
			}

			if err := q.UpdateTenant(
				ctx,
				tenant,
				oldVersion,
				oldStatus,
			); err != nil {
				return err
			}

			if err := persistTask(
				ctx,
				q,
				task,
			); err != nil {
				return err
			}

			result = &MutationResult{
				Tenant: tenant,
				Task:   task,
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func persistTask(ctx context.Context, q *repository.Queries, task *domain.Task) error {

	if err := q.CreateTask(ctx, task); err != nil {
		return err
	}

	return q.CreateOutboxEvent(ctx, task.ID)
}
