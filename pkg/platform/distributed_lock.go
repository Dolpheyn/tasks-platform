package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DistributedLock provides Redis-based distributed locking for cluster-safe operations
type DistributedLock struct {
	redisClient *redis.Client
	lockTimeout time.Duration
	instanceID  string
}

// LockResult contains information about a lock acquisition attempt
type LockResult struct {
	Acquired bool
	LockKey  string
	Value    string
	TTL      time.Duration
}

// NewDistributedLock creates a new distributed lock manager
func NewDistributedLock(redisClient *redis.Client, lockTimeout time.Duration) *DistributedLock {
	return &DistributedLock{
		redisClient: redisClient,
		lockTimeout: lockTimeout,
		instanceID:  uuid.New().String(),
	}
}

// AcquireLock attempts to acquire a distributed lock with the given key and TTL
// Returns true if the lock was acquired, false if another instance holds it
func (dl *DistributedLock) AcquireLock(ctx context.Context, key string, ttl time.Duration) (*LockResult, error) {
	lockValue := fmt.Sprintf("%s:%d", dl.instanceID, time.Now().UnixNano())
	
	// Use Redis SET with NX (only if not exists) and EX (expiration) options
	// This is atomic and ensures only one instance can acquire the lock
	result, err := dl.redisClient.SetNX(ctx, key, lockValue, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock %s: %v", key, err)
	}
	
	return &LockResult{
		Acquired: result,
		LockKey:  key,
		Value:    lockValue,
		TTL:      ttl,
	}, nil
}

// ReleaseLock releases a lock if it's held by this instance
// Uses Lua script to ensure atomic check-and-delete operation
func (dl *DistributedLock) ReleaseLock(ctx context.Context, lockResult *LockResult) error {
	if !lockResult.Acquired {
		return nil // Nothing to release
	}
	
	// Lua script to atomically check if we own the lock and delete it
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`
	
	result, err := dl.redisClient.Eval(ctx, luaScript, []string{lockResult.LockKey}, lockResult.Value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock %s: %v", lockResult.LockKey, err)
	}
	
	if result.(int64) == 0 {
		return fmt.Errorf("lock %s was not owned by this instance", lockResult.LockKey)
	}
	
	return nil
}

// ExtendLock extends the TTL of a lock if it's held by this instance
// Uses Lua script to ensure atomic check-and-extend operation
func (dl *DistributedLock) ExtendLock(ctx context.Context, lockResult *LockResult, newTTL time.Duration) error {
	if !lockResult.Acquired {
		return fmt.Errorf("cannot extend lock that was not acquired")
	}
	
	// Lua script to atomically check if we own the lock and extend its TTL
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	
	result, err := dl.redisClient.Eval(ctx, luaScript, []string{lockResult.LockKey}, lockResult.Value, int(newTTL.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock %s: %v", lockResult.LockKey, err)
	}
	
	if result.(int64) == 0 {
		return fmt.Errorf("lock %s was not owned by this instance", lockResult.LockKey)
	}
	
	lockResult.TTL = newTTL
	return nil
}

// IsLocked checks if a lock exists (regardless of who owns it)
func (dl *DistributedLock) IsLocked(ctx context.Context, key string) (bool, error) {
	exists, err := dl.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check lock existence %s: %v", key, err)
	}
	return exists > 0, nil
}

// GetLockInfo returns information about who holds a lock
func (dl *DistributedLock) GetLockInfo(ctx context.Context, key string) (string, time.Duration, error) {
	pipe := dl.redisClient.Pipeline()
	getCmd := pipe.Get(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		if err == redis.Nil {
			return "", 0, fmt.Errorf("lock %s does not exist", key)
		}
		return "", 0, fmt.Errorf("failed to get lock info %s: %v", key, err)
	}
	
	value, err := getCmd.Result()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get lock value %s: %v", key, err)
	}
	
	ttl, err := ttlCmd.Result()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get lock TTL %s: %v", key, err)
	}
	
	return value, ttl, nil
}

// GenerateCronLockKey creates a standardized lock key for cron jobs
func (dl *DistributedLock) GenerateCronLockKey(jobName string, triggerTime time.Time) string {
	// Use minute precision to ensure locks are unique per trigger time
	triggerMinute := triggerTime.Truncate(time.Minute)
	return fmt.Sprintf("cron:lock:%s:%d", jobName, triggerMinute.Unix())
}

// GetInstanceID returns the unique instance ID for this lock manager
func (dl *DistributedLock) GetInstanceID() string {
	return dl.instanceID
}