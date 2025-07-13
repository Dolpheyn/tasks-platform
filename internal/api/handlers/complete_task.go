package handlers

import (
	"context"
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/platform/taskmanager"
)

func HandleCompleteTask(ctx context.Context, req *dto.CompleteRequest, taskManager *taskmanager.TaskManager) error {
	log.Printf("[HandleCompleteTask] got request. taskID=%s", req.TaskID)
	err := taskManager.SignalTaskResultSuccess(req.TaskID)
	if err != nil {
		return err
	}

	return nil
}
