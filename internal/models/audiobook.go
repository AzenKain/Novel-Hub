package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type AudiobookChapterEntity struct {
	ID           string     `json:"id"`
	BookID       string     `json:"book_id"`
	FileID       *string    `json:"file_id,omitempty"`
	ChapterIndex int64      `json:"chapter_index"`
	Title        string     `json:"title"`
	StartSec     float64    `json:"start_sec"`
	EndSec       *float64   `json:"end_sec,omitempty"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (e *AudiobookChapterEntity) FromSqlc(r sqlc.AudiobookChapter) *AudiobookChapterEntity {
	e.ID = r.ID
	e.BookID = r.BookID
	e.FileID = convert.NullStringToStrPtr(r.FileID)
	e.ChapterIndex = r.ChapterIndex
	e.Title = r.Title
	e.StartSec = r.StartSec
	if r.EndSec.Valid {
		v := r.EndSec.Float64
		e.EndSec = &v
	}
	e.CreatedAt = convert.NullTimeToTimePtr(r.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(r.UpdatedAt)
	return e
}

type AudiobookChapterEntities []*AudiobookChapterEntity

func (e *AudiobookChapterEntities) FromSqlc(rows []sqlc.AudiobookChapter) []*AudiobookChapterEntity {
	out := make([]*AudiobookChapterEntity, len(rows))
	for i, row := range rows {
		out[i] = (&AudiobookChapterEntity{}).FromSqlc(row)
	}
	return out
}

func (e *AudiobookChapterEntity) ToResponse() *response.AudiobookChapterResponse {
	resp := &response.AudiobookChapterResponse{
		ID:           e.ID,
		BookID:       e.BookID,
		FileID:       e.FileID,
		ChapterIndex: e.ChapterIndex,
		Title:        e.Title,
		StartSec:     e.StartSec,
		EndSec:       e.EndSec,
	}
	if e.CreatedAt != nil {
		resp.CreatedAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		resp.UpdatedAt = *e.UpdatedAt
	}
	return resp
}
