package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewTask(t *testing.T) {
	types := []TaskType{
		TaskTypeDeploy,
		TaskTypeUpdate,
		TaskTypeDestroy,
	}

	for _, taskType := range types {
		t.Run(string(taskType), func(t *testing.T) {
			now := time.Now().UTC().Truncate(time.Millisecond)

			task, err := NewTask(
				"task-id",
				"tenant-id",
				taskType,
				now,
			)

			if err != nil {
				t.Fatal(err)
			}

			if task.Status != TaskStatusAccepted {
				t.Error("expected accepted status")
			}

			if task.Type != taskType ||
				task.ID != "task-id" ||
				task.TenantID != "tenant-id" {
				t.Error("unexpected task fields")
			}

			if !task.CreatedAt.Equal(now) ||
				!task.UpdatedAt.Equal(now) {
				t.Error("unexpected timestamps")
			}

			if !task.IsOpen() {
				t.Error("new task must be open")
			}

			if task.IsTerminal() {
				t.Error("new task must not be terminal")
			}
		})
	}
}

func TestInvalidTaskType(t *testing.T) {
	_, err := NewTask(
		"task-id",
		"tenant-id",
		TaskType("unknown"),
		time.Now().UTC(),
	)

	if !errors.Is(err, ErrInvalidTaskType) {
		t.Fatalf("expected invalid task type, got %v", err)
	}
}

func TestTaskApplyProgress(t *testing.T) {
	tests := []struct {
		name     string
		initial  TaskStatus
		incoming TaskStatus
		expected TaskStatus
		changed  bool
		wantErr  error
	}{
		{
			"start task",
			TaskStatusAccepted,
			TaskStatusInProgress,
			TaskStatusInProgress,
			true,
			nil,
		},
		{
			"complete task",
			TaskStatusInProgress,
			TaskStatusDone,
			TaskStatusDone,
			true,
			nil,
		},
		{
			"fail task",
			TaskStatusInProgress,
			TaskStatusFailed,
			TaskStatusFailed,
			true,
			nil,
		},
		{
			"duplicate in_progress",
			TaskStatusInProgress,
			TaskStatusInProgress,
			TaskStatusInProgress,
			false,
			nil,
		},
		{
			"duplicate done",
			TaskStatusDone,
			TaskStatusDone,
			TaskStatusDone,
			false,
			nil,
		},
		{
			"duplicate failed",
			TaskStatusFailed,
			TaskStatusFailed,
			TaskStatusFailed,
			false,
			nil,
		},
		{
			"stale progress after done",
			TaskStatusDone,
			TaskStatusInProgress,
			TaskStatusDone,
			false,
			nil,
		},
		{
			"stale progress after failed",
			TaskStatusFailed,
			TaskStatusInProgress,
			TaskStatusFailed,
			false,
			nil,
		},
		{
			"failed after done",
			TaskStatusDone,
			TaskStatusFailed,
			TaskStatusDone,
			false,
			nil,
		},
		{
			"done after failed",
			TaskStatusFailed,
			TaskStatusDone,
			TaskStatusFailed,
			false,
			nil,
		},
		{
			"early done",
			TaskStatusAccepted,
			TaskStatusDone,
			TaskStatusAccepted,
			false,
			ErrTaskProgressOutOfOrder,
		},
		{
			"early failed",
			TaskStatusAccepted,
			TaskStatusFailed,
			TaskStatusAccepted,
			false,
			ErrTaskProgressOutOfOrder,
		},
		{
			"invalid status",
			TaskStatusAccepted,
			TaskStatus("unknown"),
			TaskStatusAccepted,
			false,
			ErrInvalidTaskTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{
				Status:    tt.initial,
				UpdatedAt: time.Now().UTC().Add(-time.Hour),
			}

			before := task
			now := time.Now().UTC()

			changed, err := task.ApplyProgress(
				tt.incoming,
				now,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"expected error %v, got %v",
					tt.wantErr,
					err,
				)
			}

			if changed != tt.changed {
				t.Errorf(
					"expected changed=%v, got %v",
					tt.changed,
					changed,
				)
			}

			if task.Status != tt.expected {
				t.Errorf(
					"expected status %s, got %s",
					tt.expected,
					task.Status,
				)
			}

			if !tt.changed && task != before {
				t.Error("ignored or rejected event modified task")
			}

			if tt.changed && !task.UpdatedAt.Equal(
				now.Truncate(time.Millisecond),
			) {
				t.Error("updated timestamp is incorrect")
			}
		})
	}
}
