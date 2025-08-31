package inmemory

import (
	"fmt"

	taskmanager "github.com/dolpheyn/tasks-platform/pkg/taskmanager"
)

func (m *InMemoryTaskManager) SignalTaskResultSuccess(platformTaskID string) error {
	resultSignal, ok := m.resultSignalsByID[platformTaskID]
	if !ok {
		return fmt.Errorf("result signal not found. platformTaskID=%s", platformTaskID)
	}

	resultSignal <- taskmanager.ResultSignal{
		Success: true,
	}

	return nil
}

func (m *InMemoryTaskManager) SignalTaskResultFailure(platformTaskID string, errMessage string) error {
	resultSignal, ok := m.resultSignalsByID[platformTaskID]
	if !ok {
		return fmt.Errorf("result signal not found. platformTaskID=%s", platformTaskID)
	}

	resultSignal <- taskmanager.ResultSignal{
		Success: false,
		Msg:     errMessage,
	}

	return nil
}
