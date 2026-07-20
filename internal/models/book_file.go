package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type BookFileEntity struct {
	ID        string    `json:"id"`
	BookID    string    `json:"bookId"`
	Path      string    `json:"path"`
	Format    string    `json:"format"`
	SizeBytes int64     `json:"sizeBytes"`
	ModTime   time.Time `json:"modTime"`
	Hash      *string   `json:"hash"`
	State     *string   `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *BookFileEntity) FromSqlc(res sqlc.BookFile) *BookFileEntity {
	e.ID = res.ID
	e.BookID = res.BookID
	e.Path = res.Path
	e.Format = res.Format
	e.SizeBytes = res.SizeBytes
	e.ModTime = res.ModTime
	e.Hash = convert.NullStringToStrPtr(res.Hash)
	e.State = convert.NullStringToStrPtr(res.State)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type BookFileEntities []*BookFileEntity

func (e *BookFileEntities) FromSqlc(rows []sqlc.BookFile) []*BookFileEntity {
	slice := make([]*BookFileEntity, len(rows))
	flat := make([]BookFileEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type BookFileUploadResult struct {
	Uploaded int               `json:"uploaded"`
	Total    int               `json:"total"`
	Files    []*BookFileEntity `json:"files"`
}

type FileRefEntity struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	BookID string `json:"bookId"`
}

func (e *FileRefEntity) FromSqlc(res sqlc.ListAllFilesRow) *FileRefEntity {
	e.ID = res.ID
	e.Path = res.Path
	e.BookID = res.BookID
	return e
}

type FileRefEntities []*FileRefEntity

func (e *FileRefEntities) FromSqlc(rows []sqlc.ListAllFilesRow) []*FileRefEntity {
	slice := make([]*FileRefEntity, len(rows))
	flat := make([]FileRefEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

type DuplicateFileEntity struct {
	Hash           *string `json:"hash"`
	DuplicateCount int64   `json:"duplicateCount"`
	FileIDs        string  `json:"fileIds"`
}

func (e *DuplicateFileEntity) FromSqlc(res sqlc.GetDuplicateFilesRow) *DuplicateFileEntity {
	e.Hash = convert.NullStringToStrPtr(res.Hash)
	e.DuplicateCount = res.DuplicateCount
	e.FileIDs = res.FileIds
	return e
}

type DuplicateFileEntities []*DuplicateFileEntity

func (e *DuplicateFileEntities) FromSqlc(rows []sqlc.GetDuplicateFilesRow) []*DuplicateFileEntity {
	slice := make([]*DuplicateFileEntity, len(rows))
	flat := make([]DuplicateFileEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}
