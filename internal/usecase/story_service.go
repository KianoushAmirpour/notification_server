package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/observability"
)

type StoryServiceResponse struct {
	Message string `json:"message"`
}

type StorySchedulerService struct {
	UserRepo  domain.UserRepository
	StoryRepo domain.StoryRepository
	// WorkerPool domain.StoryWorkerPool
	TaskStreamHandler domain.StreamTaskHandler
	Logger            domain.LoggingRepository
	Stream            string
}

func NewStorySchedulerService(
	userrepo domain.UserRepository,
	storyrepo domain.StoryRepository,
	taskStreamHandler domain.StreamTaskHandler,
	logger domain.LoggingRepository,
	stream string,
) *StorySchedulerService {
	return &StorySchedulerService{UserRepo: userrepo, StoryRepo: storyrepo, TaskStreamHandler: taskStreamHandler, Logger: logger, Stream: stream}
}

func (s *StorySchedulerService) ScheduleStoryGeneration(ctx context.Context, userid int) (*StoryServiceResponse, error) {
	reqID := observability.GetRequestID(ctx)
	log := s.Logger.With("service.name", "story_scheduler", "http.request.id", reqID, "user.id", userid, "event.category", []string{"web"})

	log.Info("scheduling for story generation started", "event.type", []string{"start"})

	user, err := s.UserRepo.GetUserByID(ctx, userid)
	if err != nil {
		log.Error(
			"failed to find user by id",
			"event.action", "get_user_by_id",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	userp, err := s.UserRepo.GetUserPreferencesByID(ctx, userid)
	if err != nil {
		log.Error(
			"failed to find user preferences by id",
			"event.action", "get_user_preferences_by_id",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	keywords := strings.Join(userp.UserPreferences, "-")

	story := &domain.Story{FileName: fmt.Sprintf("story-%s", keywords), UserID: userid, Story: ""}

	story_id, err := s.StoryRepo.SaveStoryInfo(ctx, story)
	if err != nil {
		log.Error(
			"failed to save story info",
			"event.action", "save_story_info",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	storyJobID, err := s.StoryRepo.SaveStoryJob(ctx, story_id, "pending")
	if err != nil {
		log.Error(
			"failed to save story job",
			"event.action", "save_story_job",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}

	storyGenerationJob := domain.Job{
		JobID:           storyJobID,
		UserID:          userid,
		StoryID:         story_id,
		UserEmail:       user.Email,
		UserPreferences: keywords,
		RetryCounts:     0,
		RequestID:       reqID}

	err = s.TaskStreamHandler.Add(ctx, storyGenerationJob, s.Stream)
	if err != nil {
		log.Error(
			fmt.Sprintf("failed to add job to %s stream", s.Stream),
			"story.id", story_id,
			"story.job.id", storyJobID,
			"event.action", "add_story_generation_job_to_stream",
			"event.type", []string{"error", "end"},
			"event.outcome", "failed",
			"error.message", err.Error())
		return nil, err
	}
	log.Info(
		fmt.Sprintf("job succussful added to %s stream", s.Stream),
		"story.id", story_id,
		"story.job.id", storyJobID,
		"event.type", []string{"end", "creation"},
		"event.outcome", "success")

	return &StoryServiceResponse{Message: "Your story is being generated. You will be notified by email"}, nil
}
