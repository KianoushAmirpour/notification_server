package domain

import (
	"context"
)

type Job struct {
	JobID           int
	UserID          int
	StoryID         int
	UserEmail       string
	UserPreferences string
	RetryCounts     int
	RequestID       string
}

type WorkerPool interface {
	Start()
	Cancel()
	Run(workerID int)
	Wait()
}
type Message struct {
	MessageID string
	Payload   Job
}

type StreamTaskHandler interface {
	CreateConsumerGroup(ctx context.Context, stream string, group string) error
	Add(ctx context.Context, job Job, stream string) error
	Read(ctx context.Context, consumerId int, stream string) (Message, error)
	Ack(ctx context.Context, messageID string, stream string) error
	ScheduleRetry(ctx context.Context, payload Job, stream string) error
	ReEnqueue(ctx context.Context, queue string, stream string) error
	Delete(ctx context.Context, messageID string, stream string) error
}

type JobExecuter interface {
	Execute(ctx context.Context, job Job) error
}

type JobComplettionHandler interface {
	OnSuccess(ctx context.Context, job Job, MessageID string) error
	OnFailure(ctx context.Context, job Job, MessageID string) error
	SendToDQL(ctx context.Context, job Job, MessageID string) error
}
