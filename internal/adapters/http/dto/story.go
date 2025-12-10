package dto

type Story struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	UserID   int    `json:"user_id"`
	Story    string `json:"story"`
	Status   string `json:"status"`
}

type UploadStory struct {
	UserID int    `json:"user_id"`
	Story  string `json:"story"`
}
