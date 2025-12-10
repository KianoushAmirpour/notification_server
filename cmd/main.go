package main

import (
	"github.com/KianoushAmirpour/notification_server/internal/application"
	config "github.com/KianoushAmirpour/notification_server/internal/infrastructure/configs"
)

func main() {
	cfg, err := config.LoadConfigs("../.env")
	if err != nil {
		panic(err)
	}

	app := application.App{Cfg: cfg}
	app.Run()
}
