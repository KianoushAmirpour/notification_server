package main

import (
	"context"
	"fmt"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/ai"
	"github.com/KianoushAmirpour/notification_server/internal/config"
	"github.com/KianoushAmirpour/notification_server/internal/handler"
	"github.com/KianoushAmirpour/notification_server/internal/repository/postgres"
	"github.com/KianoushAmirpour/notification_server/internal/repository/redis"
	"github.com/KianoushAmirpour/notification_server/internal/router"
	"github.com/KianoushAmirpour/notification_server/internal/service"
)

func main() {

	cfg, err := config.LoadConfigs("../.env")
	if err != nil {
		panic(err)
	}

	logger := adapters.InitializeLogger(cfg.LogFile)

	// conn, err := postgres.OpenDatabaseConn(cfg.DatabaseDSN)
	// if err != nil {
	// 	panic(err)
	// }

	dbPool, err := postgres.OpenDatabaseConnPool(cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}

	redisConn, err := redis.ConnectToRedis(fmt.Sprintf("localhost:%d", cfg.RedisPort), cfg.RedisDB)
	if err != nil {
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

	gemeniClient := ai.NewGemeniClient(context.Background(), cfg)

	workerPool := adapters.NewWorkerPool(cfg.WorkerCounts, cfg.JobQueueSize, logger)
	resultchan := make(chan string, 100)
	workerPool.Start(resultchan)
	storyGenerationSvc := service.NewStoryGenerationService(UserRepo, gemeniClient, workerPool)

	h := handler.NewUserHandler(userRegisterSvc, storyGenerationSvc, cfg, iplimiter, logger)

	routerCfg := router.RouterConfig{UserHandler: h}

	g := router.SetupRoutes(routerCfg)

	go func() {
		for userEmail := range resultchan {
			fmt.Println(userEmail)
			err = mailer.SendNotification(userEmail)
			if err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		workerPool.Wg.Wait()
	}()

	g.Run(fmt.Sprintf(":%d", cfg.ServerPort))

	close(resultchan)
	workerPool.CancelFunc()
	close(workerPool.JobQueue)

}
