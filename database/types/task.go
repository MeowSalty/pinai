package types

import "time"

// 模型批量任务类型。
const (
	ModelBatchTaskTypeAdd    = "model.batch_add"
	ModelBatchTaskTypeUpdate = "model.batch_update"
	ModelBatchTaskTypeDelete = "model.batch_delete"
)

// 模型批量任务状态。
const (
	ModelBatchTaskStatusPending   = "pending"
	ModelBatchTaskStatusRunning   = "running"
	ModelBatchTaskStatusSucceeded = "succeeded"
	ModelBatchTaskStatusFailed    = "failed"
)

// ModelBatchTask 表示模型批量异步任务。
type ModelBatchTask struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Type         string     `gorm:"index:idx_model_batch_tasks_status_type,priority:2;size:64;not null" json:"type"`
	Status       string     `gorm:"index:idx_model_batch_tasks_status_type,priority:1;size:32;not null" json:"status"`
	PlatformID   uint       `gorm:"index;not null" json:"platform_id"`
	Payload      string     `gorm:"type:text;not null" json:"payload"`
	Result       string     `gorm:"type:text" json:"result,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
