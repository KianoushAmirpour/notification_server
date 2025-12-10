package logger

import (
	"log/slog"
	"os"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type Logger struct {
	SlogLogger *slog.Logger
}

func NewLogger(loggingFilePath string) *Logger {
	file, err := os.Create(loggingFilePath)
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}))

	return &Logger{SlogLogger: logger}
}

func (l Logger) Info(msg string, args ...interface{}) {
	l.SlogLogger.Info(msg, args...)

}

func (l Logger) Warn(msg string, args ...interface{}) {
	l.SlogLogger.Warn(msg, args...)
}

func (l Logger) Error(msg string, args ...interface{}) {
	l.SlogLogger.Error(msg, args...)

}

func (l Logger) With(args ...any) domain.LoggingRepository {
	return &Logger{
		SlogLogger: l.SlogLogger.With(args...),
	}
}
