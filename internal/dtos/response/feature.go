package response

import (
	"time"

	"novelhub/internal/dtos/request"
)

type HighlightResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	BookID      string    `json:"book_id"`
	ChapterID   string    `json:"chapter_id"`
	TextContent string    `json:"text_content"`
	StartIndex  int64     `json:"start_index"`
	EndIndex    int64     `json:"end_index"`
	Color       string    `json:"color"`
	Note        *string   `json:"note,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReadingSessionResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	BookID          string    `json:"book_id"`
	DurationSeconds int64     `json:"duration_seconds"`
	WordsRead       int64     `json:"words_read"`
	Date            time.Time `json:"date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ReadingGoalResponse struct {
	UserID             string    `json:"user_id"`
	TargetWordsPerDay  int64     `json:"target_words_per_day"`
	TargetBooksPerYear int64     `json:"target_books_per_year"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Rule is the parsed rule, never the stored JSON string — clients build a library
// URL straight from these fields. Reusing the request DTO keeps the shape defined
// in exactly one place; request imports nothing from here, so there is no cycle.
type SmartCollectionResponse struct {
	ID        string                         `json:"id"`
	UserID    string                         `json:"user_id"`
	Name      string                         `json:"name"`
	Rule      request.SmartCollectionRuleDto `json:"rule"`
	CreatedAt time.Time                      `json:"created_at"`
	UpdatedAt time.Time                      `json:"updated_at"`
}

type ReadingHeatmapResponse struct {
	Date            time.Time `json:"date"`
	DurationSeconds int64     `json:"duration_seconds"`
	WordsRead       int64     `json:"words_read"`
}

type CollectionResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LibraryStatsResponse struct {
	TotalBooks    int64 `json:"total_books"`
	NeedReview    int64 `json:"need_review"`
	SeriesTracked int64 `json:"series_tracked"`
}

type BookReadStatsResponse struct {
	BookID             string     `json:"book_id"`
	TotalOpenCount     int64      `json:"total_open_count"`
	QualifiedReadCount int64      `json:"qualified_read_count"`
	LastOpenedAt       *time.Time `json:"last_opened_at,omitempty"`
	LastCountedAt      *time.Time `json:"last_counted_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

type BookDownloadStatsResponse struct {
	BookID             string     `json:"book_id"`
	TotalDownloadCount int64      `json:"total_download_count"`
	LastDownloadedAt   *time.Time `json:"last_downloaded_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

type BookmarkResponse struct {
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	CreatedAt time.Time `json:"created_at"`
}

type BookRatingSummaryResponse struct {
	BookID        string  `json:"book_id"`
	RatingCount   int64   `json:"rating_count"`
	AverageRating float64 `json:"average_rating"`
}

type BookSocialStatsResponse struct {
	BookID        string     `json:"book_id"`
	BookmarkCount int64      `json:"bookmark_count"`
	RatingCount   int64      `json:"rating_count"`
	AverageRating float64    `json:"average_rating"`
	ShareCount    int64      `json:"share_count"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type BookEngagementStatsResponse struct {
	BookID        string                     `json:"book_id"`
	SocialStats   *BookSocialStatsResponse   `json:"social_stats,omitempty"`
	DownloadStats *BookDownloadStatsResponse `json:"download_stats,omitempty"`
	ReadStats     *BookReadStatsResponse     `json:"read_stats,omitempty"`
}

type BookReviewResponse struct {
	UserID    string     `json:"user_id"`
	BookID    string     `json:"book_id"`
	Rating    int64      `json:"rating"`
	Review    *string    `json:"review,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	UserName  string     `json:"user_name,omitempty"`
	UserEmail string     `json:"user_email,omitempty"`
	BookTitle string     `json:"book_title,omitempty"`
}

type BookUserStateResponse struct {
	BookID        string                     `json:"book_id"`
	Bookmarked    bool                       `json:"bookmarked"`
	MyReview      *BookReviewResponse        `json:"my_review,omitempty"`
	RatingSummary *BookRatingSummaryResponse `json:"rating_summary,omitempty"`
	SocialStats   *BookSocialStatsResponse   `json:"social_stats,omitempty"`
	DownloadStats *BookDownloadStatsResponse `json:"download_stats,omitempty"`
	ReadStats     *BookReadStatsResponse     `json:"read_stats,omitempty"`
	Collections   []string                   `json:"collections"`
}

type ReadingHistoryResponse struct {
	UserID          string    `json:"user_id"`
	BookID          string    `json:"book_id"`
	FileID          *string   `json:"file_id,omitempty"`
	ChapterID       string    `json:"chapter_id"`
	ProgressPercent *float64  `json:"progress_percent"`
	UpdatedAt       time.Time `json:"updated_at"`
	BookTitle       string    `json:"book_title"`
	BookCoverURL    *string   `json:"book_cover_url"`
	ChapterTitle    string    `json:"chapter_title"`
	ChapterIndex    int64     `json:"chapter_index"`
}

type ReadingProgressResponse struct {
	UserID             string     `json:"user_id"`
	BookID             string     `json:"book_id"`
	FileID             *string    `json:"file_id,omitempty"`
	ChapterID          string     `json:"chapter_id"`
	ChapterTitle       string     `json:"chapter_title"`
	ChapterIndex       int64      `json:"chapter_index"`
	ProgressPercent    *float64   `json:"progress_percent,omitempty"`
	LocationCfi        *string    `json:"location_cfi,omitempty"`
	LocationType       *string    `json:"location_type,omitempty"`
	OpenedCount        int64      `json:"opened_count"`
	QualifiedReadCount int64      `json:"qualified_read_count"`
	LastOpenedAt       *time.Time `json:"last_opened_at,omitempty"`
	LastCountedAt      *time.Time `json:"last_counted_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

type ReadingActivityResponse struct {
	Progress        *ReadingProgressResponse `json:"progress,omitempty"`
	Stats           *BookReadStatsResponse   `json:"stats,omitempty"`
	Counted         bool                     `json:"counted"`
	CooldownSeconds int64                    `json:"cooldown_seconds"`
}
