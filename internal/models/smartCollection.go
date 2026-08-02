package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
)

// RuleJson stays an opaque string here; the service owns marshalling it to and
// from the validated rule DTO. Nothing reads it as a raw string past that.
type SmartCollectionEntity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	RuleJson  string    `json:"ruleJson"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (e *SmartCollectionEntity) FromSqlc(res sqlc.SmartCollection) *SmartCollectionEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.RuleJson = res.RuleJson
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type SmartCollectionEntities []*SmartCollectionEntity

func (e *SmartCollectionEntities) FromSqlc(rows []sqlc.SmartCollection) []*SmartCollectionEntity {
	slice := make([]*SmartCollectionEntity, len(rows))
	flat := make([]SmartCollectionEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *SmartCollectionEntity) ToResponse() *response.SmartCollectionResponse {
	if e == nil {
		return nil
	}
	return &response.SmartCollectionResponse{
		ID:        e.ID,
		UserID:    e.UserID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
