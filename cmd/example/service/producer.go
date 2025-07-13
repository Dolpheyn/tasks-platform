package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
)

type JobProducer struct {
	taskType string
}

func NewJobProducer(taskType string) *JobProducer {
	return &JobProducer{
		taskType: taskType,
	}
}

func (p *JobProducer) Start(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	log.Println("Job producer started - will create tasks every 3 seconds")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.createRandomTask(); err != nil {
				log.Printf("Producer: Error creating task: %v", err)
			}
		}
	}
}

func (p *JobProducer) createRandomTask() error {
	taskPayload, err := createTaskPayload()
	if err != nil {
		return err
	}

	resp, err := callEnqueueTask(dto.EnqueueTaskRequest{
		JobType: p.taskType,
		Payload: taskPayload,
	})
	if err != nil {
		return fmt.Errorf("failed to send task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("task creation failed with status %d", resp.StatusCode)
	}

	log.Printf("Producer: Created task with payload size %d bytes", len(taskPayload))
	return nil
}

func createTaskPayload() (json.RawMessage, error) {
	payload := map[string]any{
		"user_id":   rand.Intn(1000),
		"operation": "data_processing",
		"data_size": rand.Intn(1000000),
		"timestamp": time.Now().Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	return payloadBytes, nil
}

func callEnqueueTask(taskReq dto.EnqueueTaskRequest) (resp *http.Response, err error) {
	jsonData, err := json.Marshal(taskReq)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %v", err)
	}

	return http.Post("http://localhost:8080/tasks", "application/json", bytes.NewBuffer(jsonData))
}
