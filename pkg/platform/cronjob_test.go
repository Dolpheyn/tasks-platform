package platform

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCronJob_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cronJob CronJob
		wantErr bool
	}{
		{
			name: "valid cron job",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "0 2 * * *",
				TaskType: "cleanup",
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			cronJob: CronJob{
				Name:     "",
				Schedule: "0 2 * * *",
				TaskType: "cleanup",
			},
			wantErr: true,
		},
		{
			name: "empty schedule",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "",
				TaskType: "cleanup",
			},
			wantErr: true,
		},
		{
			name: "empty task type",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "0 2 * * *",
				TaskType: "",
			},
			wantErr: true,
		},
		{
			name: "invalid cron schedule",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "invalid-cron",
				TaskType: "cleanup",
			},
			wantErr: true,
		},
		{
			name: "invalid timezone",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "0 2 * * *",
				TaskType: "cleanup",
				Timezone: "Invalid/Timezone",
			},
			wantErr: true,
		},
		{
			name: "valid timezone",
			cronJob: CronJob{
				Name:     "test-job",
				Schedule: "0 2 * * *",
				TaskType: "cleanup",
				Timezone: "America/New_York",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cronJob.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CronJob.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCronJob_GetLocation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantName string
		wantErr  bool
	}{
		{
			name:     "empty timezone defaults to UTC",
			timezone: "",
			wantName: "UTC",
			wantErr:  false,
		},
		{
			name:     "valid timezone",
			timezone: "America/New_York",
			wantName: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "invalid timezone",
			timezone: "Invalid/Timezone",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cj := &CronJob{Timezone: tt.timezone}
			loc, err := cj.GetLocation()
			if (err != nil) != tt.wantErr {
				t.Errorf("CronJob.GetLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && loc.String() != tt.wantName {
				t.Errorf("CronJob.GetLocation() = %v, want %v", loc.String(), tt.wantName)
			}
		})
	}
}

func TestCronJob_JSON(t *testing.T) {
	now := time.Now()
	payload := json.RawMessage(`{"action": "cleanup", "retention_days": 30}`)
	
	original := &CronJob{
		Name:      "test-job",
		Schedule:  "0 2 * * *",
		TaskType:  "cleanup",
		Payload:   payload,
		Timezone:  "UTC",
		Enabled:   true,
		Metadata:  map[string]string{"owner": "platform-team"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test ToJSON
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("CronJob.ToJSON() error = %v", err)
	}

	// Test FromJSON
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	// Compare fields
	if restored.Name != original.Name {
		t.Errorf("Name = %v, want %v", restored.Name, original.Name)
	}
	if restored.Schedule != original.Schedule {
		t.Errorf("Schedule = %v, want %v", restored.Schedule, original.Schedule)
	}
	if restored.TaskType != original.TaskType {
		t.Errorf("TaskType = %v, want %v", restored.TaskType, original.TaskType)
	}
	if string(restored.Payload) != string(original.Payload) {
		t.Errorf("Payload = %v, want %v", string(restored.Payload), string(original.Payload))
	}
}