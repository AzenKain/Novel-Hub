package response

import "time"

type LibraryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LibraryUploadResultResponse struct {
	Uploaded int `json:"uploaded"`
	Total    int `json:"total"`
}
