package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type HighlightEntity struct {
	ID          string     `json:"id"`
	UserID      int64      `json:"userId"`
	BookID      string     `json:"bookId"`
	ChapterID   string     `json:"chapterId"`
	TextContent string     `json:"textContent"`
	StartIndex  int64      `json:"startIndex"`
	EndIndex    int64      `json:"endIndex"`
	Color       string     `json:"color"`
	Note        *string    `json:"note,omitempty"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}

func (e *HighlightEntity) FromSqlc(res sqlc.Highlight) *HighlightEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.BookID = res.BookID
	e.ChapterID = res.ChapterID
	e.TextContent = res.TextContent
	e.StartIndex = res.StartIndex
	e.EndIndex = res.EndIndex
	e.Color = res.Color
	e.Note = convert.NullStringToStrPtr(res.Note)
	e.CreatedAt = convert.NullTimeToTimePtr(res.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

type HighlightEntities []*HighlightEntity

func (e *HighlightEntities) FromSqlc(rows []sqlc.Highlight) []*HighlightEntity {
	slice := make([]*HighlightEntity, len(rows))
	flat := make([]HighlightEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *HighlightEntity) ToResponse() *response.HighlightResponse {
	if e == nil {
		return nil
	}
	var createdAt, updatedAt time.Time
	if e.CreatedAt != nil {
		createdAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		updatedAt = *e.UpdatedAt
	}
	return &response.HighlightResponse{
		ID:          e.ID,
		UserID:      e.UserID,
		BookID:      e.BookID,
		ChapterID:   e.ChapterID,
		TextContent: e.TextContent,
		StartIndex:  e.StartIndex,
		EndIndex:    e.EndIndex,
		Color:       e.Color,
		Note:        e.Note,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func HighlightEntitiesToResponse(entities []*HighlightEntity) []*response.HighlightResponse {
	out := make([]*response.HighlightResponse, 0, len(entities))
	for _, h := range entities {
		if h == nil {
			continue
		}
		out = append(out, h.ToResponse())
	}
	return out
}
