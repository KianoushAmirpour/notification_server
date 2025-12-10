package usecase

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

func RunEmailJob(ctx context.Context, emailjob domain.EmailNotificationJob) error {

	log := emailjob.Logger.With("service", "run_email_notification_job", "user_email", emailjob.UserEmail)
	err := emailjob.Mailer.SendNotificationEmail(emailjob.UserEmail)
	if err != nil {
		log.Error("email_notification_failed", "email", emailjob.UserEmail, "reason", err)
		return err
	}
	log.Info("notification_sent_successfully")
	return nil
}
