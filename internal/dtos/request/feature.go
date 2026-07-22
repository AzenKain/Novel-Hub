package request

type CreateCollectionDto struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type UpdateCollectionDto struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type RecordReadingActivityDto struct {
	BookID          string   `json:"bookId" validate:"required"`
	FileID          string   `json:"fileId"`
	ChapterID       string   `json:"chapterId" validate:"required"`
	ChapterTitle    string   `json:"chapterTitle"`
	ChapterIndex    int64    `json:"chapterIndex"`
	ProgressPercent *float64 `json:"progressPercent"`
	LocationCfi     *string  `json:"locationCfi"`
	LocationType    *string  `json:"locationType"`
	EventType       string   `json:"eventType"`
}

type RecordShareDto struct {
	ClientID string `json:"clientId"`
}

type SetBookmarkDto struct {
	Bookmarked bool `json:"bookmarked"`
}

type UpsertBookReviewDto struct {
	Rating int64  `json:"rating" validate:"required,min=1,max=5"`
	Review string `json:"review" validate:"omitempty,max=4000"`
}

type CollectionBookDto struct {
	BookID string `json:"bookId" validate:"required"`
}

type CreateHighlightDto struct {
	BookID      string  `json:"bookId" validate:"required"`
	ChapterID   string  `json:"chapterId" validate:"required"`
	TextContent string  `json:"textContent" validate:"required"`
	StartIndex  int     `json:"startIndex" validate:"required,min=0"`
	EndIndex    int     `json:"endIndex" validate:"required,gtfield=StartIndex"`
	Color       string  `json:"color" validate:"required"`
	Note        *string `json:"note,omitempty"`
}

type UpdateHighlightNoteDto struct {
	Note  *string `json:"note,omitempty"`
	Color string  `json:"color" validate:"required"`
}

type RecordReadingSessionDto struct {
	BookID   string `json:"bookId" validate:"required"`
	Duration int64  `json:"duration" validate:"required,min=1"`
	Words    int64  `json:"words" validate:"min=0"`
}

