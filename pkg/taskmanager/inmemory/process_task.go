package inmemory

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	pfm "github.com/dolpheyn/tasks-platform/pkg"
	taskmanager "github.com/dolpheyn/tasks-platform/pkg/taskmanager"
)

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

type registerTaskRespose struct {
	platformTaskID       string
	pickupSignalListener <-chan taskmanager.PickupSignal
	dropSignalListener   <-chan taskmanager.DropSignal
	resultListener       <-chan taskmanager.ResultSignal
}

func (m *InMemoryTaskManager) registerTask(task *asynq.Task) (*registerTaskRespose, error) {
	taskType := task.Type()
	pickupSignal := make(chan taskmanager.PickupSignal, 1)
	dropSignal := make(chan taskmanager.DropSignal, 1)
	resultSignal := make(chan taskmanager.ResultSignal, 1)

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

func (m *InMemoryTaskManager) waitForTaskPickup(pickupSignalListener <-chan taskmanager.PickupSignal) (*taskmanager.PickupSignal, error) {
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

func (m *InMemoryTaskManager) waitForTaskResult(platformTaskID string, resultListener <-chan taskmanager.ResultSignal) error {
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
