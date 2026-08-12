package models

import (
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/jsonx"
)

type SmartFilterEntity struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Name            string    `json:"name"`
	RulesJson       string    `json:"rules_json"`
	IsPinnedSidebar bool      `json:"is_pinned_sidebar"`
	IsPinnedHome    bool      `json:"is_pinned_home"`
	HomePosition    int64     `json:"home_position"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (e *SmartFilterEntity) FromSqlc(res sqlc.SmartFilter) *SmartFilterEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.RulesJson = res.RulesJson
	e.IsPinnedSidebar = res.IsPinnedSidebar
	e.IsPinnedHome = res.IsPinnedHome
	e.HomePosition = res.HomePosition
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}

type SmartFilterEntities []*SmartFilterEntity

func (e *SmartFilterEntities) FromSqlc(rows []sqlc.SmartFilter) []*SmartFilterEntity {
	slice := make([]*SmartFilterEntity, len(rows))
	flat := make([]SmartFilterEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *SmartFilterEntity) ToResponse() *response.SmartFilterResponse {
	if e == nil {
		return nil
	}
	var rules []request.SmartFilterRuleItemDto
	if err := jsonx.Unmarshal([]byte(e.RulesJson), &rules); err != nil {
		rules = []request.SmartFilterRuleItemDto{}
	}
	return &response.SmartFilterResponse{
		ID:              e.ID,
		UserID:          e.UserID,
		Name:            e.Name,
		Rules:           rules,
		IsPinnedSidebar: e.IsPinnedSidebar,
		IsPinnedHome:    e.IsPinnedHome,
		HomePosition:    e.HomePosition,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func SmartFilterEntitiesToResponse(entities []*SmartFilterEntity) []*response.SmartFilterResponse {
	res := make([]*response.SmartFilterResponse, len(entities))
	for i, entity := range entities {
		res[i] = entity.ToResponse()
	}
	return res
}
