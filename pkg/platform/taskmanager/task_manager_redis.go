package taskmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	pfm "github.com/dolpheyn/tasks-platform/pkg/platform"
)

const (
	pendingTasksPrefix = "tasks:pending:"
	activeTasksKey     = "tasks:active"
	heartbeatsPrefix   = "tasks:heartbeats:"
	pickupSignalPrefix = "tasks:signals:pickup:"
	resultSignalPrefix = "tasks:signals:result:"
)

type RedisTaskManager struct {
	redisClient redis.UniversalClient
}

func NewRedisTaskManager(redisClient redis.UniversalClient) *RedisTaskManager {
	return &RedisTaskManager{
		redisClient: redisClient,
	}
}

func (m *RedisTaskManager) ProcessTask(ctx context.Context, task *asynq.Task) error {
	log.Printf("[RedisTaskManager::ProcessTask] registering task with tasktype %s", task.Type())

	platformTask, err := pfm.FromAsynqTask(task)
	if err != nil {
		return err
	}

	if err := m.RegisterTask(platformTask); err != nil {
		return err
	}

	platformTaskID := platformTask.ID

	pickupCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	pickupSignal, err := m.WaitForTaskPickup(pickupCtx, platformTaskID)
	if err != nil {
		return err
	}

	log.Printf("[RedisTaskManager::ProcessTask] task picked up. platformTaskID=%s clientID=%s", platformTaskID, pickupSignal.ClientID)

	resultCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	taskResult, err := m.WaitForTaskResult(resultCtx, platformTaskID)
	if err != nil {
		return err
	}

	if !taskResult.Success {
		return fmt.Errorf("[client] %s", taskResult.Msg)
	}

	return nil
}

func (m *RedisTaskManager) RegisterTask(task *pfm.PlatformTask) error {
	taskType := task.TypeName
	taskID := task.ID

	// Marshal the task
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %v", err)
	}

	// Add the task to the pending list
	pendingTasksKey := pendingTasksPrefix + taskType
	if err := m.redisClient.LPush(context.Background(), pendingTasksKey, taskJSON).Err(); err != nil {
		return fmt.Errorf("failed to add task to pending list: %v", err)
	}

	log.Printf("[RedisTaskManager::RegisterTask] registered task with taskID %s", taskID)
	return nil
}

func (m *RedisTaskManager) TryConsumeTask(ctx context.Context, taskType string, clientID string) (*pfm.PlatformTask, error) {
	// Get a task from the pending list
	pendingTasksKey := pendingTasksPrefix + taskType
	taskJSON, err := m.redisClient.RPop(context.Background(), pendingTasksKey).Bytes()
	if err == redis.Nil {
		return nil, pfm.ErrTaskQueueEmpty
	} else if err != nil {
		return nil, fmt.Errorf("failed to get task from pending list: %v", err)
	}

	// Unmarshal the task
	var task pfm.PlatformTask
	if err := json.Unmarshal(taskJSON, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %v", err)
	}

	taskID := task.ID

	// Add the task to the active hash
	if err := m.redisClient.HSet(context.Background(), activeTasksKey, taskID, taskJSON).Err(); err != nil {
		// If we fail to add to active hash, we should put it back in the pending list
		m.redisClient.LPush(context.Background(), pendingTasksKey, taskJSON)
		return nil, fmt.Errorf("failed to add task to active hash: %v", err)
	}

	// Publish the pickup signal
	pickupSignalKey := pickupSignalPrefix + taskID
	pickupSignal := PickupSignal{ClientID: clientID}
	pickupSignalJSON, err := json.Marshal(pickupSignal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pickup signal: %v", err)
	}
	if err := m.redisClient.Publish(context.Background(), pickupSignalKey, pickupSignalJSON).Err(); err != nil {
		return nil, fmt.Errorf("failed to publish pickup signal: %v", err)
	}

	return &task, nil
}

func (m *RedisTaskManager) SignalTaskResultSuccess(platformTaskID string) error {
	return m.signalTaskResult(platformTaskID, true, "")
}

func (m *RedisTaskManager) SignalTaskResultFailure(platformTaskID string, errMessage string) error {
	return m.signalTaskResult(platformTaskID, false, errMessage)
}

func (m *RedisTaskManager) signalTaskResult(platformTaskID string, success bool, msg string) error {
	resultSignalKey := resultSignalPrefix + platformTaskID
	resultSignal := ResultSignal{Success: success, Msg: msg}
	resultSignalJSON, err := json.Marshal(resultSignal)
	if err != nil {
		return fmt.Errorf("failed to marshal result signal: %v", err)
	}
	if err := m.redisClient.Publish(context.Background(), resultSignalKey, resultSignalJSON).Err(); err != nil {
		return fmt.Errorf("failed to publish result signal: %v", err)
	}
	return nil
}

func (m *RedisTaskManager) WaitForTaskPickup(ctx context.Context, taskID string) (*PickupSignal, error) {
	pickupSignalKey := pickupSignalPrefix + taskID
	pubsub := m.redisClient.Subscribe(context.Background(), pickupSignalKey)
	defer pubsub.Close()

	select {
	case msg := <-pubsub.Channel():
		var pickupSignal PickupSignal
		if err := json.Unmarshal([]byte(msg.Payload), &pickupSignal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pickup signal: %v", err)
		}
		return &pickupSignal, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("deadline exceeded without pickup")
	}
}

func (m *RedisTaskManager) WaitForTaskResult(ctx context.Context, taskID string) (*ResultSignal, error) {
	resultSignalKey := resultSignalPrefix + taskID
	pubsub := m.redisClient.Subscribe(context.Background(), resultSignalKey)
	defer pubsub.Close()

	select {
	case msg := <-pubsub.Channel():
		var resultSignal ResultSignal
		if err := json.Unmarshal([]byte(msg.Payload), &resultSignal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result signal: %v", err)
		}
		return &resultSignal, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("deadline exceeded without result")
	}
}

func (m *RedisTaskManager) UnregisterTask(platformTaskID string) error {
	// Remove from active tasks
	if err := m.redisClient.HDel(context.Background(), activeTasksKey, platformTaskID).Err(); err != nil {
		return fmt.Errorf("failed to remove task from active hash: %v", err)
	}

	// Remove heartbeats
	heartbeatsKey := heartbeatsPrefix + platformTaskID
	if err := m.redisClient.Del(context.Background(), heartbeatsKey).Err(); err != nil {
		return fmt.Errorf("failed to remove heartbeats: %v", err)
	}

	return nil
}

func (m *RedisTaskManager) RecordHeartbeat(taskID string, clientID string) error {
	heartbeatsKey := heartbeatsPrefix + taskID
	now := time.Now().Unix()
	if err := m.redisClient.ZAdd(context.Background(), heartbeatsKey, redis.Z{Score: float64(now), Member: clientID}).Err(); err != nil {
		return fmt.Errorf("failed to record heartbeat: %v", err)
	}
	return nil
}
