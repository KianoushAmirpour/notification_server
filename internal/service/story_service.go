package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
)

type StoryGenerationService struct {
	Users  repository.UserRepository
	Images repository.ImageGeneration
}

func NewStoryGenerationService(users repository.UserRepository, images repository.ImageGeneration) *StoryGenerationService {
	return &StoryGenerationService{Users: users, Images: images}
}

func (i *StoryGenerationService) GenerateStory(ctx context.Context, userid int) (*domain.StoryRequestResponse, *domain.APIError) {

	u, err := i.Users.GetUserByID(ctx, userid)
	if err != nil {
		return nil, domain.NewAPIError(err, http.StatusNotFound)
	}

	result, err := i.Images.GenerateStory(ctx, u.Preferences)
	if err != nil {
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	return &domain.StoryRequestResponse{Message: fmt.Sprintf("Your story is ready. %s", result.Text())}, nil

	// create a job
	// put the job in queue.
	// workers must be up and running, to process the job
	/// generate and row in db with every thing related to date with status = pending
	// complete the database with status sompleted
	// email notifier must be triggered with an email
	// send an email with the url to user and marks the notification sent

	// if with channels try to persist the request in case of failure or retry

}
