package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

type CollectionEntity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (e *CollectionEntity) FromSqlc(res sqlc.Collection) *CollectionEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.CreatedAt = res.CreatedAt
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

func (e *CollectionEntity) ToResponse() *response.CollectionResponse {
	if e == nil {
		return nil
	}
	return &response.CollectionResponse{
		ID:        e.ID,
		UserID:    e.UserID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func CollectionEntitiesToResponse(entities []*CollectionEntity) []*response.CollectionResponse {
	out := make([]*response.CollectionResponse, 0, len(entities))
	for _, c := range entities {
		if c == nil {
			continue
		}
		out = append(out, c.ToResponse())
	}
	return out
}
