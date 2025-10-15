package main

import (
	"fmt"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
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

	// conn, err := postgres.OpenDatabaseConn(cfg.DatabaseDSN)
	// if err != nil {
	// 	panic(err)
	// }

	pool, err := postgres.OpenDatabaseConnPool(cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}

	rdb, err := redis.ConnectToRedis(fmt.Sprintf("localhost:%d", cfg.RedisPort), cfg.RedisDB)
	if err != nil {
		panic(err)
	}
	defer rdb.Close()
	otpService := redis.NewRedisClient(rdb)

	bcryptHasher := adapters.Hasher{Cost: cfg.BcryptCost}
	mailer := adapters.Mailer{Host: cfg.SmtpHost, Port: cfg.SmtpPort, Username: cfg.SmtpUsername, Password: cfg.SmtpPassword}

	iplimiter := adapters.NewIpLimiter()
	UserRepo := postgres.NewPostgresPoolUserRepo(pool)
	svc := service.NewUserRegisterService(UserRepo, bcryptHasher, mailer, otpService, pool)
	h := handler.NewUserHandler(svc, rdb, cfg, iplimiter)

	routerCfg := router.RouterConfig{UserHandler: h}

	g := router.SetupRoutes(routerCfg)

	g.Run(fmt.Sprintf(":%d", cfg.ServerPort))

}
