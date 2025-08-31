package main

import (
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api"
	"github.com/dolpheyn/tasks-platform/internal/config"
	redistaskmanager "github.com/dolpheyn/tasks-platform/pkg/taskmanager/redis"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password, // no password set
		DB:       cfg.Redis.DB,       // use default DB
	})

	taskManager := redistaskmanager.NewRedisTaskManager(redisClient)

	server := api.NewServer(cfg, taskManager)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
