package usecase

import (
	"context"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

const UserEmailKey domain.ContextKey = "useremail"

func RunStoryJob(ctx context.Context, storygenjob domain.GenerateStoryJob) (string, error) {

	log := storygenjob.Logger.With("service", "run_story_job", "user_id", storygenjob.UserInfo.UserID, "user_preferences", storygenjob.UserInfo.UserPreferences)
	output, err := storygenjob.AI.GenerateStory(ctx, storygenjob.UserInfo.UserPreferences)
	if err != nil {
		log.Error("run_story_job_failed_generate_story_by_ai", "reason", err.Error())
		return "", err
	}

	story := domain.UploadStory{UserID: storygenjob.UserInfo.UserID, Story: output}
	err = storygenjob.StoryRepo.Upload(ctx, &story)
	if err != nil {
		log.Error("run_story_job_failed_upload_story_to_db", "reason", err.Error())
		return "", err
	}
	u, err := storygenjob.UserRepo.GetUserByID(ctx, storygenjob.UserInfo.UserID)
	if err != nil {
		log.Error("run_story_job_failed_get_user_by_id", "reason", err.Error())
		return "", err
	}
	// newctx := context.WithValue(ctx, UserEmailKey, u.Email)
	log.Info("run_story_job_successfully")
	return u.Email, nil
}
