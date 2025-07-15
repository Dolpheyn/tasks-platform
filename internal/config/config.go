package config

import (
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server         *ServerConfig
	Redis          *RedisConfig
	WorkerSessions *WorkerSessionsConfig
	Cron           *CronConfig
}

type RedisConfig struct {
	Addr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

type ServerConfig struct {
	Port int `envconfig:"PORT" default:"8080"`
}

type WorkerSessionsConfig struct {
	HeartbeatInterval time.Duration `envconfig:"HEARTBEAT_INTERVAL" default:"30s"`
	HeartbeatTimeout  time.Duration `envconfig:"HEARTBEAT_TIMEOUT" default:"90s"`
	PollTimeout       time.Duration `envconfig:"POLL_TIMEOUT" default:"30s"`
	MaxBatchSize      int           `envconfig:"MAX_BATCH_SIZE" default:"5"`
}

type CronConfig struct {
	Enabled         bool          `envconfig:"CRON_ENABLED" default:"true"`
	ConfigPath      string        `envconfig:"CRON_CONFIG_PATH" default:"./cron"`
	LockTimeout     time.Duration `envconfig:"CRON_LOCK_TIMEOUT" default:"60s"`
	Timezone        string        `envconfig:"CRON_TIMEZONE" default:"UTC"`
	RetentionPeriod time.Duration `envconfig:"CRON_RETENTION_PERIOD" default:"168h"` // 7 days
	MaxHistory      int           `envconfig:"CRON_MAX_HISTORY" default:"100"`
}

func Load() *Config {
	var cfg Config
	
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal("Failed to load config:", err)
	}
	
	return &cfg
}
