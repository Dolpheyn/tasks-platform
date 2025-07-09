package platform

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hibiken/asynq"
)

type TaskManager struct {
	mu sync.Mutex

	pendingTasksByType map[string][]*PlatformTask

	activeTasksByID   map[string]*PlatformTask
	hearbeatsByID     map[string][]*time.Time
	pickupSignalsByID map[string]chan PickupSignal
	dropSignalsByID   map[string]chan DropSignal
	resultSignalsByID map[string]chan ResultSignal
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		pendingTasksByType: map[string][]*PlatformTask{},
		activeTasksByID:    map[string]*PlatformTask{},
		hearbeatsByID:      map[string][]*time.Time{},
		pickupSignalsByID:  map[string]chan PickupSignal{},
		dropSignalsByID:    map[string]chan DropSignal{},
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

	// wait for pickup signal with timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pickupSignalListener := registerTaskRes.pickupSignalListener

	select {
	case pickup := <-pickupSignalListener:
		log.Printf("[TaskManager::ProcessTask] task picked up. platformTaskID=%s clientID=%s", registerTaskRes.platformTaskID, pickup.ClientID)
	case <-timeoutCtx.Done():
		log.Printf("[Taskmanager::ProcessTask] deadline exceeded without pickup")
		return fmt.Errorf("[taskplatform] deadline exceeded without pickup")
	}

	// task picked up, wait for result with timeout
	timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	platformTaskID := registerTaskRes.platformTaskID
	resultListener := registerTaskRes.resultListener

	select {
	case <-timeoutCtx.Done():
		log.Printf("[Taskmanager::ProcessTask] worker exceeded deadline")
		m.unregisterTask(platformTaskID)
		return fmt.Errorf("[taskplatform] worker exceeded deadline")
	case taskResult := <-resultListener:
		if !taskResult.Success {
			return fmt.Errorf("[client] %s", taskResult.Msg)
		}
	}

	return nil
}

// TryConsumeTask consumes a pending task by taskType, then signal the pickup to the waiting ProcessTask above.
// errors:
// - if the task queue is empty
// - if no listeners are found (unlikely)
func (m *TaskManager) TryConsumeTask(ctx context.Context, taskType string, clientID string) (*PlatformTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, ok := m.pendingTasksByType[taskType]
	if !ok || len(tasks) == 0 {
		return nil, ErrTaskQueueEmpty
	}

	log.Printf("[TryConsumeTask] got tasks %+v", tasks)
	task := tasks[0]
	log.Printf("[TryConsumeTask] got task %+v", task)
	tasks = tasks[1:]
	m.pendingTasksByType[taskType] = tasks

	taskID := task.ID

	if _, isDuplicate := m.activeTasksByID[taskID]; isDuplicate {
		log.Printf("[warn] task already has an active listener. taskID=%s", task.ID)
		return nil, fmt.Errorf("task already has an active listener")
	}

	pickupSignalListener, ok := m.pickupSignalsByID[taskID]
	if !ok {
		log.Printf("[warn] pickup listener not found for task. taskID=%s", task.ID)
		return nil, fmt.Errorf("pickup signal listener not found")
	}

	pickupSignalListener <- PickupSignal{ClientID: clientID}

	return task, nil
}

type registerTaskRespose struct {
	platformTaskID       string
	pickupSignalListener <-chan PickupSignal
	dropSignalListener   <-chan DropSignal
	resultListener       <-chan ResultSignal
}

func (m *TaskManager) registerTask(task *asynq.Task) (*registerTaskRespose, error) {
	taskType := task.Type()
	pickupSignal := make(chan PickupSignal, 1)
	dropSignal := make(chan DropSignal, 1)
	resultSignal := make(chan ResultSignal, 1)

	platformTask, err := FromAsynqTask(task)
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
		tasks = make([]*PlatformTask, 0, 32)
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
	}
	return res, nil
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

type PickupSignal struct {
	ClientID string
}

type DropSignal struct {
	Reason string
}

type ResultSignal struct {
	Success bool
	Msg     string
}
