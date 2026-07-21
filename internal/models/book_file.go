package models

import (
	"time"

	"novelhub/internal/dtos/response"
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

func (e *BookFileEntity) ToResponse() *response.BookFileResponse {
	if e == nil {
		return nil
	}
	return &response.BookFileResponse{
		ID:        e.ID,
		BookID:    e.BookID,
		Path:      e.Path,
		Format:    e.Format,
		SizeBytes: e.SizeBytes,
		ModTime:   e.ModTime,
		Hash:      e.Hash,
		State:     e.State,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func BookFileEntitiesToResponse(entities []*BookFileEntity) []*response.BookFileResponse {
	out := make([]*response.BookFileResponse, 0, len(entities))
	for _, f := range entities {
		if f == nil {
			continue
		}
		out = append(out, f.ToResponse())
	}
	return out
}

type BookFileUploadResult struct {
	Uploaded int               `json:"uploaded"`
	Total    int               `json:"total"`
	Files    []*BookFileEntity `json:"files"`
}

func (r *BookFileUploadResult) ToResponse() *response.BookFileUploadResultResponse {
	if r == nil {
		return nil
	}
	var files []*response.BookFileResponse
	if r.Files != nil {
		files = BookFileEntitiesToResponse(r.Files)
	}
	return &response.BookFileUploadResultResponse{
		Uploaded: r.Uploaded,
		Total:    r.Total,
		Files:    files,
	}
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

func (e *FileRefEntity) ToResponse() *response.FileRefResponse {
	if e == nil {
		return nil
	}
	return &response.FileRefResponse{
		ID:     e.ID,
		Path:   e.Path,
		BookID: e.BookID,
	}
}

func FileRefEntitiesToResponse(entities []*FileRefEntity) []*response.FileRefResponse {
	out := make([]*response.FileRefResponse, 0, len(entities))
	for _, f := range entities {
		if f == nil {
			continue
		}
		out = append(out, f.ToResponse())
	}
	return out
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

func (e *DuplicateFileEntity) ToResponse() *response.DuplicateFileResponse {
	if e == nil {
		return nil
	}
	return &response.DuplicateFileResponse{
		Hash:           e.Hash,
		DuplicateCount: e.DuplicateCount,
		FileIDs:        e.FileIDs,
	}
}

func DuplicateFileEntitiesToResponse(entities []*DuplicateFileEntity) []*response.DuplicateFileResponse {
	out := make([]*response.DuplicateFileResponse, 0, len(entities))
	for _, d := range entities {
		if d == nil {
			continue
		}
		out = append(out, d.ToResponse())
	}
	return out
}
