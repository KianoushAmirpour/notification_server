package adapters

import (
	"log/slog"
	"os"
)

var log *slog.Logger

func InitializeLogger(loggingFilePath string) *slog.Logger {

	file, err := os.Create(loggingFilePath)
	if err != nil {
		panic(err)
	}

	log = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))

	return log
}
