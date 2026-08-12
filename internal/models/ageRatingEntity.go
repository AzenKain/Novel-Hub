package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type ContentWarningEntity struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (e *ContentWarningEntity) FromSqlc(row sqlc.ContentWarning) *ContentWarningEntity {
	e.ID = row.ID
	e.Name = row.Name
	e.Description = row.Description
	e.CreatedAt = row.CreatedAt.Time
	return e
}

func (e *ContentWarningEntity) FromBookRow(row sqlc.GetBookContentWarningsRow) *ContentWarningEntity {
	e.ID = row.ID
	e.Name = row.Name
	e.Description = row.Description
	return e
}

func (e *ContentWarningEntity) ToResponse() *response.ContentWarningResponse {
	if e == nil {
		return nil
	}
	return &response.ContentWarningResponse{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		CreatedAt:   e.CreatedAt,
	}
}

func ContentWarningsToResponse(list []*ContentWarningEntity) []*response.ContentWarningResponse {
	out := make([]*response.ContentWarningResponse, 0, len(list))
	for _, item := range list {
		if item != nil {
			out = append(out, item.ToResponse())
		}
	}
	return out
}

type KidsModeInfoEntity struct {
	ID                  string  `json:"id"`
	IsKidsMode          bool    `json:"is_kids_mode"`
	KidsModePinHash     *string `json:"kids_mode_pin_hash,omitempty"`
	MaxAllowedAgeRating string  `json:"max_allowed_age_rating"`
}

func (e *KidsModeInfoEntity) FromSqlc(row sqlc.GetUserKidsModeInfoRow) *KidsModeInfoEntity {
	e.ID = row.ID
	e.IsKidsMode = row.IsKidsMode == 1
	e.KidsModePinHash = convert.NullStringToStrPtr(row.KidsModePinHash)
	e.MaxAllowedAgeRating = row.MaxAllowedAgeRating
	return e
}

func (e *KidsModeInfoEntity) ToResponse() *response.KidsModeInfoResponse {
	if e == nil {
		return nil
	}
	return &response.KidsModeInfoResponse{
		ID:                  e.ID,
		IsKidsMode:          e.IsKidsMode,
		HasPin:              e.KidsModePinHash != nil && *e.KidsModePinHash != "",
		MaxAllowedAgeRating: e.MaxAllowedAgeRating,
	}
}
