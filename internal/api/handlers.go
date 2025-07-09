package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

func handleEnqueueTask(_ context.Context, req *EnqueueTaskRequest, asynqClient *asynq.Client) (*EnqueueTaskResponse, error) {
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

func handleScheduleTask(_ context.Context, req *ScheduleTaskRequest, asynqClient *asynq.Client) (*ScheduleTaskResponse, error) {
	platformTaskID := uuid.New().String()
	pt := &platform.PlatformTask{ID: platformTaskID, Payload: req.Payload}

	task, err := pt.ToAsynqTask(req.JobType)
	if err != nil {
		return nil, err
	}

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

	if _, err := asynqClient.Enqueue(task, opts...); err != nil {
		return nil, err
	}
	return &ScheduleTaskResponse{ID: platformTaskID, Status: "scheduled"}, nil
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
