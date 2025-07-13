package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dolpheyn/tasks-platform/cmd/example/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start 2 worker services in cluster
	wg.Add(2)

	startWorker(ctx, "worker-1", 8081, "send-email", &wg)
	startWorker(ctx, "worker-2", 8082, "generate-merchant-sales-report", &wg)

	// Start job producer
	wg.Add(2)
	startProducer(ctx, "send-email", &wg)
	startProducer(ctx, "generate-merchant-sales-report", &wg)

	fmt.Println("workers and producers started")
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nShutting down...")
	cancel()

	// Wait for all services to stop
	wg.Wait()
	fmt.Println("All services stopped.")
}

func startWorker(ctx context.Context, id string, port int, taskType string, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		service1 := service.NewWorkerService(id, port, taskType)
		service1.Start(ctx)
	}()

}

func startProducer(ctx context.Context, taskType string, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		producer := service.NewJobProducer(taskType)
		if err := producer.Start(ctx); err != nil {
			log.Printf("Producer error: %v", err)
		}
	}()
}
