package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		slug  string
		valid bool
	}{
		{"abc", true},
		{"customer-one", true},
		{"tenant123", true},
		{"a--b", true},
		{strings.Repeat("a", 28), true},

		{"", false},
		{"ab", false},
		{"123tenant", false},
		{"Customer", false},
		{"tenant_one", false},
		{"tenant-", false},
		{"-tenant", false},
		{"tenant one", false},
		{strings.Repeat("a", 29), false},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			err := ValidateSlug(tt.slug)

			if tt.valid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.valid &&
				!errors.Is(err, ErrInvalidSlug) {
				t.Fatalf("expected invalid slug, got %v", err)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"Customer One", true},
		{"A", true},
		{"", false},
		{"   ", false},
		{"\t\n", false},
	}

	for _, tt := range tests {
		err := ValidateName(tt.name)

		if tt.valid && err != nil {
			t.Errorf("unexpected error for %q: %v", tt.name, err)
		}

		if !tt.valid && !errors.Is(err, ErrInvalidName) {
			t.Errorf("expected invalid name for %q", tt.name)
		}
	}
}

func TestNewTenant(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)

	tenant, err := NewTenant(
		"tenant-id",
		"customer-one",
		"Customer One",
		now,
	)

	if err != nil {
		t.Fatal(err)
	}

	if tenant.Status != TenantStatusProvisioning {
		t.Errorf("unexpected status: %s", tenant.Status)
	}

	if tenant.Version != 1 {
		t.Errorf("expected version 1, got %d", tenant.Version)
	}

	if tenant.ID != "tenant-id" ||
		tenant.Slug != "customer-one" ||
		tenant.Name != "Customer One" {
		t.Error("tenant fields do not match input")
	}

	if !tenant.CreatedAt.Equal(now) ||
		!tenant.UpdatedAt.Equal(now) {
		t.Error("unexpected timestamps")
	}
}

func TestTenantBeginUpdate(t *testing.T) {
	statuses := []TenantStatus{
		TenantStatusProvisioning,
		TenantStatusActive,
		TenantStatusUpdating,
		TenantStatusDestroying,
		TenantStatusDestroyed,
		TenantStatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			now := time.Now().UTC()

			tenant := Tenant{
				Status:  status,
				Version: 2,
			}

			before := tenant

			err := tenant.BeginUpdate(2, now)

			if status == TenantStatusActive {
				if err != nil {
					t.Fatal(err)
				}

				if tenant.Status != TenantStatusUpdating {
					t.Error("expected updating status")
				}

				if tenant.Version != 3 {
					t.Error("expected version 3")
				}

				if !tenant.UpdatedAt.Equal(
					now.Truncate(time.Millisecond),
				) {
					t.Error("updated timestamp is incorrect")
				}

				return
			}

			if !errors.Is(err, ErrTenantUpdateNotAllowed) {
				t.Fatalf("expected update not allowed, got %v", err)
			}

			if tenant != before {
				t.Error("rejected update modified tenant")
			}
		})
	}
}

func TestTenantVersionConflict(t *testing.T) {
	tenant := Tenant{
		Status:  TenantStatusActive,
		Version: 5,
	}

	before := tenant

	err := tenant.BeginUpdate(
		4,
		time.Now().UTC(),
	)

	if !errors.Is(err, ErrTenantVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}

	if tenant != before {
		t.Error("version conflict modified tenant")
	}
}

func TestTenantBeginDestroy(t *testing.T) {
	statuses := []TenantStatus{
		TenantStatusProvisioning,
		TenantStatusActive,
		TenantStatusUpdating,
		TenantStatusDestroying,
		TenantStatusDestroyed,
		TenantStatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			tenant := Tenant{
				Status:  status,
				Version: 2,
			}

			before := tenant

			err := tenant.BeginDestroy(time.Now().UTC())

			allowed := status == TenantStatusActive ||
				status == TenantStatusFailed

			if allowed {
				if err != nil {
					t.Fatal(err)
				}

				if tenant.Status != TenantStatusDestroying {
					t.Error("expected destroying status")
				}

				if tenant.Version != 3 {
					t.Error("expected version 3")
				}

				return
			}

			if !errors.Is(err, ErrTenantUpdateNotAllowed) {
				t.Fatalf("expected operation not allowed, got %v", err)
			}

			if tenant != before {
				t.Error("rejected operation modified tenant")
			}
		})
	}
}

func TestTenantApplyTaskOutcome(t *testing.T) {
	tests := []struct {
		name     string
		initial  TenantStatus
		taskType TaskType
		outcome  TaskStatus
		expected TenantStatus
	}{
		{
			"deploy success",
			TenantStatusProvisioning,
			TaskTypeDeploy,
			TaskStatusDone,
			TenantStatusActive,
		},
		{
			"deploy failure",
			TenantStatusProvisioning,
			TaskTypeDeploy,
			TaskStatusFailed,
			TenantStatusFailed,
		},
		{
			"update success",
			TenantStatusUpdating,
			TaskTypeUpdate,
			TaskStatusDone,
			TenantStatusActive,
		},
		{
			"update failure",
			TenantStatusUpdating,
			TaskTypeUpdate,
			TaskStatusFailed,
			TenantStatusFailed,
		},
		{
			"destroy success",
			TenantStatusDestroying,
			TaskTypeDestroy,
			TaskStatusDone,
			TenantStatusDestroyed,
		},
		{
			"destroy failure",
			TenantStatusDestroying,
			TaskTypeDestroy,
			TaskStatusFailed,
			TenantStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := Tenant{
				ID:      "tenant-id",
				Status:  tt.initial,
				Version: 3,
			}

			task := Task{
				ID:       "task-id",
				TenantID: tenant.ID,
				Type:     tt.taskType,
				Status:   tt.outcome,
			}

			now := time.Now().UTC()

			err := tenant.ApplyTaskOutcome(task, now)

			if err != nil {
				t.Fatal(err)
			}

			if tenant.Status != tt.expected {
				t.Errorf(
					"expected %s, got %s",
					tt.expected,
					tenant.Status,
				)
			}

			if tenant.Version != 4 {
				t.Errorf(
					"expected version 4, got %d",
					tenant.Version,
				)
			}

			if !tenant.UpdatedAt.Equal(
				now.Truncate(time.Millisecond),
			) {
				t.Error("unexpected updated timestamp")
			}
		})
	}
}

func TestTenantRejectInvalidTaskOutcome(t *testing.T) {
	tests := []struct {
		name string
		task Task
	}{
		{
			name: "wrong tenant",
			task: Task{
				TenantID: "another-tenant",
				Type:     TaskTypeDeploy,
				Status:   TaskStatusDone,
			},
		},
		{
			name: "non-terminal task",
			task: Task{
				TenantID: "tenant-id",
				Type:     TaskTypeDeploy,
				Status:   TaskStatusInProgress,
			},
		},
		{
			name: "incorrect operation",
			task: Task{
				TenantID: "tenant-id",
				Type:     TaskTypeDestroy,
				Status:   TaskStatusDone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := Tenant{
				ID:      "tenant-id",
				Status:  TenantStatusProvisioning,
				Version: 1,
			}

			before := tenant

			err := tenant.ApplyTaskOutcome(
				tt.task,
				time.Now().UTC(),
			)

			if !errors.Is(err, ErrInvalidTenantTransition) {
				t.Fatalf("expected invalid transition, got %v", err)
			}

			if tenant != before {
				t.Error("invalid outcome modified tenant")
			}
		})
	}
}
