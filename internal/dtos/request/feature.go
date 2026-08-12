package request

import (
	"strings"
	"time"

	"novelhub/pkg/constants"
)

type GetUserCollectionsDto struct {
	Limit  int64  `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor string `json:"cursor,omitempty" query:"cursor" validate:"omitempty,readlist_cursor"`
}

func (d *GetUserCollectionsDto) GetLimit() int64 {
	if d.Limit <= 0 {
		return 50
	}
	if d.Limit > constants.MaxPaginationLimit {
		return constants.MaxPaginationLimit
	}
	return d.Limit
}

func (d *GetUserCollectionsDto) ParseCursor() (*time.Time, string) {
	if d.Cursor == "" {
		return nil, ""
	}
	parts := strings.SplitN(d.Cursor, "|", 2)
	if len(parts) == 2 {
		if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			return &t, parts[1]
		}
	} else if t, err := time.Parse(time.RFC3339Nano, d.Cursor); err == nil {
		return &t, ""
	}
	return nil, ""
}

type GetHighlightsQueryDto struct {
	ChapterID string `json:"chapter_id" query:"chapter_id" validate:"required"`
}

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
	EndIndex    int     `json:"end_index" validate:"gte=0"`
	Color       string  `json:"color" validate:"required"`
	Note        *string `json:"note,omitempty"`
	CfiRange    *string `json:"cfi_range,omitempty"`
}

type UpdateHighlightNoteDto struct {
	Note  *string `json:"note,omitempty"`
	Color string  `json:"color" validate:"required"`
}

type RecordReadingSessionDto struct {
	BookID      string `json:"book_id" validate:"required,uuid"`
	Duration    int64  `json:"duration" validate:"required,min=1"`
	Words       int64  `json:"words" validate:"min=0"`
	SessionDate string `json:"session_date,omitempty" validate:"omitempty,datetime=2006-01-02"`
}

type UpsertReadingGoalDto struct {
	TargetWordsPerDay  int64 `json:"target_words_per_day" validate:"required,min=1,max=1000000"`
	TargetBooksPerYear int64 `json:"target_books_per_year" validate:"required,min=1,max=10000"`
}

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

type SmartFilterRuleItemDto struct {
	Field    string `json:"field" validate:"required,oneof=status format rating_gte author_id series_id tag_id"`
	Operator string `json:"operator" validate:"required,oneof=eq gte lte"`
	Value    string `json:"value" validate:"required"`
}

type UpsertSmartFilterDto struct {
	Name            string                   `json:"name" validate:"required,min=1,max=100"`
	Rules           []SmartFilterRuleItemDto `json:"rules" validate:"required,dive"`
	IsPinnedSidebar bool                     `json:"is_pinned_sidebar"`
	IsPinnedHome    bool                     `json:"is_pinned_home"`
}

type ReorderHomeShelfItemDto struct {
	ID       string `json:"id" validate:"required"`
	Position int64  `json:"position" validate:"gte=0"`
}

type ReorderHomeShelvesDto struct {
	Shelves []ReorderHomeShelfItemDto `json:"shelves" validate:"required,dive"`
}
