package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/amitc1996/tenant-provisioning/internal/domain"
)

func (s *Service) GetTask(
	ctx context.Context,
	id string,
) (*domain.Task, error) {

	taskID, err := uuid.Parse(id)

	if err != nil {
		return nil, domain.ErrTaskNotFound
	}

	return s.repo.GetTaskByID(
		ctx,
		taskID.String(),
	)
}

func (s *Service) ListTasks(ctx context.Context, limit int, offset int) ([]domain.Task, error) {

	if limit < 1 || limit > 100 || offset < 0 {
		return nil, domain.ErrInvalidPagination
	}

	return s.repo.ListTasks(
		ctx,
		limit,
		offset,
	)
}
