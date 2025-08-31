package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server *ServerConfig
	Redis  *RedisConfig
}

type ServerConfig struct {
	Port int `envconfig:"PORT" default:"8080"`
}

type RedisConfig struct {
	Addr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func Load() *Config {
	var cfg Config

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	return &cfg
}
