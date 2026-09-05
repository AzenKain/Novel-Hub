package models

import (
	"database/sql"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type HighlightEntity struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	BookID      string     `json:"book_id"`
	ChapterID   string     `json:"chapter_id"`
	TextContent string     `json:"text_content"`
	StartIndex  int64      `json:"start_index"`
	EndIndex    int64      `json:"end_index"`
	Color       string     `json:"color"`
	Note        *string    `json:"note,omitempty"`
	CfiRange    *string    `json:"cfi_range,omitempty"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func (e *HighlightEntity) FromSqlc(res sqlc.Highlight) *HighlightEntity {
	return e.fillFromHighlightRow(
		res.ID, res.UserID, res.BookID, res.ChapterID, res.TextContent, res.StartIndex,
		res.EndIndex, res.Color, res.Note, res.CfiRange, res.CreatedAt, res.UpdatedAt,
	)
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
		CfiRange:    e.CfiRange,
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

// HighlightBookEntity decorates a highlight with its book title + author, from the export JOINs.
type HighlightBookEntity struct {
	HighlightEntity
	BookTitle  string
	AuthorName string
}

func (e *HighlightBookEntity) FromSqlc(res sqlc.GetHighlightsByBookRow) *HighlightBookEntity {
	(&e.HighlightEntity).fillFromHighlightRow(
		res.ID, res.UserID, res.BookID, res.ChapterID, res.TextContent, res.StartIndex,
		res.EndIndex, res.Color, res.Note, res.CfiRange, res.CreatedAt, res.UpdatedAt,
	)
	e.BookTitle = res.BookTitle
	e.AuthorName = res.AuthorName
	return e
}

func (e *HighlightBookEntity) FromSqlcByIDs(res sqlc.GetHighlightBooksByIDsRow) *HighlightBookEntity {
	(&e.HighlightEntity).fillFromHighlightRow(
		res.ID, res.UserID, res.BookID, res.ChapterID, res.TextContent, res.StartIndex,
		res.EndIndex, res.Color, res.Note, res.CfiRange, res.CreatedAt, res.UpdatedAt,
	)
	e.BookTitle = res.BookTitle
	e.AuthorName = res.AuthorName
	return e
}

func (e *HighlightEntity) fillFromHighlightRow(id, userID, bookID, chapterID, textContent string, startIndex, endIndex int64, color string, note, cfiRange sql.NullString, createdAt, updatedAt sql.NullTime) *HighlightEntity {
	e.ID = id
	e.UserID = userID
	e.BookID = bookID
	e.ChapterID = chapterID
	e.TextContent = textContent
	e.StartIndex = startIndex
	e.EndIndex = endIndex
	e.Color = color
	e.Note = convert.NullStringToStrPtr(note)
	e.CfiRange = convert.NullStringToStrPtr(cfiRange)
	e.CreatedAt = convert.NullTimeToTimePtr(createdAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(updatedAt)
	return e
}
