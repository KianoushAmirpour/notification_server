package usecase

import (
	"context"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type StoryGenerationService struct {
	UserRepo  domain.UserRepository
	StoryRepo domain.StoryRepository
	AI        domain.GenerateStoryRepository
	Logger    domain.LoggingRepository
}

func NewStoryGenerationService(
	userrepo domain.UserRepository,
	storyrepo domain.StoryRepository,
	ai domain.GenerateStoryRepository,
	logger domain.LoggingRepository,
) *StoryGenerationService {
	return &StoryGenerationService{UserRepo: userrepo, StoryRepo: storyrepo, AI: ai, Logger: logger}
}

func (s *StoryGenerationService) Execute(ctx context.Context, job domain.Job) error {
	log := s.Logger.With(
		"service.name", "story_generator",
		"http.request.id", job.RequestID,
		"user.id", job.UserID,
		"story.id", job.StoryID,
		"story.job.id", job.JobID,
		"event.category", []string{"process"})

	log.Info("story generatoin started", "event.type", []string{"start"})

	aiStartTime := time.Now()
	output, err := s.AI.GenerateStory(ctx, job.UserPreferences)
	aiDurationTime := time.Since(aiStartTime)
	if err != nil {
		log.Error(
			"failed to generate story by ai service",
			"event.action", "generate_story_by_ai",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error(),
			"event.duration", aiDurationTime.Nanoseconds())
		return err
	}

	story := domain.UploadStory{UserID: job.UserID, Story: output}
	err = s.StoryRepo.UploadStory(ctx, &story)
	if err != nil {
		log.Error(
			"failed to upload story to database",
			"event.action", "upload_story",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	_, err = s.StoryRepo.SaveEmailJob(ctx, job.StoryID, job.UserID, "pending")
	if err != nil {
		log.Error(
			"failed to save email job",
			"event.action", "save_email_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return err
	}

	log.Info(
		"story generated succussfully",
		"event.type", []string{"end", "creation"},
		"event.outcome", "success",
		"event.duration", "event.duration", aiDurationTime.Nanoseconds())

	return nil

}
