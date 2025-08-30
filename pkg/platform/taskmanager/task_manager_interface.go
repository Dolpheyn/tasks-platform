package taskmanager

import (
	"context"

	pfm "github.com/dolpheyn/tasks-platform/pkg/platform"
	"github.com/hibiken/asynq"
)

type TaskManagerInterface interface {
	ProcessTask(ctx context.Context, task *asynq.Task) error
	TryConsumeTask(ctx context.Context, taskType string, clientID string) (*pfm.PlatformTask, error)
	SignalTaskResultSuccess(platformTaskID string) error
	SignalTaskResultFailure(platformTaskID string, errMessage string) error
	RecordHeartbeat(taskID string, clientID string) error
}
