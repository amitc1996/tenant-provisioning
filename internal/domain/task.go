package domain

import "time"

type TaskType string

const (
	TaskTypeDeploy  TaskType = "deploy"
	TaskTypeUpdate  TaskType = "update"
	TaskTypeDestroy TaskType = "destroy"
)

type TaskStatus string

const (
	TaskStatusAccepted   TaskStatus = "accepted"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
)

type Task struct {
	ID        string
	TenantID  string
	Type      TaskType
	Status    TaskStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewTask(id string, tenantID string, taskType TaskType, now time.Time) (*Task, error) {
	switch taskType {
	case TaskTypeDeploy, TaskTypeUpdate, TaskTypeDestroy:
	default:
		return nil, ErrInvalidTaskType
	}

	now = normalizeTime(now)

	return &Task{
		ID:        id,
		TenantID:  tenantID,
		Type:      taskType,
		Status:    TaskStatusAccepted,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusDone ||
		t.Status == TaskStatusFailed
}

func (t *Task) IsOpen() bool {
	return t.Status == TaskStatusAccepted ||
		t.Status == TaskStatusInProgress
}

func (t *Task) ApplyProgress(nextStatus TaskStatus, now time.Time) (bool, error) {
	switch nextStatus {
	case TaskStatusInProgress,
		TaskStatusDone,
		TaskStatusFailed:
	default:
		return false, ErrInvalidTaskTransition
	}

	if t.IsTerminal() {
		return false, nil
	}

	if t.Status == nextStatus {
		return false, nil
	}

	switch t.Status {
	case TaskStatusAccepted:

		if nextStatus != TaskStatusInProgress {
			return false, ErrTaskProgressOutOfOrder
		}

	case TaskStatusInProgress:

		if nextStatus != TaskStatusDone &&
			nextStatus != TaskStatusFailed {
			return false, ErrInvalidTaskTransition
		}

	default:
		return false, ErrInvalidTaskTransition
	}

	t.Status = nextStatus
	t.UpdatedAt = normalizeTime(now)

	return true, nil
}
