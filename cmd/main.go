package main

import (
	"log"

	"github.com/dolpheyn/tasks-platform/internal/api"
	"github.com/dolpheyn/tasks-platform/internal/config"
)

func main() {
	cfg := config.Load()

	server := api.NewServer(cfg)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
