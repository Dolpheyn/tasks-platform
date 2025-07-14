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
	log.Println("Job producer started - will create tasks randomly every 1 - 5 seconds")

	for {
		delay := time.Duration(rand.Intn(4000)+1000) * time.Millisecond // 1s–6s
		timer := time.After(delay)

		select {
		case <-ctx.Done():
			return nil
		case <-timer:
			if err := p.createRandomTask(); err != nil {
				log.Printf("Producer: Error creating task: %v", err)
			}

			// Create 10s scheduled task using ProcessIn
			taskPayload, err := createTaskPayload()
			if err != nil {
				log.Printf("Producer: Error creating task payload: %v", err)
				return nil
			}

			processIn := dto.ScheduleTaskRequest{
				JobType: "scheduled-process-in-" + p.taskType,
				Payload: taskPayload,
				ProcessIn: &dto.ProcessIn{
					DurationSecs: 10,
				},
			}
			if err := p.createScheduledTask(processIn); err != nil {
				log.Printf("Producer: Error creating 10s scheduled task: %v", err)
			}

			processAtTime := time.Now().Add(15 * time.Second)
			processAt := dto.ScheduleTaskRequest{
				JobType: "scheduled-process-at-" + p.taskType,
				Payload: taskPayload,
				ProcessAt: &dto.ProcessAt{
					Time: processAtTime,
				},
			}
			if err := p.createScheduledTask(processAt); err != nil {
				log.Printf("Producer: Error creating 15s scheduled task: %v", err)
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

func (p *JobProducer) createScheduledTask(taskReq dto.ScheduleTaskRequest) error {
	resp, err := callScheduleTask(taskReq)
	if err != nil {
		return fmt.Errorf("failed to send scheduled task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scheduled task creation failed with status %d", resp.StatusCode)
	}

	scheduleType := "ProcessIn"
	scheduleValue := ""
	if taskReq.ProcessIn != nil {
		scheduleValue = fmt.Sprintf("%ds", taskReq.ProcessIn.DurationSecs)
	} else if taskReq.ProcessAt != nil {
		scheduleType = "ProcessAt"
		scheduleValue = taskReq.ProcessAt.Time.Format("15:04:05")
	}

	log.Printf("Producer: Created scheduled task (%s: %s) with payload size %d bytes", scheduleType, scheduleValue, len(taskReq.Payload))
	return nil
}

func callEnqueueTask(taskReq dto.EnqueueTaskRequest) (resp *http.Response, err error) {
	jsonData, err := json.Marshal(taskReq)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %v", err)
	}

	return http.Post("http://localhost:8080/tasks", "application/json", bytes.NewBuffer(jsonData))
}

func callScheduleTask(taskReq dto.ScheduleTaskRequest) (resp *http.Response, err error) {
	jsonData, err := json.Marshal(taskReq)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal scheduled task request: %v", err)
	}

	return http.Post("http://localhost:8080/tasks/scheduled", "application/json", bytes.NewBuffer(jsonData))
}
