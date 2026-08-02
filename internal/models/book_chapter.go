package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type ChapterEntity struct {
	ID           string    `json:"id"`
	BookID       string    `json:"book_id"`
	Title        string    `json:"title"`
	ContentPath  *string   `json:"content_path"`
	ChapterIndex int64     `json:"chapter_index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (e *ChapterEntity) FromSqlc(res sqlc.Chapter) *ChapterEntity {
	e.ID = res.ID
	e.BookID = res.BookID
	e.Title = res.Title
	e.ContentPath = convert.NullStringToStrPtr(res.ContentPath)
	e.ChapterIndex = res.ChapterIndex
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type ChapterEntities []*ChapterEntity

func (e *ChapterEntities) FromSqlc(rows []sqlc.Chapter) []*ChapterEntity {
	slice := make([]*ChapterEntity, len(rows))
	flat := make([]ChapterEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

func (e *ChapterEntity) ToResponse() *response.ChapterResponse {
	if e == nil {
		return nil
	}
	return &response.ChapterResponse{
		ID:           e.ID,
		BookID:       e.BookID,
		Title:        e.Title,
		ContentPath:  e.ContentPath,
		ChapterIndex: e.ChapterIndex,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func ChapterEntitiesToResponse(entities []*ChapterEntity) []*response.ChapterResponse {
	out := make([]*response.ChapterResponse, 0, len(entities))
	for _, c := range entities {
		if c == nil {
			continue
		}
		out = append(out, c.ToResponse())
	}
	return out
}
