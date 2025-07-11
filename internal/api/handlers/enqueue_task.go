package handlers

import (
	"context"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

func HandleEnqueueTask(_ context.Context, req *dto.EnqueueTaskRequest, asynqClient *asynq.Client) (*dto.EnqueueTaskResponse, error) {
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

	res := &dto.EnqueueTaskResponse{
		ID:     platformTaskID,
		Status: "enqueued",
	}
	return res, nil
}
