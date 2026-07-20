package models

import (
	"database/sql"
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type LibraryStatsEntity struct {
	TotalBooks    int64 `json:"totalBooks"`
	NeedReview    int64 `json:"needReview"`
	SeriesTracked int64 `json:"seriesTracked"`
}

type CollectionEntity struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ReadingHistoryEntity struct {
	UserID          int64     `json:"userId"`
	BookID          string    `json:"bookId"`
	FileID          *string   `json:"fileId,omitempty"`
	ChapterID       string    `json:"chapterId"`
	ProgressPercent *float64  `json:"progressPercent"`
	UpdatedAt       time.Time `json:"updatedAt"`
	BookTitle       string    `json:"bookTitle"`
	BookCoverURL    *string   `json:"bookCoverUrl"`
	ChapterTitle    string    `json:"chapterTitle"`
	ChapterIndex    int64     `json:"chapterIndex"`
}

type ReadingProgressEntity struct {
	UserID             int64      `json:"userId"`
	BookID             string     `json:"bookId"`
	FileID             *string    `json:"fileId,omitempty"`
	ChapterID          string     `json:"chapterId"`
	ChapterTitle       string     `json:"chapterTitle"`
	ChapterIndex       int64      `json:"chapterIndex"`
	ProgressPercent    *float64   `json:"progressPercent,omitempty"`
	OpenedCount        int64      `json:"openedCount"`
	QualifiedReadCount int64      `json:"qualifiedReadCount"`
	LastOpenedAt       *time.Time `json:"lastOpenedAt,omitempty"`
	LastCountedAt      *time.Time `json:"lastCountedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type BookReadStatsEntity struct {
	BookID             string     `json:"bookId"`
	TotalOpenCount     int64      `json:"totalOpenCount"`
	QualifiedReadCount int64      `json:"qualifiedReadCount"`
	LastOpenedAt       *time.Time `json:"lastOpenedAt,omitempty"`
	LastCountedAt      *time.Time `json:"lastCountedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type BookDownloadStatsEntity struct {
	BookID             string     `json:"bookId"`
	TotalDownloadCount int64      `json:"totalDownloadCount"`
	LastDownloadedAt   *time.Time `json:"lastDownloadedAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty"`
}

type BookmarkEntity struct {
	UserID    int64     `json:"userId"`
	BookID    string    `json:"bookId"`
	CreatedAt time.Time `json:"createdAt"`
}

type BookReviewEntity struct {
	UserID    int64      `json:"userId"`
	BookID    string     `json:"bookId"`
	Rating    int64      `json:"rating"`
	Review    *string    `json:"review,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UserName  string     `json:"userName,omitempty"`
	UserEmail string     `json:"userEmail,omitempty"`
	BookTitle string     `json:"bookTitle,omitempty"`
}

type BookRatingSummaryEntity struct {
	BookID        string  `json:"bookId"`
	RatingCount   int64   `json:"ratingCount"`
	AverageRating float64 `json:"averageRating"`
}

type BookSocialStatsEntity struct {
	BookID        string     `json:"bookId"`
	BookmarkCount int64      `json:"bookmarkCount"`
	RatingCount   int64      `json:"ratingCount"`
	AverageRating float64    `json:"averageRating"`
	ShareCount    int64      `json:"shareCount"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
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

type ReadingActivityInput struct {
	UserID          int64
	BookID          string
	FileID          *string
	ChapterID       string
	ChapterTitle    string
	ChapterIndex    int64
	ProgressPercent *float64
	EventType       string
}

type ReadingActivityEntity struct {
	Progress        *ReadingProgressEntity `json:"progress"`
	Stats           *BookReadStatsEntity   `json:"stats"`
	Counted         bool                   `json:"counted"`
	CooldownSeconds int64                  `json:"cooldownSeconds"`
}

type ShareInput struct {
	BookID     string
	ActorKey   string
	OccurredAt time.Time
}

func (e *LibraryStatsEntity) FromSqlc(res sqlc.GetLibraryStatsRow) *LibraryStatsEntity {
	if e == nil {
		e = &LibraryStatsEntity{}
	}
	e.TotalBooks = res.TotalBooks
	e.NeedReview = res.NeedReview
	e.SeriesTracked = res.SeriesTracked
	return e
}

func (e *CollectionEntity) FromSqlc(res sqlc.Collection) *CollectionEntity {
	if e == nil {
		e = &CollectionEntity{}
	}
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type CollectionEntities []*CollectionEntity

func (e *CollectionEntities) FromSqlc(rows []sqlc.Collection) []*CollectionEntity {
	collections := make([]*CollectionEntity, len(rows))
	for i, row := range rows {
		collections[i] = (&CollectionEntity{}).FromSqlc(row)
	}
	return collections
}

func (e *ReadingHistoryEntity) FromSqlc(res sqlc.GetRecentReadingHistoryRow) *ReadingHistoryEntity {
	if e == nil {
		e = &ReadingHistoryEntity{}
	}
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.FileID = convert.NullStringToStrPtr(res.FileID)
	e.ChapterID = res.ChapterID
	if res.ProgressPercent.Valid {
		e.ProgressPercent = &res.ProgressPercent.Float64
	}
	e.UpdatedAt = res.UpdatedAt.Time
	e.BookTitle = res.BookTitle
	e.BookCoverURL = convert.NullStringToStrPtr(res.BookCoverUrl)
	e.ChapterTitle = res.ChapterTitle
	e.ChapterIndex = res.ChapterIndex
	return e
}

func (e *ReadingProgressEntity) FromSqlc(res sqlc.ReadingProgress) *ReadingProgressEntity {
	if e == nil {
		e = &ReadingProgressEntity{}
	}
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.FileID = convert.NullStringToStrPtr(res.FileID)
	e.ChapterID = res.ChapterRef
	e.ChapterTitle = res.ChapterTitle
	e.ChapterIndex = res.ChapterIndex
	if res.ProgressPercent.Valid {
		e.ProgressPercent = &res.ProgressPercent.Float64
	}
	e.OpenedCount = res.OpenedCount
	e.QualifiedReadCount = res.QualifiedReadCount
	e.LastOpenedAt = nullTimePtr(res.LastOpenedAt)
	e.LastCountedAt = nullTimePtr(res.LastCountedAt)
	e.UpdatedAt = nullTimePtr(res.UpdatedAt)
	return e
}

func (e *BookReadStatsEntity) FromSqlc(res sqlc.BookReadStat) *BookReadStatsEntity {
	if e == nil {
		e = &BookReadStatsEntity{}
	}
	e.BookID = res.BookID
	e.TotalOpenCount = res.TotalOpenCount
	e.QualifiedReadCount = res.QualifiedReadCount
	e.LastOpenedAt = nullTimePtr(res.LastOpenedAt)
	e.LastCountedAt = nullTimePtr(res.LastCountedAt)
	e.UpdatedAt = nullTimePtr(res.UpdatedAt)
	return e
}

func (e *BookDownloadStatsEntity) FromSqlc(res sqlc.BookDownloadStat) *BookDownloadStatsEntity {
	if e == nil {
		e = &BookDownloadStatsEntity{}
	}
	e.BookID = res.BookID
	e.TotalDownloadCount = res.TotalDownloadCount
	e.LastDownloadedAt = nullTimePtr(res.LastDownloadedAt)
	e.UpdatedAt = nullTimePtr(res.UpdatedAt)
	return e
}

func (e *BookmarkEntity) FromSqlc(res sqlc.Bookmark) *BookmarkEntity {
	if e == nil {
		e = &BookmarkEntity{}
	}
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.CreatedAt = res.CreatedAt.Time
	return e
}

func (e *BookReviewEntity) FromSqlc(res sqlc.BookReview) *BookReviewEntity {
	if e == nil {
		e = &BookReviewEntity{}
	}
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.Rating = res.Rating
	e.Review = convert.NullStringToStrPtr(res.Review)
	e.CreatedAt = nullTimePtr(res.CreatedAt)
	e.UpdatedAt = nullTimePtr(res.UpdatedAt)
	return e
}

func (e *BookRatingSummaryEntity) FromSqlc(res sqlc.GetBookRatingSummaryRow) *BookRatingSummaryEntity {
	if e == nil {
		e = &BookRatingSummaryEntity{}
	}
	e.BookID = res.BookID
	e.RatingCount = res.RatingCount
	e.AverageRating = res.AverageRating
	return e
}

func (e *BookSocialStatsEntity) FromSqlc(res sqlc.BookSocialStat) *BookSocialStatsEntity {
	if e == nil {
		e = &BookSocialStatsEntity{}
	}
	e.BookID = res.BookID
	e.BookmarkCount = res.BookmarkCount
	e.RatingCount = res.RatingCount
	e.AverageRating = res.AverageRating
	e.ShareCount = res.ShareCount
	e.UpdatedAt = nullTimePtr(res.UpdatedAt)
	return e
}

type BookReviewEntities []*BookReviewEntity

func (e *BookReviewEntities) FromSqlc(rows []sqlc.BookReview) []*BookReviewEntity {
	reviews := make([]*BookReviewEntity, len(rows))
	for i, row := range rows {
		reviews[i] = (&BookReviewEntity{}).FromSqlc(row)
	}
	return reviews
}

type ReadingHistoryEntities []*ReadingHistoryEntity

func (e *ReadingHistoryEntities) FromSqlc(rows []sqlc.GetRecentReadingHistoryRow) []*ReadingHistoryEntity {
	history := make([]*ReadingHistoryEntity, len(rows))
	for i, row := range rows {
		history[i] = (&ReadingHistoryEntity{}).FromSqlc(row)
	}
	return history
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
