package domain

import (
	"context"
)

type Story struct {
	FileName string
	UserID   int
	Story    string
}

type UploadStory struct {
	UserID int
	Story  string
}

type StoryRepository interface {
	SaveStoryInfo(ctx context.Context, s *Story) (int, error)
	UploadStory(ctx context.Context, s *UploadStory) error
	SaveStoryJob(ctx context.Context, storyID int, status string) (int, error)
	SaveEmailJob(ctx context.Context, storyID, userID int, status string) (int, error)
	UpdateStoryJob(ctx context.Context, storyID int, status string) error
	UpdateEmailJob(ctx context.Context, storyID int, userID int, status string) error
}

type GenerateStoryRepository interface {
	GenerateStory(ctx context.Context, preferences string) (string, error)
}
