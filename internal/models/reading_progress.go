package models

import (
	"time"

	"novelhub/internal/dtos/response"
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

func (e *ReadingHistoryEntity) ToResponse() *response.ReadingHistoryResponse {
	if e == nil {
		return nil
	}
	return &response.ReadingHistoryResponse{
		UserID:          e.UserID,
		BookID:          e.BookID,
		FileID:          e.FileID,
		ChapterID:       e.ChapterID,
		ProgressPercent: e.ProgressPercent,
		UpdatedAt:       e.UpdatedAt,
		BookTitle:       e.BookTitle,
		BookCoverURL:    e.BookCoverURL,
		ChapterTitle:    e.ChapterTitle,
		ChapterIndex:    e.ChapterIndex,
	}
}

func ReadingHistoryEntitiesToResponse(entities []*ReadingHistoryEntity) []*response.ReadingHistoryResponse {
	out := make([]*response.ReadingHistoryResponse, 0, len(entities))
	for _, h := range entities {
		if h == nil {
			continue
		}
		out = append(out, h.ToResponse())
	}
	return out
}

type ReadingProgressEntity struct {
	UserID             int64      `json:"userId"`
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
	e.LocationCfi = convert.NullStringToStrPtr(res.LocationCfi)
	e.LocationType = convert.NullStringToStrPtr(res.LocationType)
	e.OpenedCount = res.OpenedCount
	e.QualifiedReadCount = res.QualifiedReadCount
	e.LastOpenedAt = convert.NullTimeToTimePtr(res.LastOpenedAt)
	e.LastCountedAt = convert.NullTimeToTimePtr(res.LastCountedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

func (e *ReadingProgressEntity) ToResponse() *response.ReadingProgressResponse {
	if e == nil {
		return nil
	}
	return &response.ReadingProgressResponse{
		UserID:             e.UserID,
		BookID:             e.BookID,
		FileID:             e.FileID,
		ChapterID:          e.ChapterID,
		ChapterTitle:       e.ChapterTitle,
		ChapterIndex:       e.ChapterIndex,
		ProgressPercent:    e.ProgressPercent,
		LocationCfi:        e.LocationCfi,
		LocationType:       e.LocationType,
		OpenedCount:        e.OpenedCount,
		QualifiedReadCount: e.QualifiedReadCount,
		LastOpenedAt:       e.LastOpenedAt,
		LastCountedAt:      e.LastCountedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}

type ReadingActivityInput struct {
	UserID          int64
	BookID          string
	FileID          *string
	ChapterID       string
	ChapterTitle    string
	ChapterIndex    int64
	ProgressPercent *float64
	LocationCfi     *string
	LocationType    *string
	EventType       string
}

type ReadingActivityEntity struct {
	Progress        *ReadingProgressEntity `json:"progress"`
	Stats           *BookReadStatsEntity   `json:"stats"`
	Counted         bool                   `json:"counted"`
	CooldownSeconds int64                  `json:"cooldownSeconds"`
}

func (e *ReadingActivityEntity) ToResponse() *response.ReadingActivityResponse {
	if e == nil {
		return nil
	}
	return &response.ReadingActivityResponse{
		Progress:        e.Progress.ToResponse(),
		Stats:           e.Stats.ToResponse(),
		Counted:         e.Counted,
		CooldownSeconds: e.CooldownSeconds,
	}
}
