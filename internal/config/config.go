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

func Load() *Config {
	var cfg Config
	
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal("Failed to load config:", err)
	}
	
	return &cfg
}
