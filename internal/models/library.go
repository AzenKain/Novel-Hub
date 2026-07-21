package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

type LibraryEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LibraryUploadResult struct {
	Uploaded int `json:"uploaded"`
	Total    int `json:"total"`
}

func (e *LibraryEntity) FromSqlc(res sqlc.Library) *LibraryEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type LibraryEntities []*LibraryEntity

func (e *LibraryEntities) FromSqlc(rows []sqlc.Library) []*LibraryEntity {
	slice := make([]*LibraryEntity, len(rows))
	flat := make([]LibraryEntity, len(rows))
	for i, res := range rows {
		slice[i] = flat[i].FromSqlc(res)
	}
	return slice
}

func (e *LibraryEntity) ToResponse() *response.LibraryResponse {
	if e == nil {
		return nil
	}
	return &response.LibraryResponse{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func LibraryEntitiesToResponse(entities []*LibraryEntity) []*response.LibraryResponse {
	out := make([]*response.LibraryResponse, 0, len(entities))
	for _, l := range entities {
		if l == nil {
			continue
		}
		out = append(out, l.ToResponse())
	}
	return out
}

func (r *LibraryUploadResult) ToResponse() *response.LibraryUploadResultResponse {
	if r == nil {
		return nil
	}
	return &response.LibraryUploadResultResponse{
		Uploaded: r.Uploaded,
		Total:    r.Total,
	}
}
