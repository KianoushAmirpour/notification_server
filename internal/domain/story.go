package domain

import "context"

type Story struct {
	FileName string
	UserID   int
	Story    string
	Status   string
}

type UploadStory struct {
	UserID int
	Story  string
}

type StoryRepository interface {
	Save(ctx context.Context, s *Story) error
	Upload(ctx context.Context, s *UploadStory) error
}

type GenerateStoryRepository interface {
	GenerateStory(ctx context.Context, preferences string) (string, error)
}
