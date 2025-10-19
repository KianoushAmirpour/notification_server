package domain

const StoryGenerationThemePrompt = "fancy funny theme"

type Story struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	UserID   int    `json:"user_id"`
	Story    string `json:"story"`
	Status   string `json:"status"`
}

type StoryRequestResponse struct {
	Message string `json:"message"`
}

type UploadStory struct {
	UserID int    `json:"user_id"`
	Story  string `json:"story"`
}
