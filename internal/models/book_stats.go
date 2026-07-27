package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type LibraryStatsEntity struct {
	TotalBooks    int64 `json:"totalBooks"`
	NeedReview    int64 `json:"needReview"`
	SeriesTracked int64 `json:"seriesTracked"`
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
	BookID             string     `json:"bookId"`
	TotalOpenCount     int64      `json:"totalOpenCount"`
	QualifiedReadCount int64      `json:"qualifiedReadCount"`
	LastOpenedAt       *time.Time `json:"lastOpenedAt,omitempty"`
	LastCountedAt      *time.Time `json:"lastCountedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
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
	BookID             string     `json:"bookId"`
	TotalDownloadCount int64      `json:"totalDownloadCount"`
	LastDownloadedAt   *time.Time `json:"lastDownloadedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
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
	UserID    string    `json:"userId"`
	BookID    string    `json:"bookId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *BookmarkEntity) FromSqlc(res sqlc.Bookmark) *BookmarkEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.CreatedAt = res.CreatedAt.Time
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
	BookID        string  `json:"bookId"`
	RatingCount   int64   `json:"ratingCount"`
	AverageRating float64 `json:"averageRating"`
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
	BookID        string     `json:"bookId"`
	BookmarkCount int64      `json:"bookmarkCount"`
	RatingCount   int64      `json:"ratingCount"`
	AverageRating float64    `json:"averageRating"`
	ShareCount    int64      `json:"shareCount"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
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
	BookID        string                   `json:"bookId"`
	SocialStats   *BookSocialStatsEntity   `json:"socialStats"`
	DownloadStats *BookDownloadStatsEntity `json:"downloadStats"`
	ReadStats     *BookReadStatsEntity     `json:"readStats"`
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
	BookID        string                   `json:"bookId"`
	Bookmarked    bool                     `json:"bookmarked"`
	MyReview      *BookReviewEntity        `json:"myReview,omitempty"`
	RatingSummary *BookRatingSummaryEntity `json:"ratingSummary"`
	SocialStats   *BookSocialStatsEntity   `json:"socialStats"`
	DownloadStats *BookDownloadStatsEntity `json:"downloadStats"`
	ReadStats     *BookReadStatsEntity     `json:"readStats"`
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
