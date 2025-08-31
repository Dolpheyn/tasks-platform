package handlers

import (
	"context"
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/taskmanager"
)

func HandleFailTask(ctx context.Context, req *dto.FailRequest, taskManager taskmanager.TaskManager) error {
	log.Printf("[HandleFailTask] got request. taskID=%s", req.TaskID)
	err := taskManager.SignalTaskResultFailure(req.TaskID, req.Error)
	if err != nil {
		return err
	}

	return nil
}
