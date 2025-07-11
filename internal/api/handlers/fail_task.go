package handlers

import (
	"context"
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

func HandleFailTask(ctx context.Context, req *dto.FailRequest, taskManager *platform.TaskManager) error {
	log.Printf("[HandleFailTask] got request. taskID=%s", req.TaskID)
	err := taskManager.RegisterResultFailure(req.TaskID, req.Error)
	if err != nil {
		return err
	}

	return nil
}
