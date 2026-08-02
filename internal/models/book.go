package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookEntity struct {
	ID           string            `json:"id"`
	LibraryID    string            `json:"library_id"`
	Title        string            `json:"title"`
	AuthorID     *string           `json:"author_id"`
	AuthorName   *string           `json:"author_name,omitempty"`
	Description  *string           `json:"description"`
	CoverURL     *string           `json:"cover_url"`
	Status       string            `json:"status"`
	MetadataJSON *string           `json:"metadata_json"`
	Files        []*BookFileEntity `json:"files,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func (e *BookEntity) FromSqlc(res sqlc.Book) *BookEntity {
	status := ""
	if res.Status.Valid {
		status = res.Status.String
	}
	e.ID = res.ID
	e.LibraryID = res.LibraryID
	e.Title = res.Title
	e.AuthorID = convert.NullStringToStrPtr(res.AuthorID)
	e.Description = convert.NullStringToStrPtr(res.Description)
	e.CoverURL = convert.NullStringToStrPtr(res.CoverUrl)
	e.Status = status
	e.MetadataJSON = convert.NullStringToStrPtr(res.MetadataJson)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type BookEntities []*BookEntity

func (e *BookEntities) FromSqlc(rows []sqlc.Book) []*BookEntity {
	slice := make([]*BookEntity, len(rows))
	flat := make([]BookEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

func (e *BookEntity) ToResponse() *response.BookResponse {
	if e == nil {
		return nil
	}
	var files []*response.BookFileResponse
	if e.Files != nil {
		files = BookFileEntitiesToResponse(e.Files)
	}
	return &response.BookResponse{
		ID:           e.ID,
		LibraryID:    e.LibraryID,
		Title:        e.Title,
		AuthorID:     e.AuthorID,
		AuthorName:   e.AuthorName,
		Description:  e.Description,
		CoverURL:     e.CoverURL,
		Status:       e.Status,
		MetadataJSON: e.MetadataJSON,
		Files:        files,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func BookEntitiesToResponse(entities []*BookEntity) []*response.BookResponse {
	out := make([]*response.BookResponse, 0, len(entities))
	for _, b := range entities {
		if b == nil {
			continue
		}
		out = append(out, b.ToResponse())
	}
	return out
}

type FTSResultEntity struct {
	BookID    string `json:"book_id"`
	ChapterID string `json:"chapter_id"`
	Title     string `json:"title"`
}

func (e *FTSResultEntity) FromSqlc(res sqlc.SearchFTSRow) *FTSResultEntity {
	e.BookID = res.BookID
	e.ChapterID = res.ChapterID
	e.Title = res.Title
	return e
}

type FTSResultEntities []*FTSResultEntity

func (e *FTSResultEntities) FromSqlc(rows []sqlc.SearchFTSRow) []*FTSResultEntity {
	slice := make([]*FTSResultEntity, len(rows))
	flat := make([]FTSResultEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

func (e *FTSResultEntity) ToResponse() *response.FTSResultResponse {
	if e == nil {
		return nil
	}
	return &response.FTSResultResponse{
		BookID:    e.BookID,
		ChapterID: e.ChapterID,
		Title:     e.Title,
	}
}

func FTSResultEntitiesToResponse(entities []*FTSResultEntity) []*response.FTSResultResponse {
	out := make([]*response.FTSResultResponse, 0, len(entities))
	for _, f := range entities {
		if f == nil {
			continue
		}
		out = append(out, f.ToResponse())
	}
	return out
}
