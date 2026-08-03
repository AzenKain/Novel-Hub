package request

type CreateCollectionDto struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type UpdateCollectionDto struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type RecordReadingActivityDto struct {
	BookID          string   `json:"book_id" validate:"required"`
	FileID          string   `json:"file_id"`
	ChapterID       string   `json:"chapter_id" validate:"required"`
	ChapterTitle    string   `json:"chapter_title"`
	ChapterIndex    int64    `json:"chapter_index"`
	ProgressPercent *float64 `json:"progress_percent"`
	LocationCfi     *string  `json:"location_cfi"`
	LocationType    *string  `json:"location_type"`
	EventType       string   `json:"event_type"`
}

type RecordShareDto struct {
	ClientID string `json:"client_id"`
}

type SetBookmarkDto struct {
	Bookmarked bool `json:"bookmarked"`
}

type UpsertBookReviewDto struct {
	Rating int64  `json:"rating" validate:"required,min=1,max=5"`
	Review string `json:"review" validate:"omitempty,max=4000"`
}

type CollectionBookDto struct {
	BookID string `json:"book_id" validate:"required"`
}

type CreateHighlightDto struct {
	BookID      string  `json:"book_id" validate:"required"`
	ChapterID   string  `json:"chapter_id" validate:"required"`
	TextContent string  `json:"text_content" validate:"required"`
	StartIndex  int     `json:"start_index" validate:"gte=0"`
	EndIndex    int     `json:"end_index" validate:"gtfield=StartIndex"`
	Color       string  `json:"color" validate:"required"`
	Note        *string `json:"note,omitempty"`
}

type UpdateHighlightNoteDto struct {
	Note  *string `json:"note,omitempty"`
	Color string  `json:"color" validate:"required"`
}

type RecordReadingSessionDto struct {
	BookID   string `json:"book_id" validate:"required,uuid"`
	Duration int64  `json:"duration" validate:"required,min=1"`
	Words    int64  `json:"words" validate:"min=0"`
}

type UpsertReadingGoalDto struct {
	TargetWordsPerDay  int64 `json:"target_words_per_day" validate:"required,min=1,max=1000000"`
	TargetBooksPerYear int64 `json:"target_books_per_year" validate:"required,min=1,max=10000"`
}

// SmartCollectionRuleDto mirrors the filter fields of SearchBookDto. It is a
// closed struct on purpose: rule_json is read back out and replayed into a
// library URL, so this is a trust boundary — free-form JSON must not survive a
// round trip through the database.
type SmartCollectionRuleDto struct {
	Search     string `json:"search,omitempty" validate:"omitempty,max=200"`
	LibraryID  string `json:"library_id,omitempty" validate:"omitempty,max=200"`
	Nav        string `json:"nav,omitempty" validate:"omitempty,max=200"`
	Collection string `json:"collection,omitempty" validate:"omitempty,max=200"`
	Chip       string `json:"chip,omitempty" validate:"omitempty,max=200"`
	Facet      string `json:"facet,omitempty" validate:"omitempty,max=200"`
	FacetID    string `json:"facet_id,omitempty" validate:"omitempty,max=200"`
}

type UpsertSmartCollectionDto struct {
	Name string                 `json:"name" validate:"required,min=1,max=100"`
	Rule SmartCollectionRuleDto `json:"rule"`
}
