package models

import (
	"time"

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

type BookmarkEntity struct {
	UserID    int64     `json:"userId"`
	BookID    string    `json:"bookId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *BookmarkEntity) FromSqlc(res sqlc.Bookmark) *BookmarkEntity {
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.CreatedAt = res.CreatedAt.Time
	return e
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

type BookEngagementStatsEntity struct {
	BookID        string                   `json:"bookId"`
	SocialStats   *BookSocialStatsEntity   `json:"socialStats"`
	DownloadStats *BookDownloadStatsEntity `json:"downloadStats"`
	ReadStats     *BookReadStatsEntity     `json:"readStats"`
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

type ShareInput struct {
	BookID     string
	ActorKey   string
	OccurredAt time.Time
}
