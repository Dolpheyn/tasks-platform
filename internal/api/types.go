package api

import (
	"encoding/json"
	"time"
)

type Task struct {
	TaskID  string          `json:"task_id"`
	Payload json.RawMessage `json:"payload"`
}

// EnqueueTaskRequest for creating new tasks
type EnqueueTaskRequest struct {
	JobType  string          `json:"job_type" binding:"required"`
	Payload  json.RawMessage `json:"payload" binding:"required"`
	Queue    string          `json:"queue"`
	MaxRetry int             `json:"max_retry"`
	Deadline time.Time       `json:"deadline"`
}

// EnqueueTaskResponse for task creation response
type EnqueueTaskResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// PollTaskRequest for getting tasks
type PollTaskRequest struct {
	WorkerID string `form:"worker_id" binding:"required"`
	TaskType string `form:"task_type" binding:"required"`
}

// PollResponse for getting tasks
type PollResponse struct {
	Task *Task `json:"task"`
}

// HeartbeatRequest for keeping tasks alive
type HeartbeatRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	TaskID   string `json:"task_id" binding:"required"`
}

// CompleteRequest for finishing tasks
type CompleteRequest struct {
	WorkerID string          `json:"worker_id" binding:"required"`
	TaskID   string          `json:"task_id" binding:"required"`
	Result   json.RawMessage `json:"result,omitempty"`
}

// FailRequest for failing tasks
type FailRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	TaskID   string `json:"task_id" binding:"required"`
	Error    string `json:"error" binding:"required"`
	Retry    bool   `json:"retry"`
}
