package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookEntity struct {
	ID           string            `json:"id"`
	LibraryID    string            `json:"libraryId"`
	Title        string            `json:"title"`
	AuthorID     *string           `json:"authorId"`
	AuthorName   *string           `json:"authorName,omitempty"`
	Description  *string           `json:"description"`
	CoverURL     *string           `json:"coverUrl"`
	Status       string            `json:"status"`
	MetadataJSON *string           `json:"metadataJson"`
	Files        []*BookFileEntity `json:"files,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
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

type FTSResultEntity struct {
	BookID    string `json:"bookId"`
	ChapterID string `json:"chapterId"`
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
