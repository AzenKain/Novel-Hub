package request

type UpsertAudiobookChapterDto struct {
	FileID       *string  `json:"file_id"`
	ChapterIndex int64    `json:"chapter_index" validate:"gte=0"`
	Title        string   `json:"title" validate:"required,min=1,max=200"`
	StartSec     float64  `json:"start_sec" validate:"gte=0"`
	EndSec       *float64 `json:"end_sec" validate:"omitempty,gte=0"`
}

type LookupAudiobookChaptersDto struct {
	ASIN string `json:"asin" validate:"required,min=9,max=20"`
}

type MergeAudioDto struct {
	Title   string   `json:"title" validate:"required,min=1,max=200"`
	FileIDs []string `json:"file_ids" validate:"required,min=2,max=100,dive,required"`
}