package main

import (
	"github.com/KianoushAmirpour/notification_server/internal/application"
	config "github.com/KianoushAmirpour/notification_server/internal/infrastructure/configs"
)

// @title Notification Service API
// @version 1.0
// @description REST API for notifications
// @host localhost:4000
// @BasePath /
func main() {
	cfg, err := config.LoadConfigs(".env")
	if err != nil {
		panic(err)
	}

	app := application.App{Cfg: cfg}
	app.Run()
}
