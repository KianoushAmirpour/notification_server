package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type StoryServiceResponse struct {
	Message string `json:"message"`
}

type StoryGenerationService struct {
	UserRepo   domain.UserRepository
	StoryRepo  domain.StoryRepository
	AI         domain.GenerateStoryRepository
	WorkerPool domain.StoryWorkerPool
	Logger     domain.LoggingRepository
}

func NewStoryGenerationService(
	userrepo domain.UserRepository,
	storyrepo domain.StoryRepository,
	ai domain.GenerateStoryRepository,
	pool domain.StoryWorkerPool,
	logger domain.LoggingRepository,
) *StoryGenerationService {
	return &StoryGenerationService{UserRepo: userrepo, StoryRepo: storyrepo, AI: ai, WorkerPool: pool, Logger: logger}
}

func (s *StoryGenerationService) GenerateStory(ctx context.Context, userid int) (*StoryServiceResponse, error) {
	start := time.Now()
	log := s.Logger.With("service", "story_generation", "user_id", userid)

	u, err := s.UserRepo.GetUserByID(ctx, userid)
	if err != nil {
		log.Error("story_generation_failed_get_user_by_id", "reason", err.Error())
		return nil, err
	}

	keywords := strings.Join(u.Preferences, "-")

	story := &domain.Story{FileName: fmt.Sprintf("story-%s", keywords), UserID: userid, Story: "", Status: "pending"}

	err = s.StoryRepo.Save(ctx, story)
	if err != nil {
		log.Error("story_generation_failed_store_story_metadata", "reason", err.Error())
		return nil, err
	}

	storyGenJob := domain.GenerateStoryJob{
		UserInfo:  domain.StoryUserInfo{UserID: userid, UserPreferences: keywords},
		AI:        s.AI,
		UserRepo:  s.UserRepo,
		StoryRepo: s.StoryRepo,
		Logger:    s.Logger,
	}

	s.WorkerPool.Submit(storyGenJob)
	log.Info("push_to_story_generation_jobqueue")
	log.Info("story_generation_successfully", "duration_us", int(time.Since(start).Microseconds()))
	return &StoryServiceResponse{Message: "Your story is being generated. You wull be notified by email"}, nil
}
