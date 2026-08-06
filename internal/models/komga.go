package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type KomgaSeriesEntity struct {
	ID           string
	Name         string
	LibraryID    string
	BookCount    int64
	LastModified string
}

func (e *KomgaSeriesEntity) FromSqlc(row sqlc.GetKomgaSeriesByIDsRow) *KomgaSeriesEntity {
	e.ID = row.ID
	e.Name = row.Name
	e.LibraryID = row.LibraryID
	e.BookCount = row.BookCount
	e.LastModified = row.LastModified
	return e
}

type KomgaSeriesBookEntity struct {
	ID          string
	LibraryID   string
	Title       string
	Description *string
	CoverURL    *string
	UpdatedAt   *time.Time
	CreatedAt   *time.Time
	SeriesIndex *string
	NumberSort  float64
}

func (e *KomgaSeriesBookEntity) FromSqlc(row sqlc.ListKomgaSeriesBooksRow) *KomgaSeriesBookEntity {
	e.ID = row.ID
	e.LibraryID = row.LibraryID
	e.Title = row.Title
	e.Description = convert.NullStringToStrPtr(row.Description)
	e.CoverURL = convert.NullStringToStrPtr(row.CoverUrl)
	if row.UpdatedAt.Valid {
		updated := row.UpdatedAt.Time
		e.UpdatedAt = &updated
	}
	e.CreatedAt = &row.CreatedAt
	e.SeriesIndex = convert.NullStringToStrPtr(row.SeriesIndex)
	e.NumberSort = row.NumberSort
	return e
}

type KomgaBookSeriesRefEntity struct {
	SeriesID    string
	SeriesName  string
	SeriesIndex *string
	NumberSort  float64
}

func (e *KomgaBookSeriesRefEntity) FromSqlc(row sqlc.GetKomgaBookSeriesRow) *KomgaBookSeriesRefEntity {
	e.SeriesID = row.ID
	e.SeriesName = row.Name
	e.SeriesIndex = convert.NullStringToStrPtr(row.SeriesIndex)
	e.NumberSort = row.NumberSort
	return e
}

type KomgaSeriesProgressEntity struct {
	BooksCount           int64
	BooksReadCount       int64
	BooksInProgressCount int64
	LastReadNumberSort   float64
	MaxNumberSort        float64
}

func (e *KomgaSeriesProgressEntity) FromSqlcList(row sqlc.ListKomgaSeriesProgressRow) *KomgaSeriesProgressEntity {
	e.BooksCount = row.BooksCount
	e.BooksReadCount = row.BooksReadCount
	e.BooksInProgressCount = row.BooksInProgressCount
	e.LastReadNumberSort = row.LastReadNumberSort
	e.MaxNumberSort = row.MaxNumberSort
	return e
}

func (e *KomgaSeriesProgressEntity) FromSqlc(row sqlc.GetKomgaSeriesProgressRow) *KomgaSeriesProgressEntity {
	e.BooksCount = row.BooksCount
	e.BooksReadCount = row.BooksReadCount
	e.BooksInProgressCount = row.BooksInProgressCount
	e.LastReadNumberSort = row.LastReadNumberSort
	e.MaxNumberSort = row.MaxNumberSort
	return e
}

func (e *KomgaSeriesProgressEntity) BooksUnreadCount() int64 {
	unread := e.BooksCount - e.BooksReadCount - e.BooksInProgressCount
	if unread < 0 {
		return 0
	}
	return unread
}
