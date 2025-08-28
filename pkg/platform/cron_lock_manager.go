package platform

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// CronLockManager provides high-level lock management specifically for cron jobs
type CronLockManager struct {
	distributedLock *DistributedLock
	defaultTTL      time.Duration
}

// NewCronLockManager creates a new cron-specific lock manager
func NewCronLockManager(redisClient *redis.Client, lockTimeout time.Duration) *CronLockManager {
	return &CronLockManager{
		distributedLock: NewDistributedLock(redisClient, lockTimeout),
		defaultTTL:      lockTimeout,
	}
}

// TryExecuteWithLock attempts to acquire a lock for a cron job and execute the provided function
// This is the main method used by the cron scheduler to ensure cluster-safe execution
func (clm *CronLockManager) TryExecuteWithLock(ctx context.Context, cronJob *CronJob, triggerTime time.Time, executeFn func() error) error {
	lockKey := clm.distributedLock.GenerateCronLockKey(cronJob.Name, triggerTime)
	
	// Calculate TTL based on when the next cron trigger would occur
	ttl, err := cronJob.GetLockTTL(triggerTime)
	if err != nil {
		log.Printf("[CronLockManager] Failed to calculate TTL for job %s, using default: %v", cronJob.Name, err)
		ttl = clm.defaultTTL
	}
	
	log.Printf("[CronLockManager] Acquiring lock for job %s with TTL %v", cronJob.Name, ttl)
	
	// Try to acquire the lock
	lockResult, err := clm.distributedLock.AcquireLock(ctx, lockKey, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for job %s: %v", cronJob.Name, err)
	}
	
	if !lockResult.Acquired {
		// Another instance is already executing this job
		log.Printf("[CronLockManager] Job %s already being executed by another instance, skipping", cronJob.Name)
		return nil // This is not an error - it's expected behavior in a cluster
	}
	
	// We acquired the lock, ensure we release it when done
	defer func() {
		if err := clm.distributedLock.ReleaseLock(ctx, lockResult); err != nil {
			log.Printf("[CronLockManager] Failed to release lock for job %s: %v", cronJob.Name, err)
		}
	}()
	
	log.Printf("[CronLockManager] Acquired lock for job %s, executing...", cronJob.Name)
	
	// Execute the function with the lock held
	if err := executeFn(); err != nil {
		return fmt.Errorf("job execution failed for %s: %v", cronJob.Name, err)
	}
	
	log.Printf("[CronLockManager] Successfully executed job %s", cronJob.Name)
	return nil
}

// IsJobLocked checks if a specific cron job is currently locked (being executed)
func (clm *CronLockManager) IsJobLocked(ctx context.Context, jobName string, triggerTime time.Time) (bool, error) {
	lockKey := clm.distributedLock.GenerateCronLockKey(jobName, triggerTime)
	return clm.distributedLock.IsLocked(ctx, lockKey)
}

// GetJobLockInfo returns information about who is currently executing a job
func (clm *CronLockManager) GetJobLockInfo(ctx context.Context, jobName string, triggerTime time.Time) (string, time.Duration, error) {
	lockKey := clm.distributedLock.GenerateCronLockKey(jobName, triggerTime)
	return clm.distributedLock.GetLockInfo(ctx, lockKey)
}

// ExtendJobLock extends the lock for a running job (useful for long-running tasks)
func (clm *CronLockManager) ExtendJobLock(ctx context.Context, lockResult *LockResult, newTTL time.Duration) error {
	return clm.distributedLock.ExtendLock(ctx, lockResult, newTTL)
}

// GetInstanceID returns the unique instance ID for this lock manager
func (clm *CronLockManager) GetInstanceID() string {
	return clm.distributedLock.GetInstanceID()
}

// CleanupExpiredLocks removes expired lock keys (maintenance operation)
// This is typically called periodically to clean up Redis
func (clm *CronLockManager) CleanupExpiredLocks(ctx context.Context, jobNames []string, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	
	for _, jobName := range jobNames {
		// Generate lock keys for the past period and check if they're expired
		for t := cutoffTime; t.Before(time.Now()); t = t.Add(time.Minute) {
			lockKey := clm.distributedLock.GenerateCronLockKey(jobName, t)
			
			// Check if the lock exists and is expired
			exists, err := clm.distributedLock.IsLocked(ctx, lockKey)
			if err != nil {
				log.Printf("[CronLockManager] Error checking lock %s: %v", lockKey, err)
				continue
			}
			
			if exists {
				// Get TTL to see if it's actually expired
				_, ttl, err := clm.distributedLock.GetLockInfo(ctx, lockKey)
				if err != nil {
					continue // Lock might have been released
				}
				
				if ttl <= 0 {
					log.Printf("[CronLockManager] Found expired lock %s, cleaning up", lockKey)
					// Note: Redis should automatically clean up expired keys,
					// but we could add manual cleanup here if needed
				}
			}
		}
	}
	
	return nil
}