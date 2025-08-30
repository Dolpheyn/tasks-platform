package main

import (
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api"
	"github.com/dolpheyn/tasks-platform/internal/config"
	"github.com/dolpheyn/tasks-platform/pkg/platform/taskmanager"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	taskManager := taskmanager.NewRedisTaskManager(redisClient)

	server := api.NewServer(cfg, taskManager)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
