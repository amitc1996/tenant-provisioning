package domain

import (
	"regexp"
	"strings"
	"time"
)

type TenantStatus string

const (
	TenantStatusProvisioning TenantStatus = "provisioning"
	TenantStatusActive       TenantStatus = "active"
	TenantStatusUpdating     TenantStatus = "updating"
	TenantStatusDestroying   TenantStatus = "destroying"
	TenantStatusDestroyed    TenantStatus = "destroyed"
	TenantStatusFailed       TenantStatus = "failed"
)

type Tenant struct {
	ID        string
	Slug      string
	Name      string
	Status    TenantStatus
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

var slugPattern = regexp.MustCompile(
	`^[a-z][-a-z0-9]{1,26}[a-z0-9]$`,
)

func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return ErrInvalidSlug
	}

	return nil
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	return nil
}

func NewTenant(id string, slug string, name string, now time.Time) (*Tenant, error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}

	if err := ValidateName(name); err != nil {
		return nil, err
	}

	now = normalizeTime(now)

	return &Tenant{
		ID:        id,
		Slug:      slug,
		Name:      name,
		Status:    TenantStatusProvisioning,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func normalizeTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Millisecond)
}

func (t *Tenant) CanUpdate() bool {
	return t.Status == TenantStatusActive
}

func (t *Tenant) CanDestroy() bool {
	return t.Status == TenantStatusActive ||
		t.Status == TenantStatusFailed
}

func (t *Tenant) BeginUpdate(expectedVersion int, now time.Time) error {

	if t.Version != expectedVersion {
		return ErrTenantVersionConflict
	}

	if !t.CanUpdate() {
		return ErrTenantUpdateNotAllowed
	}

	t.changeStatus(TenantStatusUpdating, now)

	return nil
}

func (t *Tenant) BeginDestroy(now time.Time) error {
	if !t.CanDestroy() {
		return ErrTenantUpdateNotAllowed
	}

	t.changeStatus(TenantStatusDestroying, now)

	return nil
}

func (t *Tenant) changeStatus(status TenantStatus, now time.Time) {
	t.Status = status
	t.Version++
	t.UpdatedAt = normalizeTime(now)
}

func (t *Tenant) ApplyTaskOutcome(task Task, now time.Time) error {
	if task.TenantID != t.ID {
		return ErrInvalidTenantTransition
	}

	if !task.IsTerminal() {
		return ErrInvalidTenantTransition
	}

	var expectedStatus TenantStatus
	var successStatus TenantStatus

	switch task.Type {

	case TaskTypeDeploy:
		expectedStatus = TenantStatusProvisioning
		successStatus = TenantStatusActive

	case TaskTypeUpdate:
		expectedStatus = TenantStatusUpdating
		successStatus = TenantStatusActive

	case TaskTypeDestroy:
		expectedStatus = TenantStatusDestroying
		successStatus = TenantStatusDestroyed

	default:
		return ErrInvalidTaskType
	}

	if t.Status != expectedStatus {
		return ErrInvalidTenantTransition
	}

	if task.Status == TaskStatusFailed {
		t.changeStatus(TenantStatusFailed, now)

		return nil
	}

	t.changeStatus(successStatus, now)

	return nil
}
