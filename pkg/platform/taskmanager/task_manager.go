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

type TaskManager struct {
	mu sync.Mutex

	pendingTasksByType map[string][]*pfm.PlatformTask

	activeTasksByID   map[string]*pfm.PlatformTask
	hearbeatsByID     map[string][]*time.Time
	pickupSignalsByID map[string]chan PickupSignal
	dropSignalsByID   map[string]chan DropSignal
	resultSignalsByID map[string]chan ResultSignal
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		pendingTasksByType: map[string][]*pfm.PlatformTask{},
		activeTasksByID:    map[string]*pfm.PlatformTask{},
		hearbeatsByID:      map[string][]*time.Time{},
		pickupSignalsByID:  map[string]chan PickupSignal{},
		dropSignalsByID:    map[string]chan DropSignal{},
		resultSignalsByID:  map[string]chan ResultSignal{},
	}
}

// ProcessTask takes task from asynq, waits for it to get picked up by a client.
// when signalled that the task is being picked up, it immediately request to asynq for the lease of the task to be extended.
// then it check the heartbeat record, and keep extending the lease for as long as the client is sending heartbeat (with some timeout).
func (m *TaskManager) ProcessTask(ctx context.Context, task *asynq.Task) error {
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

func (m *TaskManager) waitForTaskPickup(pickupSignalListener <-chan PickupSignal) (*PickupSignal, error) {
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

func (m *TaskManager) waitForTaskResult(platformTaskID string, resultListener <-chan ResultSignal) error {
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

func (m *TaskManager) unregisterTask(platformTaskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activeTasksByID, platformTaskID)
	delete(m.hearbeatsByID, platformTaskID)
	closeChanAndDelete(m.pickupSignalsByID, platformTaskID)
	closeChanAndDelete(m.dropSignalsByID, platformTaskID)
	closeChanAndDelete(m.resultSignalsByID, platformTaskID)

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
