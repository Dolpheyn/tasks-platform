package taskmanager

import (
	"fmt"
	"log"

	pfm "github.com/dolpheyn/tasks-platform/pkg/platform"

	"github.com/hibiken/asynq"
)

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
