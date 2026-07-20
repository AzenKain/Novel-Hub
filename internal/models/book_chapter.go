package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type ChapterEntity struct {
	ID           string    `json:"id"`
	BookID       string    `json:"bookId"`
	Title        string    `json:"title"`
	ContentPath  *string   `json:"contentPath"`
	ChapterIndex int64     `json:"chapterIndex"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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
