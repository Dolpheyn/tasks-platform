package inmemory

import (
	"sync"
	"time"

	pfm "github.com/dolpheyn/tasks-platform/pkg"
	taskmanager "github.com/dolpheyn/tasks-platform/pkg/taskmanager"
)

// InMemoryTaskManager is an in-memory implementation of TaskManager.
// NOTE: This implementation is not suitable for production use as it's not scalable and resilient.
// It's provided for demonstration and testing purposes.
type InMemoryTaskManager struct {
	mu sync.Mutex

	pendingTasksByType map[string][]*pfm.PlatformTask

	activeTasksByID   map[string]*pfm.PlatformTask
	heartbeatsByID    map[string][]*time.Time
	pickupSignalsByID map[string]chan taskmanager.PickupSignal
	dropSignalsByID   map[string]chan taskmanager.DropSignal
	resultSignalsByID map[string]chan taskmanager.ResultSignal
}

func NewInMemoryTaskManager() *InMemoryTaskManager {
	return &InMemoryTaskManager{
		pendingTasksByType: map[string][]*pfm.PlatformTask{},
		activeTasksByID:    map[string]*pfm.PlatformTask{},
		heartbeatsByID:     map[string][]*time.Time{},
		pickupSignalsByID:  make(map[string]chan taskmanager.PickupSignal),
		dropSignalsByID:    make(map[string]chan taskmanager.DropSignal),
		resultSignalsByID:  make(map[string]chan taskmanager.ResultSignal),
	}
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
