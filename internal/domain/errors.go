package domain

import "errors"

var (
	ErrTenantNotFound = errors.New("tenant_not_found")

	ErrTenantAlreadyExists = errors.New("tenant_already_exists")

	ErrTenantUpdateNotAllowed = errors.New("tenant_update_not_allowed")

	ErrTenantVersionConflict = errors.New("tenant_version_conflict")

	ErrTaskNotFound = errors.New("task_not_found")

	ErrInvalidSlug = errors.New("invalid_slug")

	ErrInvalidName = errors.New("invalid_name")

	ErrInvalidTaskType = errors.New("invalid_task_type")

	ErrInvalidTenantTransition = errors.New("invalid_tenant_transition")

	ErrInvalidTaskTransition = errors.New("invalid_task_transition")

	ErrTaskProgressOutOfOrder = errors.New("task_progress_out_of_order")
)
