package usecase

import (
	"context"
	"fmt"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type StoryGenerationJobCompletion struct {
	StoryRepo             domain.StoryRepository
	TaskStreamHandler     domain.StreamTaskHandler
	StoryStream           string
	NotificationStream    string
	DeadLetterQueueStream string
	Logger                domain.LoggingRepository
}

type EmailNotificationJobCompletion struct {
	StoryRepo             domain.StoryRepository
	TaskStreamHandler     domain.StreamTaskHandler
	EmailStream           string
	DeadLetterQueueStream string
	Logger                domain.LoggingRepository
}

func NewStoryGenerationJobCompletion(
	storyRepo domain.StoryRepository,
	taskStreamHandler domain.StreamTaskHandler,
	storyStream string,
	notificationStream string,
	dlq string,
	logger domain.LoggingRepository,
) *StoryGenerationJobCompletion {
	return &StoryGenerationJobCompletion{
		StoryRepo:             storyRepo,
		TaskStreamHandler:     taskStreamHandler,
		StoryStream:           storyStream,
		NotificationStream:    notificationStream,
		DeadLetterQueueStream: dlq,
		Logger:                logger,
	}
}

func NewEmailNotificationJobCompletion(
	storyRepo domain.StoryRepository,
	taskStreamHandler domain.StreamTaskHandler,
	emailStream string,
	dlq string,
	logger domain.LoggingRepository,
) *EmailNotificationJobCompletion {
	return &EmailNotificationJobCompletion{
		StoryRepo:             storyRepo,
		TaskStreamHandler:     taskStreamHandler,
		EmailStream:           emailStream,
		DeadLetterQueueStream: dlq,
		Logger:                logger,
	}
}

func (s StoryGenerationJobCompletion) OnSuccess(ctx context.Context, job domain.Job, MessageID string) error {
	log := s.Logger.With(
		"service.name", "story-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("completion of story job on success condition started", "event.type", []string{"start"})

	if err := s.TaskStreamHandler.Ack(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", s.StoryStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())

		return err
	}

	if err := s.TaskStreamHandler.Delete(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", s.StoryStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := s.TaskStreamHandler.Add(ctx, job, s.NotificationStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to add the job to %s", s.NotificationStream),
			"event.action", "add_email_notification_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	if err := s.StoryRepo.UpdateStoryJob(ctx, job.StoryID, "completed"); err != nil {
		log.Error(
			"failed to update story job status",
			"event.action", "update_story_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	log.Info(
		"completion of story job on success condition finished succussfully",
		"event.type", []string{"end"},
		"event.outcome", "success")

	return nil
}

func (e EmailNotificationJobCompletion) OnSuccess(ctx context.Context, job domain.Job, MessageID string) error {
	log := e.Logger.With(
		"service.name", "email-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("completion of email job on success condition started", "event.type", []string{"start"})

	if err := e.TaskStreamHandler.Ack(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", e.EmailStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.TaskStreamHandler.Delete(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", e.EmailStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	if err := e.StoryRepo.UpdateEmailJob(ctx, job.StoryID, job.UserID, "completed"); err != nil {
		log.Error(
			"failed to update email job status",
			"event.action", "update_email_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	log.Info(
		"completion of email job on success condition finished successfully",
		"event.type", []string{"end"},
		"event.outcome", "success")
	return nil
}

func (s StoryGenerationJobCompletion) OnFailure(ctx context.Context, job domain.Job, MessageID string) error {
	log := s.Logger.With(
		"service.name", "story-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("completion of story job on failure condition started", "event.type", []string{"start"})

	job.RetryCounts++
	err := s.TaskStreamHandler.ScheduleRetry(ctx, job, s.StoryStream)
	if err != nil {
		log.Error(
			"failed to schedule for retry",
			"event.action", "schedule_retry",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	if err := s.TaskStreamHandler.Ack(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", s.StoryStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := s.TaskStreamHandler.Delete(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", s.StoryStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := s.StoryRepo.UpdateStoryJob(ctx, job.StoryID, "processing"); err != nil {
		log.Error(
			"failed to update the story job status",
			"event.action", "update_story_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	log.Info(
		"completion of story job on failure condition finished successfully",
		"event.type", []string{"end"},
		"event.outcome", "success")

	return nil
}

func (e EmailNotificationJobCompletion) OnFailure(ctx context.Context, job domain.Job, MessageID string) error {
	log := e.Logger.With(
		"service.name", "email-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("completion of story job on failure condition started", "event.type", []string{"start"})

	job.RetryCounts++
	if err := e.TaskStreamHandler.ScheduleRetry(ctx, job, e.EmailStream); err != nil {
		log.Error(
			"failed to schedule for retry",
			"event.action", "schedule_retry",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	if err := e.TaskStreamHandler.Ack(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", e.EmailStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.TaskStreamHandler.Delete(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", e.EmailStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.StoryRepo.UpdateEmailJob(ctx, job.StoryID, job.UserID, "processing"); err != nil {
		log.Error(
			"failed to update email job status",
			"event.action", "update_email_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	log.Info(
		"completion of email job on failure condition finished successfully",
		"event.type", []string{"end"},
		"event.outcome", "success")

	return nil
}

func (s StoryGenerationJobCompletion) SendToDQL(ctx context.Context, job domain.Job, MessageID string) error {

	log := s.Logger.With(
		"service.name", "story-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("sending the story job to dlq started", "event.type", []string{"start"})

	if err := s.TaskStreamHandler.Add(ctx, job, s.DeadLetterQueueStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to add the job to %s", s.DeadLetterQueueStream),
			"event.action", "add_job_to_dlq",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	if err := s.TaskStreamHandler.Ack(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", s.StoryStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := s.TaskStreamHandler.Delete(ctx, MessageID, s.StoryStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", s.StoryStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := s.StoryRepo.UpdateStoryJob(ctx, job.StoryID, "failed"); err != nil {
		log.Error(
			"failed to update story job status",
			"event.action", "update_story_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	log.Info(
		"sending the story job to dlq finished succussfully",
		"event.type", []string{"end"},
		"event.outcome", "success")

	return nil
}

func (e EmailNotificationJobCompletion) SendToDQL(ctx context.Context, job domain.Job, MessageID string) error {
	log := e.Logger.With(
		"service.name", "email-workers",
		"http.request.id", job.RequestID,
		"stream.message.id", MessageID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})
	log.Info("sending the email job to dlq started", "event.type", []string{"start"})

	if err := e.TaskStreamHandler.Add(ctx, job, e.DeadLetterQueueStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to add the job to %s", e.DeadLetterQueueStream),
			"event.action", "add_job_to_dlq",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.TaskStreamHandler.Ack(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to acknowledge the message from %s", e.EmailStream),
			"event.action", "ack_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.TaskStreamHandler.Delete(ctx, MessageID, e.EmailStream); err != nil {
		log.Error(
			fmt.Sprintf("failed to delete the message from %s", e.EmailStream),
			"event.action", "delete_message",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	if err := e.StoryRepo.UpdateEmailJob(ctx, job.StoryID, job.UserID, "failed"); err != nil {
		log.Error(
			"failed to update email job status",
			"event.action", "update_email_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	log.Info(
		"sending the email job to dlq finished succussfully",
		"event.type", []string{"end"},
		"event.outcome", "success")

	return nil
}
