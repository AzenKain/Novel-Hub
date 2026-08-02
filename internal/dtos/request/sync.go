package request

type ProgressSyncDto struct {
	BookID          string   `json:"book_id" validate:"required"`
	ProgressPercent *float64 `json:"progress_percent"`
	LocationCfi     *string  `json:"location_cfi"`
	LocationType    *string  `json:"location_type"`
	ChapterTitle    *string  `json:"chapter_title"`
	ChapterIndex    *int64   `json:"chapter_index"`
	Timestamp       *int64   `json:"timestamp"`
	Device          *string  `json:"device"`
}

type KosyncPushProgressDto struct {
	Document   string  `json:"document"`
	Progress   float64 `json:"progress"`
	Percentage float64 `json:"percentage"`
	Timestamp  int64   `json:"timestamp"`
	Device     string  `json:"device"`
}
