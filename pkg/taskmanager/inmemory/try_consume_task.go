package inmemory

import (
	"context"
	"fmt"
	"log"

	pfm "github.com/dolpheyn/tasks-platform/pkg"
	taskmanager "github.com/dolpheyn/tasks-platform/pkg/taskmanager"
)

// TryConsumeTask consumes a pending task by taskType, then signal the pickup to the waiting ProcessTask above.
// errors:
// - if the task queue is empty
// - if no listeners are found (unlikely)
func (m *InMemoryTaskManager) TryConsumeTask(ctx context.Context, taskType string, clientID string) (*pfm.PlatformTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, ok := m.pendingTasksByType[taskType]
	if !ok || len(tasks) == 0 {
		return nil, pfm.ErrTaskQueueEmpty
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

	pickupSignalListener <- taskmanager.PickupSignal{ClientID: clientID}

	return task, nil
}
