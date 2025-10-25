package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

func (s *StoryGenerationService) GenerateStory(ctx context.Context, userid int, logger *slog.Logger) (*domain.StoryRequestResponse, *domain.DomainError) {
	start := time.Now()
	log := logger.With(slog.String("service", "story_generateion"), slog.Int("user_id", userid))

	u, err := s.Users.GetUserByID(ctx, userid)
	if err != nil {
		log.Error("story_generateion_failed_get_user_by_id", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "user not found", err)
	}

	keywords := strings.Join(u.Preferences, "_")

	story := &domain.Story{FileName: fmt.Sprintf("story_%s", keywords), UserID: userid, Story: "", Status: "pending"}

	err = s.Users.SaveStoryMetaData(ctx, story)
	if err != nil {
		log.Error("story_generateion_failed_store_story_metadata", slog.String("reason", err.Error()))
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to save save story meta data", err)
	}

	job := &adapters.GenerateStoryJob{
		UserID:          userid,
		UserPreferences: keywords,
		StoryGenerator:  s.StoryGen,
		UserRepo:        s.Users,
		Logger:          logger,
	}

	s.WorkerPool.Submit(job)
	log.Info("push_to_jobqueue")
	log.Info("story_generateion_successful", slog.Int("duration_us", int(time.Since(start).Microseconds())))
	return &domain.StoryRequestResponse{Message: "Your story is being generated"}, nil
}
