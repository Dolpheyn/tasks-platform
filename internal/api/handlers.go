package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

func handleEnqueueTask(ctx context.Context, req *EnqueueTaskRequest, asynqClient *asynq.Client) (*EnqueueTaskResponse, error) {
	platformTaskID := uuid.New().String()

	platformTask := &platform.PlatformTask{
		ID:      platformTaskID,
		Payload: req.Payload,
	}

	task, err := platformTask.ToAsynqTask(req.JobType)
	if err != nil {
		return nil, err
	}

	if _, err := asynqClient.Enqueue(task); err != nil {
		return nil, err
	}

	res := &EnqueueTaskResponse{
		ID:     platformTaskID,
		Status: "enqueued",
	}
	return res, nil
}

func handlePollTask(ctx context.Context, req *PollTaskRequest, taskManager *platform.TaskManager) (*PollResponse, error) {
	platformTask, err := taskManager.TryConsumeTask(ctx, req.TaskType, req.WorkerID)
	if err != nil {
		return nil, err
	}

	res := &PollResponse{
		Task: &Task{
			TaskID:  platformTask.ID,
			Payload: platformTask.Payload,
		},
	}
	return res, nil
}
