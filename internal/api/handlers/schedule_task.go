package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	dto "github.com/dolpheyn/tasks-platform/internal/api/dto"
	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

func HandleScheduleTask(_ context.Context, req *dto.ScheduleTaskRequest, asynqClient *asynq.Client) (*dto.ScheduleTaskResponse, error) {
	platformTaskID := uuid.New().String()
	pt := &platform.PlatformTask{ID: platformTaskID, TypeName: req.JobType, Payload: req.Payload}

	task, err := pt.ToAsynqTask()
	if err != nil {
		return nil, err
	}

	asynqOpts := toAsynqOptions(req)
	asynqOpts = append(asynqOpts, asynq.Retention(time.Minute*10))

	if _, err := asynqClient.Enqueue(task, asynqOpts...); err != nil {
		return nil, err
	}
	return &dto.ScheduleTaskResponse{ID: platformTaskID, Status: "scheduled"}, nil
}

func toAsynqOptions(req *dto.ScheduleTaskRequest) []asynq.Option {
	opts := []asynq.Option{}
	if req.Queue != "" {
		opts = append(opts, asynq.Queue(req.Queue))
	}
	if req.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(req.MaxRetry))
	}
	if req.Deadline != nil {
		opts = append(opts, asynq.Deadline(*req.Deadline))
	}

	// scheduling - validation handled by binding tags
	if req.ProcessAt != nil {
		opts = append(opts, asynq.ProcessAt(req.ProcessAt.Time))
	} else {
		opts = append(opts, asynq.ProcessIn(time.Duration(req.ProcessIn.DurationSecs)*time.Second))
	}

	return opts
}
