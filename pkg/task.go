package platform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// PlatformTask represents a task in the platform with ID and payload
type PlatformTask struct {
	ID               string     `json:"id"`
	TypeName         string     `json:"type_name"`
	PlatformDeadline *time.Time `json:"platform_deadline"`
	Payload          []byte     `json:"payload"`
}

// FromAsynqTask converts asynq.Task payload to PlatformTask
func FromAsynqTask(task *asynq.Task) (*PlatformTask, error) {
	var pt PlatformTask
	if err := json.Unmarshal(task.Payload(), &pt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal platform task: %v", err)
	}
	pt.TypeName = task.Type()
	return &pt, nil
}

// ToAsynqTask converts PlatformTask to asynq.Task
func (pt *PlatformTask) ToAsynqTask() (*asynq.Task, error) {
	payload, err := json.Marshal(pt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal platform task: %v", err)
	}
	return asynq.NewTask(pt.TypeName, payload), nil
}
