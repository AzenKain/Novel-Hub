package request

type ProgressSyncDto struct {
	BookID          string   `json:"bookId" validate:"required"`
	ProgressPercent *float64 `json:"progressPercent"`
	LocationCfi     *string  `json:"locationCfi"`
	LocationType    *string  `json:"locationType"`
	ChapterTitle    *string  `json:"chapterTitle"`
	ChapterIndex    *int64   `json:"chapterIndex"`
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
