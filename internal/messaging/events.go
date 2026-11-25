package messaging

type TaskEvent struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type ProgressEvent struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}
