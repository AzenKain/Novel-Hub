package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

type ReadListEntity struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (e *ReadListEntity) FromSqlc(res sqlc.ReadList) *ReadListEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.Description = res.Description.String
	e.CreatedAt = res.CreatedAt
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type ReadListEntities []*ReadListEntity

func (e *ReadListEntities) FromSqlc(rows []sqlc.ReadList) []*ReadListEntity {
	slice := make([]*ReadListEntity, len(rows))
	flat := make([]ReadListEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *ReadListEntity) ToResponse(bookCount int64) *response.ReadListResponse {
	if e == nil {
		return nil
	}
	return &response.ReadListResponse{
		ID:          e.ID,
		UserID:      e.UserID,
		Name:        e.Name,
		Description: e.Description,
		BookCount:   bookCount,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func ReadListEntitiesToResponse(entities []*ReadListEntity, counts map[string]int64) []*response.ReadListResponse {
	out := make([]*response.ReadListResponse, 0, len(entities))
	for _, rl := range entities {
		if rl == nil {
			continue
		}
		out = append(out, rl.ToResponse(counts[rl.ID]))
	}
	return out
}
