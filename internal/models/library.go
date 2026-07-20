package models

import (
	"time"

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
