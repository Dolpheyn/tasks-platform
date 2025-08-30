package handlers

import (
	"context"
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/platform/taskmanager"
)

func HandlePollTask(ctx context.Context, req *dto.PollTaskRequest, taskManager taskmanager.TaskManagerInterface) (*dto.PollResponse, error) {
	log.Printf("[handlePollTask] got request. taskType=%s", req.TaskType)
	platformTask, err := taskManager.TryConsumeTask(ctx, req.TaskType, req.WorkerID)
	if err != nil {
		return nil, err
	}

	res := &dto.PollResponse{
		Task: &dto.Task{
			TaskID:  platformTask.ID,
			Payload: platformTask.Payload,
		},
	}
	return res, nil
}
