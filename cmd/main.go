package main

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

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/ai"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/handler"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
	"github.com/KianoushAmirpour/notification_server/internal/repository/postgres"
	"github.com/KianoushAmirpour/notification_server/internal/repository/redis"
	"github.com/KianoushAmirpour/notification_server/internal/router"
	"github.com/KianoushAmirpour/notification_server/internal/service"
)

func main() {

	rootctx, rootcancel := context.WithCancel(context.Background())
	defer rootcancel()

	cfg, err := config.LoadConfigs("../.env")
	if err != nil {
		panic(err)
	}

	logger := adapters.InitializeLogger(cfg.LogFile)

	dbPool, err := postgres.OpenDatabaseConnPool(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("database connection failed", slog.String("reason", err.Error()))
		panic(err)
	}
	defer dbPool.Close()

	redisConn, err := redis.ConnectToRedis(fmt.Sprintf("localhost:%d", cfg.RedisPort), cfg.RedisDB)
	if err != nil {
		logger.Error("redis connection failed", slog.String("reason", err.Error()))
		panic(err)
	}
	defer redisConn.Close()

	otpService := redis.NewRedisClient(redisConn)

	bcryptHasher := adapters.Hasher{Cost: cfg.BcryptCost}

	mailer := adapters.Mailer{
		Host:      cfg.SmtpHost,
		Port:      cfg.SmtpPort,
		Username:  cfg.SmtpUsername,
		Password:  cfg.SmtpPassword,
		FromEmail: cfg.FromEmail,
		Logger:    logger}

	iplimiter := adapters.NewIpLimiter()

	UserRepo := postgres.NewPostgresPoolUserRepo(dbPool)

	userRegisterSvc := service.NewUserRegisterService(UserRepo, bcryptHasher, mailer, otpService, dbPool)

	gemeniClient := ai.NewGemeniClient(rootctx, cfg, logger)

	resultchan := make(chan repository.Job, 10)
	emailWorkerPool := adapters.NewEmailWorkerPool(rootctx, cfg.WorkerCounts, cfg.JobQueueSize, logger)
	emailWorkerPool.Start(resultchan)
	workerPool := adapters.NewWorkerPool(rootctx, cfg.WorkerCounts, cfg.JobQueueSize, logger, mailer)
	workerPool.Start(resultchan)

	storyGenerationSvc := service.NewStoryGenerationService(UserRepo, gemeniClient, workerPool)

	h := handler.NewUserHandler(userRegisterSvc, storyGenerationSvc, cfg, iplimiter, logger)

	routerCfg := router.RouterConfig{UserHandler: h}

	g := router.SetupRoutes(routerCfg)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: g,
	}

	go func() {
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start the server", slog.String("reason", err.Error()))
		}
		logger.Info("successfully start the server")
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan

	shutdownctx, shutdowncancelFunc := context.WithTimeout(context.Background(), time.Duration(cfg.ServerShutdownTimeout)*time.Second)
	defer shutdowncancelFunc()
	if err := server.Shutdown(shutdownctx); err != nil {
		logger.Error("server closed with error", slog.String("reason", err.Error()))
	}

	workerPool.CancelFunc()
	emailWorkerPool.CancelFunc()

	workerPool.Wg.Wait()
	emailWorkerPool.Wg.Wait()

	close(workerPool.JobQueue)
	close(resultchan)
	logger.Info("check number of goroutine", slog.Int("number", runtime.NumGoroutine()))

}
