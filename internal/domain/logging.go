package domain

type LoggingRepository interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	With(args ...interface{}) LoggingRepository
}
