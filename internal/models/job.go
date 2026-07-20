package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type JobEntity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      *string   `json:"status"`
	Progress    *int64    `json:"progress"`
	Total       *int64    `json:"total"`
	ErrorMsg    *string   `json:"errorMsg"`
	PayloadJSON *string   `json:"payloadJson"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (e *JobEntity) FromSqlc(res sqlc.Job) *JobEntity {
	if e == nil {
		e = &JobEntity{}
	}
	e.ID = res.ID
	e.Type = res.Type
	e.Status = convert.NullStringToStrPtr(res.Status)
	if res.Progress.Valid {
		e.Progress = &res.Progress.Int64
	}
	if res.Total.Valid {
		e.Total = &res.Total.Int64
	}
	e.ErrorMsg = convert.NullStringToStrPtr(res.ErrorMsg)
	e.PayloadJSON = convert.NullStringToStrPtr(res.PayloadJson)
	e.CreatedAt = res.CreatedAt.Time
	e.UpdatedAt = res.UpdatedAt.Time
	return e
}
