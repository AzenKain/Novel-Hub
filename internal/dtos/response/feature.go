package response

import "time"

type HighlightResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	BookID      string    `json:"bookId"`
	ChapterID   string    `json:"chapterId"`
	TextContent string    `json:"textContent"`
	StartIndex  int64     `json:"startIndex"`
	EndIndex    int64     `json:"endIndex"`
	Color       string    `json:"color"`
	Note        *string   `json:"note,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ReadingSessionResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	BookID          string    `json:"bookId"`
	DurationSeconds int64     `json:"durationSeconds"`
	WordsRead       int64     `json:"wordsRead"`
	Date            time.Time `json:"date"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ReadingHeatmapResponse struct {
	Date            time.Time `json:"date"`
	DurationSeconds int64     `json:"durationSeconds"`
	WordsRead       int64     `json:"wordsRead"`
}

type CollectionResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LibraryStatsResponse struct {
	TotalBooks    int64 `json:"totalBooks"`
	NeedReview    int64 `json:"needReview"`
	SeriesTracked int64 `json:"seriesTracked"`
}

type BookReadStatsResponse struct {
	BookID             string     `json:"bookId"`
	TotalOpenCount     int64      `json:"totalOpenCount"`
	QualifiedReadCount int64      `json:"qualifiedReadCount"`
	LastOpenedAt       *time.Time `json:"lastOpenedAt,omitempty"`
	LastCountedAt      *time.Time `json:"lastCountedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type BookDownloadStatsResponse struct {
	BookID             string     `json:"bookId"`
	TotalDownloadCount int64      `json:"totalDownloadCount"`
	LastDownloadedAt   *time.Time `json:"lastDownloadedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type BookmarkResponse struct {
	UserID    string    `json:"userId"`
	BookID    string    `json:"bookId"`
	CreatedAt time.Time `json:"createdAt"`
}

type BookRatingSummaryResponse struct {
	BookID        string  `json:"bookId"`
	RatingCount   int64   `json:"ratingCount"`
	AverageRating float64 `json:"averageRating"`
}

type BookSocialStatsResponse struct {
	BookID        string     `json:"bookId"`
	BookmarkCount int64      `json:"bookmarkCount"`
	RatingCount   int64      `json:"ratingCount"`
	AverageRating float64    `json:"averageRating"`
	ShareCount    int64      `json:"shareCount"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

type BookEngagementStatsResponse struct {
	BookID        string                     `json:"bookId"`
	SocialStats   *BookSocialStatsResponse   `json:"socialStats,omitempty"`
	DownloadStats *BookDownloadStatsResponse `json:"downloadStats,omitempty"`
	ReadStats     *BookReadStatsResponse     `json:"readStats,omitempty"`
}

type BookReviewResponse struct {
	UserID    string     `json:"userId"`
	BookID    string     `json:"bookId"`
	Rating    int64      `json:"rating"`
	Review    *string    `json:"review,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UserName  string     `json:"userName,omitempty"`
	UserEmail string     `json:"userEmail,omitempty"`
	BookTitle string     `json:"bookTitle,omitempty"`
}

type BookUserStateResponse struct {
	BookID        string                     `json:"bookId"`
	Bookmarked    bool                       `json:"bookmarked"`
	MyReview      *BookReviewResponse        `json:"myReview,omitempty"`
	RatingSummary *BookRatingSummaryResponse `json:"ratingSummary,omitempty"`
	SocialStats   *BookSocialStatsResponse   `json:"socialStats,omitempty"`
	DownloadStats *BookDownloadStatsResponse `json:"downloadStats,omitempty"`
	ReadStats     *BookReadStatsResponse     `json:"readStats,omitempty"`
	Collections   []string                   `json:"collections"`
}

type ReadingHistoryResponse struct {
	UserID          string    `json:"userId"`
	BookID          string    `json:"bookId"`
	FileID          *string   `json:"fileId,omitempty"`
	ChapterID       string    `json:"chapterId"`
	ProgressPercent *float64  `json:"progressPercent"`
	LocationCfi     *string   `json:"locationCfi,omitempty"`
	LocationType    *string   `json:"locationType,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
	BookTitle       string    `json:"bookTitle"`
	BookCoverURL    *string   `json:"bookCoverUrl"`
	ChapterTitle    string    `json:"chapterTitle"`
	ChapterIndex    int64     `json:"chapterIndex"`
}

type ReadingProgressResponse struct {
	UserID             string     `json:"userId"`
	BookID             string     `json:"bookId"`
	FileID             *string    `json:"fileId,omitempty"`
	ChapterID          string     `json:"chapterId"`
	ChapterTitle       string     `json:"chapterTitle"`
	ChapterIndex       int64      `json:"chapterIndex"`
	ProgressPercent    *float64   `json:"progressPercent,omitempty"`
	LocationCfi        *string    `json:"locationCfi,omitempty"`
	LocationType       *string    `json:"locationType,omitempty"`
	OpenedCount        int64      `json:"openedCount"`
	QualifiedReadCount int64      `json:"qualifiedReadCount"`
	LastOpenedAt       *time.Time `json:"lastOpenedAt,omitempty"`
	LastCountedAt      *time.Time `json:"lastCountedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type ReadingActivityResponse struct {
	Progress        *ReadingProgressResponse `json:"progress,omitempty"`
	Stats           *BookReadStatsResponse   `json:"stats,omitempty"`
	Counted         bool                     `json:"counted"`
	CooldownSeconds int64                    `json:"cooldownSeconds"`
}
