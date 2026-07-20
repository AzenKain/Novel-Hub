package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

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

func (e *ReadingHistoryEntity) FromSqlc(res sqlc.GetRecentReadingHistoryRow) *ReadingHistoryEntity {
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

type ReadingHistoryEntities []*ReadingHistoryEntity

func (e *ReadingHistoryEntities) FromSqlc(rows []sqlc.GetRecentReadingHistoryRow) []*ReadingHistoryEntity {
	slice := make([]*ReadingHistoryEntity, len(rows))
	flat := make([]ReadingHistoryEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
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

func (e *ReadingProgressEntity) FromSqlc(res sqlc.ReadingProgress) *ReadingProgressEntity {
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
	e.LastOpenedAt = convert.NullTimeToTimePtr(res.LastOpenedAt)
	e.LastCountedAt = convert.NullTimeToTimePtr(res.LastCountedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
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
