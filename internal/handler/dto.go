package handler

import (
	"time"

	"github.com/amitc1996/tenant-provisioning/internal/application"
	"github.com/amitc1996/tenant-provisioning/internal/domain"
)

type CreateTenantRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type UpdateTenantRequest struct {
	Slug    *string `json:"slug"`
	Name    *string `json:"name"`
	Version int     `json:"version"`
}

type TenantResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TaskResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type MutationResponse struct {
	Tenant TenantResponse `json:"tenant"`
	Task   TaskResponse   `json:"task"`
}

func toTenantResponse(t *domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Slug:      t.Slug,
		Name:      t.Name,
		Status:    string(t.Status),
		Version:   t.Version,
		CreatedAt: formatTime(t.CreatedAt),
		UpdatedAt: formatTime(t.UpdatedAt),
	}
}

func toTaskResponse(t *domain.Task) TaskResponse {
	return TaskResponse{
		ID:        t.ID,
		TenantID:  t.TenantID,
		Type:      string(t.Type),
		Status:    string(t.Status),
		CreatedAt: formatTime(t.CreatedAt),
		UpdatedAt: formatTime(t.UpdatedAt),
	}
}

func toMutationResponse(
	result *application.MutationResult,
) MutationResponse {
	return MutationResponse{
		Tenant: toTenantResponse(result.Tenant),
		Task:   toTaskResponse(result.Task),
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(
		"2006-01-02T15:04:05.000Z",
	)
}
