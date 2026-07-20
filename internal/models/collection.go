package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
)

type CollectionEntity struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *CollectionEntity) FromSqlc(res sqlc.Collection) *CollectionEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type CollectionEntities []*CollectionEntity

func (e *CollectionEntities) FromSqlc(rows []sqlc.Collection) []*CollectionEntity {
	slice := make([]*CollectionEntity, len(rows))
	flat := make([]CollectionEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}
