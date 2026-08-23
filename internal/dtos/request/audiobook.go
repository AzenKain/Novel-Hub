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


type MergeAudioSegment struct {
	FileID   string  `json:"file_id" validate:"required"`
	StartSec float64 `json:"start_sec" validate:"gte=0"`
	EndSec   float64 `json:"end_sec" validate:"required,gtfield=StartSec"`
	Gain     float64 `json:"gain" validate:"gte=0,lte=5"` // 1.0 = original, 0-5x range
}

type MergeAudioDto struct {
	Title    string              `json:"title" validate:"required,min=1,max=200"`
	Segments []MergeAudioSegment `json:"segments" validate:"required,min=2,max=100,dive"`
}