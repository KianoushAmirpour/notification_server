package usecase

import (
	"context"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type EmailSenderService struct {
	Mailer domain.Mailer
	Logger domain.LoggingRepository
}

func NewEmailSenderService(mailer domain.Mailer, logger domain.LoggingRepository) *EmailSenderService {
	return &EmailSenderService{
		Mailer: mailer,
		Logger: logger,
	}
}
func (es *EmailSenderService) Execute(ctx context.Context, emailjob domain.Job) error {

	log := es.Logger.With(
		"service.name", "notification",
		"http.request.id", emailjob.RequestID,
		"user.id", emailjob.UserID,
		"story.id", emailjob.StoryID,
		"story.job.id", emailjob.JobID,
		"event.category", []string{"email"})

	start := time.Now()
	err := es.Mailer.SendNotificationEmail(emailjob.UserEmail)
	if err != nil {
		log.Error(
			"failed to send notification email to user",
			"event.action", "send_email_notification",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}
	log.Info(
		"notification email sent succussfully",
		"event.type", []string{"end"},
		"event.outcome", "success",
		"event.duration", int(time.Since(start).Nanoseconds()))
	return nil
}
