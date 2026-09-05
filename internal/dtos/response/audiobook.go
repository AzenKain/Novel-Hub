package response

import "time"

type AudiobookChapterResponse struct {
	ID           string    `json:"id"`
	BookID       string    `json:"book_id"`
	FileID       *string   `json:"file_id,omitempty"`
	ChapterIndex int64     `json:"chapter_index"`
	Title        string    `json:"title"`
	StartSec     float64   `json:"start_sec"`
	EndSec       *float64  `json:"end_sec,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
