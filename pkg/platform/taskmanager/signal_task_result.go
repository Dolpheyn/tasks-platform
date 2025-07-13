package taskmanager

import (
	"fmt"
)

func (m *TaskManager) SignalTaskResultSuccess(platformTaskID string) error {
	resultSignal, ok := m.resultSignalsByID[platformTaskID]
	if !ok {
		return fmt.Errorf("result signal not found. platformTaskID=%s", platformTaskID)
	}

	resultSignal <- ResultSignal{
		Success: true,
	}

	return nil
}

func (m *TaskManager) SignalTaskResultFailure(platformTaskID string, errMessage string) error {
	resultSignal, ok := m.resultSignalsByID[platformTaskID]
	if !ok {
		return fmt.Errorf("result signal not found. platformTaskID=%s", platformTaskID)
	}

	resultSignal <- ResultSignal{
		Success: false,
		Msg:     errMessage,
	}

	return nil
}
