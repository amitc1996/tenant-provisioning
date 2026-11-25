package application

import (
	"github.com/amitc1996/tenant-provisioning/internal/domain"
	"github.com/amitc1996/tenant-provisioning/internal/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

type MutationResult struct {
	Tenant *domain.Tenant
	Task   *domain.Task
}
