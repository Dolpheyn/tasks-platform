package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	pfm "github.com/dolpheyn/tasks-platform/pkg"
)

func HandleEnqueueTask(_ context.Context, req *dto.EnqueueTaskRequest, asynqClient *asynq.Client) (*dto.EnqueueTaskResponse, error) {
	platformTaskID := uuid.New().String()

	platformTask := &pfm.PlatformTask{
		ID:       platformTaskID,
		TypeName: req.JobType,
		Payload:  req.Payload,
	}

	task, err := platformTask.ToAsynqTask()
	if err != nil {
		return nil, err
	}

	if _, err := asynqClient.Enqueue(task, asynq.Retention(time.Minute*10)); err != nil {
		return nil, err
	}

	res := &dto.EnqueueTaskResponse{
		ID:     platformTaskID,
		Status: "enqueued",
	}
	return res, nil
}
