package taskmanager

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hibiken/asynq"

	pfm "github.com/dolpheyn/tasks-platform/pkg/platform"
)

// InMemoryTaskManager is an in-memory implementation of TaskManagerInterface.
// NOTE: This implementation is not suitable for production use as it's not scalable and resilient.
// It's provided for demonstration and testing purposes.
type InMemoryTaskManager struct {
	mu sync.Mutex

	pendingTasksByType map[string][]*pfm.PlatformTask

	activeTasksByID   map[string]*pfm.PlatformTask
	heartbeatsByID    map[string][]*time.Time
	pickupSignalsByID map[string]chan PickupSignal
	dropSignalsByID   map[string]chan DropSignal
	resultSignalsByID map[string]chan ResultSignal
}

func NewInMemoryTaskManager() *InMemoryTaskManager {
	return &InMemoryTaskManager{
		pendingTasksByType: map[string][]*pfm.PlatformTask{},
		activeTasksByID:    map[string]*pfm.PlatformTask{},
		heartbeatsByID:     map[string][]*time.Time{},
		pickupSignalsByID:  map[string]chan PickupSignal{},
		dropSignalsByID:    map[string]chan DropSignal{},
		resultSignalsByID:  map[string]chan ResultSignal{},
	}
}

type registerTaskRespose struct {
	platformTaskID       string
	pickupSignalListener <-chan PickupSignal
	dropSignalListener   <-chan DropSignal
	resultListener       <-chan ResultSignal
}

func (m *InMemoryTaskManager) registerTask(task *asynq.Task) (*registerTaskRespose, error) {
	taskType := task.Type()
	pickupSignal := make(chan PickupSignal, 1)
	dropSignal := make(chan DropSignal, 1)
	resultSignal := make(chan ResultSignal, 1)

	platformTask, err := pfm.FromAsynqTask(task)
	log.Printf("[registerTask] platformTask=%+v", platformTask)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal platform task: %v", err)
	}

	platformTaskID := platformTask.ID

	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, ok := m.pendingTasksByType[taskType]
	log.Printf("[registerTask] taskType len=%v", len(tasks))
	if !ok {
		tasks = make([]*pfm.PlatformTask, 0, 32)
	}

	tasks = append(tasks, platformTask)

	m.pendingTasksByType[taskType] = tasks
	m.pickupSignalsByID[platformTaskID] = pickupSignal
	m.dropSignalsByID[platformTaskID] = dropSignal
	m.resultSignalsByID[platformTaskID] = resultSignal

	res := &registerTaskRespose{
		platformTaskID:       platformTaskID,
		pickupSignalListener: pickupSignal,
		dropSignalListener:   dropSignal,
		resultListener:       resultSignal,
	}
	return res, nil
}

// ProcessTask takes task from asynq, waits for it to get picked up by a client.
// when signalled that the task is being picked up, it immediately request to asynq for the lease of the task to be extended.
// then it check the heartbeat record, and keep extending the lease for as long as the client is sending heartbeat (with some timeout).
func (m *InMemoryTaskManager) ProcessTask(ctx context.Context, task *asynq.Task) error {
	log.Printf("[TaskManager::ProcessTask] registering task with tasktype %s", task.Type())

	registerTaskRes, err := m.registerTask(task)
	if err != nil {
		return err
	}

	platformTaskID := registerTaskRes.platformTaskID

	pickupSignal, err := m.waitForTaskPickup(registerTaskRes.pickupSignalListener)
	if err != nil {
		return err
	}

	log.Printf("[TaskManager::ProcessTask] task picked up. platformTaskID=%s clientID=%s", platformTaskID, pickupSignal.ClientID)

	return m.waitForTaskResult(platformTaskID, registerTaskRes.resultListener)
}

func (m *InMemoryTaskManager) waitForTaskPickup(pickupSignalListener <-chan PickupSignal) (*PickupSignal, error) {
	// wait for pickup signal with timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	select {
	case pickup := <-pickupSignalListener:
		return &pickup, nil
	case <-timeoutCtx.Done():
		log.Printf("[Taskmanager::waitForTaskPickup] deadline exceeded without pickup")
		return nil, fmt.Errorf("[taskplatform] deadline exceeded without pickup")
	}
}

func (m *InMemoryTaskManager) waitForTaskResult(platformTaskID string, resultListener <-chan ResultSignal) error {
	// task picked up, wait for result with timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		log.Printf("[Taskmanager::waitForTaskResult] worker exceeded deadline")
		m.unregisterTask(platformTaskID)
		return fmt.Errorf("[taskplatform] worker exceeded deadline")
	case taskResult := <-resultListener:
		if !taskResult.Success {
			return fmt.Errorf("[client] %s", taskResult.Msg)
		}
		return nil
	}
}

func (m *InMemoryTaskManager) unregisterTask(platformTaskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activeTasksByID, platformTaskID)
	delete(m.heartbeatsByID, platformTaskID)
	closeChanAndDelete(m.pickupSignalsByID, platformTaskID)
	closeChanAndDelete(m.dropSignalsByID, platformTaskID)
	closeChanAndDelete(m.resultSignalsByID, platformTaskID)

	return nil
}

func (m *InMemoryTaskManager) RecordHeartbeat(taskID string, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	heartbeats, ok := m.heartbeatsByID[taskID]
	if !ok {
		heartbeats = make([]*time.Time, 0, 32)
	}

	now := time.Now()
	heartbeats = append(heartbeats, &now)
	m.heartbeatsByID[taskID] = heartbeats

	return nil
}

func closeChanAndDelete[T any](m map[string]chan T, key string) {
	c, has := m[key]
	if !has {
		return
	}
	close(c)
	delete(m, key)
}
