package api

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	dto "github.com/dolpheyn/tasks-platform/internal/api/dto"
	handlers "github.com/dolpheyn/tasks-platform/internal/api/handlers"
	"github.com/dolpheyn/tasks-platform/internal/config"
	"github.com/dolpheyn/tasks-platform/pkg/platform/taskmanager"
)

type Server struct {
	config      *config.Config
	router      *gin.Engine
	asynqClient *asynq.Client

	taskManager *taskmanager.TaskManager
}

func NewServer(cfg *config.Config) *Server {
	router := gin.Default()

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Username: "tasks-platform",
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	server := &Server{
		config:      cfg,
		router:      router,
		asynqClient: asynq.NewClient(&redisOpt),
		taskManager: taskmanager.NewTaskManager(),
	}

	server.startTaskManager(&redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	server.setupRoutes()
	return server
}

func (s *Server) Run() error {
	return s.router.Run(fmt.Sprintf(":%d", s.config.Server.Port))
}

func (s *Server) setupRoutes() {
	s.router.POST("/tasks", s.enqueueTask)
	s.router.POST("/tasks/scheduled", s.scheduleTask)

	s.router.GET("/poll", s.pollTasks)

	s.router.POST("/heartbeat", s.sendHeartbeat)
	s.router.POST("/complete", s.completeTask)
	s.router.POST("/fail", s.failTask)

	s.router.GET("/health", s.healthCheck)
}

func (s *Server) startTaskManager(redisOpt *asynq.RedisClientOpt, cfg asynq.Config) {
	go func() {
		asynqServer := asynq.NewServer(redisOpt, cfg)

		if err := asynqServer.Run(s.taskManager); err != nil {
			log.Fatal(err)
		}
	}()
}

func (s *Server) enqueueTask(ctx *gin.Context) {
	var req dto.EnqueueTaskRequest
	if !extractReqJSON(ctx, &req) {
		return
	}

	res, err := handlers.HandleEnqueueTask(ctx.Request.Context(), &req, s.asynqClient)
	if err != nil {
		resErrorInternal(ctx, err)
		return
	}

	ctx.JSON(200, res)
}

func (s *Server) pollTasks(ctx *gin.Context) {
	var req dto.PollTaskRequest
	if !extractReqForm(ctx, &req) {
		return
	}

	res, err := handlers.HandlePollTask(ctx.Request.Context(), &req, s.taskManager)
	if err != nil {
		resErrorInternal(ctx, err)
	}

	ctx.JSON(200, res)
}

func (s *Server) scheduleTask(ctx *gin.Context) {
	var req dto.ScheduleTaskRequest
	if !extractReqJSON(ctx, &req) {
		return
	}

	res, err := handlers.HandleScheduleTask(ctx.Request.Context(), &req, s.asynqClient)
	if err != nil {
		resErrorInternal(ctx, err)
		return
	}
	ctx.JSON(200, res)
}

func (s *Server) sendHeartbeat(ctx *gin.Context) {
}

func (s *Server) completeTask(ctx *gin.Context) {
	var req dto.CompleteRequest
	if !extractReqJSON(ctx, &req) {
		return
	}

	err := handlers.HandleCompleteTask(ctx, &req, s.taskManager)
	if err != nil {
		resErrorInternal(ctx, err)
		return
	}
	ctx.JSON(202, "")
}

func (s *Server) failTask(ctx *gin.Context) {
	var req dto.FailRequest

	err := handlers.HandleFailTask(ctx, &req, s.taskManager)
	if err != nil {
		resErrorInternal(ctx, err)
		return
	}
	ctx.JSON(202, "")
}

func (s *Server) healthCheck(ctx *gin.Context) {
}

// Helpers

func resErrorInternal(ctx *gin.Context, err error) {
	ctx.JSON(500, gin.H{"error": err.Error()})
}

func extractReqJSON(ctx *gin.Context, obj any) bool {
	if err := ctx.ShouldBindJSON(obj); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return false
	}

	return true
}

func extractReqForm(ctx *gin.Context, obj any) bool {
	if err := ctx.ShouldBind(obj); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return false
	}

	return true
}
