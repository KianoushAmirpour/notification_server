package domain

import (
	"context"
)

type ContextKey string

type StoryUserInfo struct {
	UserID          int
	UserPreferences string
}

type GenerateStoryJob struct {
	UserInfo  StoryUserInfo
	AI        GenerateStoryRepository
	UserRepo  UserRepository
	StoryRepo StoryRepository
	Logger    LoggingRepository
}

type EmailNotificationJob struct {
	UserEmail string
	Mailer    Mailer
	Logger    LoggingRepository
}

type StoryWorkerPool interface {
	Submit(job GenerateStoryJob)
	Start(resultchan chan EmailNotificationJob)
	Cancel()
	Wait()
	Close()
}

type EmailWorkerPool interface {
	Start(resultchan chan EmailNotificationJob)
	Cancel()
	Wait()
	Close()
}

type StoryOrchestrator interface {
	RunStoryJob(ctx context.Context, storygenjob GenerateStoryJob) (string, error)
}

type EmailOrchestrator interface {
	RunEmailJob(ctx context.Context, emailjob EmailNotificationJob) error
}
