package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type LibraryStatsEntity struct {
	TotalBooks    int64 `json:"total_books"`
	NeedReview    int64 `json:"need_review"`
	SeriesTracked int64 `json:"series_tracked"`
}

func (e *LibraryStatsEntity) FromSqlc(res sqlc.GetLibraryStatsRow) *LibraryStatsEntity {
	e.TotalBooks = res.TotalBooks
	e.NeedReview = res.NeedReview
	e.SeriesTracked = res.SeriesTracked
	return e
}

func (e *LibraryStatsEntity) ToResponse() *response.LibraryStatsResponse {
	if e == nil {
		return nil
	}
	return &response.LibraryStatsResponse{
		TotalBooks:    e.TotalBooks,
		NeedReview:    e.NeedReview,
		SeriesTracked: e.SeriesTracked,
	}
}

type BookReadStatsEntity struct {
	BookID             string     `json:"book_id"`
	TotalOpenCount     int64      `json:"total_open_count"`
	QualifiedReadCount int64      `json:"qualified_read_count"`
	LastOpenedAt       *time.Time `json:"last_opened_at,omitempty"`
	LastCountedAt      *time.Time `json:"last_counted_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

func (e *BookReadStatsEntity) FromSqlc(res sqlc.BookReadStat) *BookReadStatsEntity {
	e.BookID = res.BookID
	e.TotalOpenCount = res.TotalOpenCount
	e.QualifiedReadCount = res.QualifiedReadCount
	e.LastOpenedAt = convert.NullTimeToTimePtr(res.LastOpenedAt)
	e.LastCountedAt = convert.NullTimeToTimePtr(res.LastCountedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

func (e *BookReadStatsEntity) ToResponse() *response.BookReadStatsResponse {
	if e == nil {
		return nil
	}
	return &response.BookReadStatsResponse{
		BookID:             e.BookID,
		TotalOpenCount:     e.TotalOpenCount,
		QualifiedReadCount: e.QualifiedReadCount,
		LastOpenedAt:       e.LastOpenedAt,
		LastCountedAt:      e.LastCountedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

type BookDownloadStatsEntity struct {
	BookID             string     `json:"book_id"`
	TotalDownloadCount int64      `json:"total_download_count"`
	LastDownloadedAt   *time.Time `json:"last_downloaded_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

func (e *BookDownloadStatsEntity) FromSqlc(res sqlc.BookDownloadStat) *BookDownloadStatsEntity {
	e.BookID = res.BookID
	e.TotalDownloadCount = res.TotalDownloadCount
	e.LastDownloadedAt = convert.NullTimeToTimePtr(res.LastDownloadedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

func (e *BookDownloadStatsEntity) ToResponse() *response.BookDownloadStatsResponse {
	if e == nil {
		return nil
	}
	return &response.BookDownloadStatsResponse{
		BookID:             e.BookID,
		TotalDownloadCount: e.TotalDownloadCount,
		LastDownloadedAt:   e.LastDownloadedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

type BookmarkEntity struct {
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (e *BookmarkEntity) FromSqlc(res sqlc.Bookmark) *BookmarkEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.CreatedAt = res.CreatedAt
	return e
}

func (e *BookmarkEntity) ToResponse() *response.BookmarkResponse {
	if e == nil {
		return nil
	}
	return &response.BookmarkResponse{
		UserID:    e.UserID,
		BookID:    e.BookID,
		CreatedAt: e.CreatedAt,
	}
}

type BookRatingSummaryEntity struct {
	BookID        string  `json:"book_id"`
	RatingCount   int64   `json:"rating_count"`
	AverageRating float64 `json:"average_rating"`
}

func (e *BookRatingSummaryEntity) FromSqlc(res sqlc.GetBookRatingSummaryRow) *BookRatingSummaryEntity {
	e.BookID = res.BookID
	e.RatingCount = res.RatingCount
	e.AverageRating = res.AverageRating
	return e
}

func (e *BookRatingSummaryEntity) ToResponse() *response.BookRatingSummaryResponse {
	if e == nil {
		return nil
	}
	return &response.BookRatingSummaryResponse{
		BookID:        e.BookID,
		RatingCount:   e.RatingCount,
		AverageRating: e.AverageRating,
	}
}

type BookSocialStatsEntity struct {
	BookID        string     `json:"book_id"`
	BookmarkCount int64      `json:"bookmark_count"`
	RatingCount   int64      `json:"rating_count"`
	AverageRating float64    `json:"average_rating"`
	ShareCount    int64      `json:"share_count"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

func (e *BookSocialStatsEntity) FromSqlc(res sqlc.BookSocialStat) *BookSocialStatsEntity {
	e.BookID = res.BookID
	e.BookmarkCount = res.BookmarkCount
	e.RatingCount = res.RatingCount
	e.AverageRating = res.AverageRating
	e.ShareCount = res.ShareCount
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

func (e *BookSocialStatsEntity) ToResponse() *response.BookSocialStatsResponse {
	if e == nil {
		return nil
	}
	return &response.BookSocialStatsResponse{
		BookID:        e.BookID,
		BookmarkCount: e.BookmarkCount,
		RatingCount:   e.RatingCount,
		AverageRating: e.AverageRating,
		ShareCount:    e.ShareCount,
		UpdatedAt:     e.UpdatedAt,
	}
}

type BookEngagementStatsEntity struct {
	BookID        string                   `json:"book_id"`
	SocialStats   *BookSocialStatsEntity   `json:"social_stats"`
	DownloadStats *BookDownloadStatsEntity `json:"download_stats"`
	ReadStats     *BookReadStatsEntity     `json:"read_stats"`
}

func (e *BookEngagementStatsEntity) ToResponse() *response.BookEngagementStatsResponse {
	if e == nil {
		return nil
	}
	return &response.BookEngagementStatsResponse{
		BookID:        e.BookID,
		SocialStats:   e.SocialStats.ToResponse(),
		DownloadStats: e.DownloadStats.ToResponse(),
		ReadStats:     e.ReadStats.ToResponse(),
	}
}

type BookUserStateEntity struct {
	BookID        string                   `json:"book_id"`
	Bookmarked    bool                     `json:"bookmarked"`
	MyReview      *BookReviewEntity        `json:"my_review,omitempty"`
	RatingSummary *BookRatingSummaryEntity `json:"rating_summary"`
	SocialStats   *BookSocialStatsEntity   `json:"social_stats"`
	DownloadStats *BookDownloadStatsEntity `json:"download_stats"`
	ReadStats     *BookReadStatsEntity     `json:"read_stats"`
	Collections   []string                 `json:"collections"`
}

func (e *BookUserStateEntity) ToResponse() *response.BookUserStateResponse {
	if e == nil {
		return nil
	}
	cols := e.Collections
	if cols == nil {
		cols = []string{}
	}
	return &response.BookUserStateResponse{
		BookID:        e.BookID,
		Bookmarked:    e.Bookmarked,
		MyReview:      e.MyReview.ToResponse(),
		RatingSummary: e.RatingSummary.ToResponse(),
		SocialStats:   e.SocialStats.ToResponse(),
		DownloadStats: e.DownloadStats.ToResponse(),
		ReadStats:     e.ReadStats.ToResponse(),
		Collections:   cols,
	}
}

type ShareInput struct {
	BookID     string
	ActorKey   string
	OccurredAt time.Time
}
