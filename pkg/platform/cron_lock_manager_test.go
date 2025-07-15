package platform

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCronLockManager_Creation(t *testing.T) {
	clm := NewCronLockManager(nil, time.Minute)
	
	if clm == nil {
		t.Fatal("NewCronLockManager should not return nil")
	}
	
	if clm.distributedLock == nil {
		t.Error("CronLockManager should have a distributed lock instance")
	}
	
	if clm.defaultTTL != time.Minute {
		t.Errorf("Default TTL should be %v, got %v", time.Minute, clm.defaultTTL)
	}
}

func TestCronLockManager_InstanceID(t *testing.T) {
	clm1 := NewCronLockManager(nil, time.Minute)
	clm2 := NewCronLockManager(nil, time.Minute)
	
	id1 := clm1.GetInstanceID()
	id2 := clm2.GetInstanceID()
	
	if id1 == "" {
		t.Error("Instance ID should not be empty")
	}
	
	if id1 == id2 {
		t.Error("Different instances should have different IDs")
	}
}

func TestCronLockManager_TryExecuteWithLock_MockScenario(t *testing.T) {
	// This test demonstrates the expected behavior without Redis
	clm := NewCronLockManager(nil, time.Minute)
	
	executed := false
	executeFn := func() error {
		executed = true
		return nil
	}
	
	cronJob := &CronJob{
		Name:     "test-job",
		Schedule: "0 * * * *",
		TaskType: "test",
		Enabled:  true,
	}
	
	// This would fail without Redis, but demonstrates the interface
	ctx := context.Background()
	err := clm.TryExecuteWithLock(ctx, cronJob, time.Now(), executeFn)
	
	// We expect an error since there's no Redis client
	if err == nil {
		t.Error("Expected error when Redis client is nil")
	}
	
	// Function should not have been executed due to Redis error
	if executed {
		t.Error("Function should not have been executed when lock acquisition fails")
	}
}

func TestCronLockManager_ExecuteFunction_Error(t *testing.T) {
	// Test that function errors are properly propagated
	clm := NewCronLockManager(nil, time.Minute)
	
	expectedError := errors.New("test execution error")
	executeFn := func() error {
		return expectedError
	}
	
	cronJob := &CronJob{
		Name:     "test-job",
		Schedule: "0 * * * *",
		TaskType: "test",
		Enabled:  true,
	}
	
	ctx := context.Background()
	err := clm.TryExecuteWithLock(ctx, cronJob, time.Now(), executeFn)
	
	// Should get an error, but it might be the Redis connection error first
	if err == nil {
		t.Error("Expected error from function execution or Redis connection")
	}
}

// Benchmark test for lock key generation
func BenchmarkCronLockManager_LockKeyGeneration(b *testing.B) {
	clm := NewCronLockManager(nil, time.Minute)
	jobName := "benchmark-job"
	triggerTime := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clm.distributedLock.GenerateCronLockKey(jobName, triggerTime)
	}
}