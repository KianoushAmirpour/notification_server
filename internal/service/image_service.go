package service

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type ImageGenerationService struct {
}

func NewImageGenerationService() *ImageGenerationService {
	return &ImageGenerationService{}
}

func (i *ImageGenerationService) GenerateImage(ctx context.Context, req domain.RequestedImage) {

	// retrieve user id and prefernces.
	// create a job
	// put the job in queue.
	// workers must be up and running, to process the job
	/// generate and row in db with every thing related to date with status = pending
	// complete the database with status sompleted
	// email notifier must be triggered with an email
	// send an email with the url to user and marks the notification sent

	// if with channels try to persist the request in case of failure or retry

}
