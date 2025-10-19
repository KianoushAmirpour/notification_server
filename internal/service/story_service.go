package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/KianoushAmirpour/notification_server/internal/adapters"
	"github.com/KianoushAmirpour/notification_server/internal/adapters/ai"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
)

type StoryGenerationService struct {
	Users      repository.UserRepository
	StoryGen   *ai.GemeniClient
	WorkerPool repository.WorkerPool
}

func NewStoryGenerationService(users repository.UserRepository, storygen *ai.GemeniClient, pool repository.WorkerPool) *StoryGenerationService {
	return &StoryGenerationService{Users: users, StoryGen: storygen, WorkerPool: pool}
}

func (s *StoryGenerationService) GenerateStory(ctx context.Context, userid int) (*domain.StoryRequestResponse, *domain.APIError) {

	u, err := s.Users.GetUserByID(ctx, userid)
	if err != nil {
		return nil, domain.NewAPIError(err, http.StatusNotFound)
	}

	keywords := strings.Join(u.Preferences, "_")

	story := &domain.Story{FileName: fmt.Sprintf("story_%s", keywords), UserID: userid, Story: "", Status: "pending"}

	err = s.Users.SaveStoryMetaData(ctx, story)
	if err != nil {
		return nil, domain.NewAPIError(err, http.StatusInternalServerError)
	}

	job := &adapters.GenerateStoryJob{
		UserID:          userid,
		UserPreferences: keywords,
		StoryGenerator:  s.StoryGen,
		UserRepo:        s.Users,
	}

	s.WorkerPool.Submit(job)

	return &domain.StoryRequestResponse{Message: "Your story is being generated"}, nil

	// complete the database with status sompleted
	// email notifier must be triggered with an email
	// send an email with the url to user and marks the notification sent

	// if with channels try to persist the request in case of failure or retry

}
