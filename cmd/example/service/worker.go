package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dolpheyn/tasks-platform/internal/api/dto"
)

type WorkerService struct {
	id       string
	taskType string
	port     int
}

func NewWorkerService(id string, taskType string) *WorkerService {
	return &WorkerService{
		id:       id,
		taskType: taskType,
	}
}

func (w *WorkerService) Start(ctx context.Context) {
	go w.pollForTasks(ctx)
}

func (w *WorkerService) pollForTasks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tryPollTask()
		}
	}
}

func (w *WorkerService) tryPollTask() {
	// Poll task from main service using GET with query parameters
	url := fmt.Sprintf("http://localhost:8080/poll?task_type=%s&worker_id=%s", w.taskType, w.id)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Worker %s: Error polling task: %v", w.id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var pollResp *dto.PollResponse
	if err := json.NewDecoder(resp.Body).Decode(&pollResp); err != nil {
		log.Printf("Worker %s: Error decoding poll response: %v", w.id, err)
		return
	}
	if pollResp == nil || pollResp.Task == nil {
		return
	}

	w.processTask(pollResp.Task, w.taskType)
}

func (w *WorkerService) processTask(task *dto.Task, taskType string) {
	log.Printf("Worker %s: Processing task %s of type %s", w.id, task.TaskID, taskType)

	go func() {
		time.Sleep(5 * time.Second)

		w.sendTaskComplete(task.TaskID)

		log.Printf("Worker %s: Completed task %s", w.id, task.TaskID)
	}()
}

func (w *WorkerService) sendTaskComplete(taskID string) {
	completeReq := map[string]any{
		"task_id":   taskID,
		"worker_id": w.id,
	}

	jsonData, err := json.Marshal(completeReq)
	if err != nil {
		log.Printf("Worker %s: Error marshaling completion request: %v", w.id, err)
		return
	}

	resp, err := http.Post("http://localhost:8080/complete", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Worker %s: Error sending completion: %v", w.id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		log.Printf("Worker %s: Completion request failed with status %d", w.id, resp.StatusCode)
	}
}
