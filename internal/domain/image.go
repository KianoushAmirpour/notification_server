package domain

const ImageGenerationPrompt = ""

type Image struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	UserID   int    `json:"user_id"`
	Url      string `json:"url"`
	Ratio    string `json:"ratio"`
	Size     int    `json:"size"`
	Status   string `json:"status"`
}

type RequestedImage struct {
	Description string `json:"description" validate:"required"`
}
