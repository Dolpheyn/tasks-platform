package api

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/dolpheyn/tasks-platform/internal/config"
	"github.com/dolpheyn/tasks-platform/pkg/platform"
)

type Server struct {
	config      *config.Config
	router      *gin.Engine
	asynqClient *asynq.Client

	taskManager *platform.TaskManager
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
		taskManager: platform.NewTaskManager(),
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
	s.router.GET("/apoll", s.pollTasks)
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

func (s *Server) enqueueTask(ctx *gin.Context) {
	var req EnqueueTaskRequest
	if !extractReqJSON(ctx, &req) {
		return
	}

	res, err := handleEnqueueTask(ctx.Request.Context(), &req, s.asynqClient)
	if err != nil {
		resErrorInternal(ctx, err)
		return
	}

	ctx.JSON(200, res)
}

func (s *Server) pollTasks(ctx *gin.Context) {
	var req PollTaskRequest
	if !extractReqForm(ctx, &req) {
		return
	}

	res, err := handlePollTask(ctx.Request.Context(), &req, s.taskManager)
	if err != nil {
		resErrorInternal(ctx, err)
	}

	ctx.JSON(200, res)
}

func (s *Server) scheduleTask(ctx *gin.Context) {
}

func (s *Server) sendHeartbeat(ctx *gin.Context) {
}

func (s *Server) completeTask(ctx *gin.Context) {
}

func (s *Server) failTask(ctx *gin.Context) {
}

func (s *Server) healthCheck(ctx *gin.Context) {
}

func resErrorInternal(ctx *gin.Context, err error) {
	ctx.JSON(500, gin.H{"error": err.Error()})
}
