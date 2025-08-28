package platform

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// CronJob represents a scheduled recurring task
type CronJob struct {
	Name        string            `json:"name" yaml:"name"`
	Schedule    string            `json:"schedule" yaml:"schedule"`
	TaskType    string            `json:"task_type" yaml:"task_type"`
	Payload     json.RawMessage   `json:"payload" yaml:"payload"`
	Timezone    string            `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Validate checks if the CronJob configuration is valid
func (cj *CronJob) Validate() error {
	if cj.Name == "" {
		return fmt.Errorf("cron job name cannot be empty")
	}
	
	if cj.Schedule == "" {
		return fmt.Errorf("cron job schedule cannot be empty")
	}
	
	if cj.TaskType == "" {
		return fmt.Errorf("cron job task_type cannot be empty")
	}
	
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(cj.Schedule); err != nil {
		return fmt.Errorf("invalid cron schedule '%s': %v", cj.Schedule, err)
	}
	
	// Validate timezone if specified
	if cj.Timezone != "" {
		if _, err := time.LoadLocation(cj.Timezone); err != nil {
			return fmt.Errorf("invalid timezone '%s': %v", cj.Timezone, err)
		}
	}
	
	return nil
}

// GetLocation returns the time.Location for the cron job's timezone
func (cj *CronJob) GetLocation() (*time.Location, error) {
	if cj.Timezone == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(cj.Timezone)
}

// ToJSON serializes the CronJob to JSON bytes
func (cj *CronJob) ToJSON() ([]byte, error) {
	return json.Marshal(cj)
}

// FromJSON deserializes JSON bytes to CronJob
func FromJSON(data []byte) (*CronJob, error) {
	var cj CronJob
	if err := json.Unmarshal(data, &cj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cron job: %v", err)
	}
	return &cj, nil
}

// GetNextTriggerTime calculates the next time this cron job should run
func (cj *CronJob) GetNextTriggerTime(from time.Time) (time.Time, error) {
	loc, err := cj.GetLocation()
	if err != nil {
		return time.Time{}, err
	}
	
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cj.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron schedule: %v", err)
	}
	
	// Convert time to the job's timezone
	fromInTZ := from.In(loc)
	nextTime := schedule.Next(fromInTZ)
	
	return nextTime, nil
}

// GetLockTTL calculates the appropriate TTL for a lock based on when the next trigger would occur
func (cj *CronJob) GetLockTTL(triggerTime time.Time) (time.Duration, error) {
	nextTrigger, err := cj.GetNextTriggerTime(triggerTime)
	if err != nil {
		return 0, err
	}
	
	// TTL should be the time until the next trigger, with a small buffer
	ttl := nextTrigger.Sub(triggerTime)
	
	// Add a 30-second buffer to account for clock skew and processing time
	ttl += 30 * time.Second
	
	// Ensure minimum TTL of 1 minute for safety
	if ttl < time.Minute {
		ttl = time.Minute
	}
	
	// Cap maximum TTL at 24 hours for safety
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	
	return ttl, nil
}