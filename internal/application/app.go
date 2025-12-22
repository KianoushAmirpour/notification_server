package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	router "github.com/KianoushAmirpour/notification_server/internal/adapters/http"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/handler"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/http/middleware"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/ai"
	config "github.com/KianoushAmirpour/notification_server/internal/infrastructure/configs"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/notification"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/queue"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/repository/postgres"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/repository/redis"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/security"
	"github.com/KianoushAmirpour/notification_server/internal/usecase"
	"github.com/KianoushAmirpour/notification_server/pkg/logger"
)

type App struct {
	Cfg *config.Config
}

func (a App) Run() {

	rootctx, rootcancel := context.WithCancel(context.Background())
	defer rootcancel()

	logger := logger.NewLogger(a.Cfg.LogFile)

	dbPool, err := postgres.OpenDatabaseConnPool(a.Cfg.DatabaseDSN)
	if err != nil {
		logger.Error("database connection failed", "reason", err.Error())
		panic(err)
	}
	defer dbPool.Close()

	redisConn, err := redis.ConnectToRedis(fmt.Sprintf("localhost:%d", a.Cfg.RedisPort), a.Cfg.RedisDB)
	if err != nil {
		logger.Error("redis connection failed", slog.String("reason", err.Error()))
		panic(err)
	}
	defer redisConn.Close()

	bcryptPasswordHasher := security.Hasher{Cost: a.Cfg.BcryptCost}

	otpService := redis.NewRedisClient(redisConn, bcryptPasswordHasher)

	storyGenerationTask := &redis.Task{Client: redisConn, GroupName: a.Cfg.StoryConsumerGroup}
	emailNotificationTask := &redis.Task{Client: redisConn, GroupName: a.Cfg.EmailConsumerGroup}

	mailer := notification.Mailer{
		Host:      a.Cfg.SmtpHost,
		Port:      a.Cfg.SmtpPort,
		Username:  a.Cfg.SmtpUsername,
		Password:  a.Cfg.SmtpPassword,
		FromEmail: a.Cfg.FromEmail,
		Logger:    logger}

	iplimiter := middleware.NewIpLimiter()

	UserRepo := postgres.NewUserRepo(dbPool)
	UserVerificationRepo := postgres.NewUserVerificationRepo(dbPool)
	StoryRepo := postgres.NewStoryRepo(dbPool)
	RefreshTokenRepo := postgres.NewRefreshTokenRepo(dbPool)

	otpgenerator := security.Otpgen{OTPLength: a.Cfg.OTPLength}

	jwttoken := security.JwtAuth{AccessSecret: []byte(a.Cfg.JwtAccessSecret), RefreshSecret: []byte(a.Cfg.JwtRefreshSecret), Issuer: a.Cfg.JwtISS}

	userRegisterSvc := usecase.NewUserRegisterService(UserRepo, UserVerificationRepo, bcryptPasswordHasher, mailer, otpService, otpgenerator, jwttoken, RefreshTokenRepo, logger)

	gemeniClient, err := ai.NewGemeniClient(rootctx, a.Cfg.GeminiAPI, a.Cfg.GeminiModel)
	if err != nil {
		logger.Error("failed to create gemeni client", "reason", err.Error())
		panic(err)
	}

	storyJobExecuter := usecase.NewStoryGenerationService(
		UserRepo,
		StoryRepo,
		gemeniClient,
		logger)
	storyJobCompletionHandler := usecase.NewStoryGenerationJobCompletion(
		StoryRepo,
		storyGenerationTask,
		a.Cfg.StoryGenerationStream,
		a.Cfg.EmailNotificationStream,
		a.Cfg.StoryDLQStream,
		logger)

	emailJobExecuter := usecase.NewEmailSenderService(mailer, logger)
	emailJobCompletionHandler := usecase.NewEmailNotificationJobCompletion(
		StoryRepo,
		emailNotificationTask,
		a.Cfg.EmailNotificationStream,
		a.Cfg.EmailDLQStream,
		logger)

	// resultchan := make(chan domain.EmailNotificationJob, 100)
	storyWorkerPool := queue.NewWorkerPool(rootctx, a.Cfg.WorkerCounts, logger, storyGenerationTask,
		storyJobExecuter, storyJobCompletionHandler, a.Cfg.StoryGenerationStream, a.Cfg.StoryConsumerGroup, a.Cfg.JobRetryCount)
	emailWorkerPool := queue.NewWorkerPool(rootctx, a.Cfg.WorkerCounts, logger, emailNotificationTask, emailJobExecuter, emailJobCompletionHandler,
		a.Cfg.EmailNotificationStream, a.Cfg.EmailConsumerGroup, a.Cfg.JobRetryCount)
	storyWorkerPool.Start()
	emailWorkerPool.Start()
	// emailWorkerPool := queue.NewEmailWorkerPool(rootctx, a.Cfg.WorkerCounts, a.Cfg.JobQueueSize, logger)
	// emailWorkerPool.Start(resultchan)
	// workerPool := queue.NewWorkerPool(rootctx, a.Cfg.WorkerCounts, a.Cfg.JobQueueSize, logger, mailer)
	// workerPool.Start(resultchan)

	schedulerStoryConsumer := queue.NewShedulerWorkerPool(rootctx, a.Cfg.SchedulerWorkerCounts, logger, a.Cfg.StoryRetryStream, storyGenerationTask, a.Cfg.StoryGenerationStream)
	schedulerStoryConsumer.Start()
	schedulerEmailConsumer := queue.NewShedulerWorkerPool(rootctx, a.Cfg.SchedulerWorkerCounts, logger, a.Cfg.EmailRetryStream, emailNotificationTask, a.Cfg.EmailNotificationStream)
	schedulerEmailConsumer.Start()

	storyScheduleService := usecase.NewStorySchedulerService(UserRepo, StoryRepo, storyGenerationTask, logger, a.Cfg.StoryGenerationStream)

	h := handler.NewUserHandler(userRegisterSvc, storyScheduleService, iplimiter, jwttoken, logger,
		a.Cfg.OTPExpiration, a.Cfg.JwtISS, a.Cfg.JwtAccessSecret, a.Cfg.JwtRefreshSecret, a.Cfg.RataLimitCapacity, a.Cfg.RataLimitFillRate,
		a.Cfg.MaxAllowedSize)

	routerCfg := router.RouterConfig{UserHandler: h}

	g := router.SetupRoutes(routerCfg)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Cfg.ServerPort),
		Handler: g,
	}

	go func() {
		serverErr := server.ListenAndServe()
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			logger.Error("failed to start the server", "reason", err.Error())
		}
		logger.Info("successfully start the server")
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan

	shutdownctx, shutdowncancelFunc := context.WithTimeout(context.Background(), time.Duration(a.Cfg.ServerShutdownTimeout)*time.Second)
	defer shutdowncancelFunc()
	if err := server.Shutdown(shutdownctx); err != nil {
		logger.Error("server closed with error", "reason", err.Error())
	}

	storyWorkerPool.Cancel()
	emailWorkerPool.Cancel()

	storyWorkerPool.Wait()
	emailWorkerPool.Wait()

	logger.Info("check number of goroutine", "number", runtime.NumGoroutine())

}
