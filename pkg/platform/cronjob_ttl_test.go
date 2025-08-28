package platform

import (
	"testing"
	"time"
)

func TestCronJob_GetNextTriggerTime(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		from     time.Time
		timezone string
		wantNext string // Expected next trigger time format
	}{
		{
			name:     "daily at 2 AM",
			schedule: "0 2 * * *",
			from:     time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
			timezone: "UTC",
			wantNext: "2024-01-01 02:00:00", // Same day at 2 AM
		},
		{
			name:     "daily at 2 AM - after trigger time",
			schedule: "0 2 * * *",
			from:     time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			timezone: "UTC",
			wantNext: "2024-01-02 02:00:00", // Next day at 2 AM
		},
		{
			name:     "every hour",
			schedule: "0 * * * *",
			from:     time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
			timezone: "UTC",
			wantNext: "2024-01-01 13:00:00", // Next hour
		},
		{
			name:     "every 5 minutes",
			schedule: "*/5 * * * *",
			from:     time.Date(2024, 1, 1, 12, 32, 0, 0, time.UTC),
			timezone: "UTC",
			wantNext: "2024-01-01 12:35:00", // Next 5-minute mark
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronJob := &CronJob{
				Name:     "test-job",
				Schedule: tt.schedule,
				TaskType: "test",
				Timezone: tt.timezone,
				Enabled:  true,
			}

			nextTime, err := cronJob.GetNextTriggerTime(tt.from)
			if err != nil {
				t.Fatalf("GetNextTriggerTime() error = %v", err)
			}

			// Format the time for comparison
			got := nextTime.Format("2006-01-02 15:04:05")
			if got != tt.wantNext {
				t.Errorf("GetNextTriggerTime() = %v, want %v", got, tt.wantNext)
			}
		})
	}
}

func TestCronJob_GetLockTTL(t *testing.T) {
	tests := []struct {
		name        string
		schedule    string
		triggerTime time.Time
		minTTL      time.Duration
		maxTTL      time.Duration
	}{
		{
			name:        "daily job",
			schedule:    "0 2 * * *",
			triggerTime: time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC),
			minTTL:      23*time.Hour + 30*time.Second, // Almost 24 hours
			maxTTL:      24*time.Hour + time.Minute,    // Just over 24 hours
		},
		{
			name:        "hourly job",
			schedule:    "0 * * * *",
			triggerTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			minTTL:      59 * time.Minute,      // Almost 1 hour
			maxTTL:      61 * time.Minute,      // Just over 1 hour
		},
		{
			name:        "every 5 minutes",
			schedule:    "*/5 * * * *",
			triggerTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			minTTL:      4*time.Minute + 30*time.Second, // Almost 5 minutes
			maxTTL:      6 * time.Minute,                // Just over 5 minutes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronJob := &CronJob{
				Name:     "test-job",
				Schedule: tt.schedule,
				TaskType: "test",
				Timezone: "UTC",
				Enabled:  true,
			}

			ttl, err := cronJob.GetLockTTL(tt.triggerTime)
			if err != nil {
				t.Fatalf("GetLockTTL() error = %v", err)
			}

			if ttl < tt.minTTL {
				t.Errorf("GetLockTTL() = %v, should be at least %v", ttl, tt.minTTL)
			}

			if ttl > tt.maxTTL {
				t.Errorf("GetLockTTL() = %v, should be at most %v", ttl, tt.maxTTL)
			}

			// TTL should always be at least 1 minute
			if ttl < time.Minute {
				t.Errorf("GetLockTTL() = %v, should be at least 1 minute", ttl)
			}

			// TTL should never exceed 24 hours
			if ttl > 24*time.Hour {
				t.Errorf("GetLockTTL() = %v, should not exceed 24 hours", ttl)
			}
		})
	}
}

func TestCronJob_GetLockTTL_InvalidSchedule(t *testing.T) {
	cronJob := &CronJob{
		Name:     "test-job",
		Schedule: "invalid-cron",
		TaskType: "test",
		Timezone: "UTC",
		Enabled:  true,
	}

	_, err := cronJob.GetLockTTL(time.Now())
	if err == nil {
		t.Error("GetLockTTL() should return error for invalid cron schedule")
	}
}

func TestCronJob_GetLockTTL_MinimumTTL(t *testing.T) {
	// Test a very frequent job (every second, which isn't valid cron but tests the minimum)
	cronJob := &CronJob{
		Name:     "test-job",
		Schedule: "* * * * *", // Every minute (most frequent valid cron)
		TaskType: "test",
		Timezone: "UTC",
		Enabled:  true,
	}

	ttl, err := cronJob.GetLockTTL(time.Now())
	if err != nil {
		t.Fatalf("GetLockTTL() error = %v", err)
	}

	// Even for very frequent jobs, TTL should be at least 1 minute
	if ttl < time.Minute {
		t.Errorf("GetLockTTL() = %v, should enforce minimum of 1 minute", ttl)
	}
}

func TestCronJob_GetLockTTL_WithTimezone(t *testing.T) {
	cronJob := &CronJob{
		Name:     "test-job",
		Schedule: "0 2 * * *", // Daily at 2 AM
		TaskType: "test",
		Timezone: "America/New_York",
		Enabled:  true,
	}

	// Trigger time in UTC
	triggerTime := time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC) // 2 AM EST = 7 AM UTC

	ttl, err := cronJob.GetLockTTL(triggerTime)
	if err != nil {
		t.Fatalf("GetLockTTL() error = %v", err)
	}

	// Should be approximately 24 hours (daily job)
	expectedTTL := 24 * time.Hour
	tolerance := time.Hour

	if ttl < expectedTTL-tolerance || ttl > expectedTTL+tolerance {
		t.Errorf("GetLockTTL() = %v, expected around %v", ttl, expectedTTL)
	}
}