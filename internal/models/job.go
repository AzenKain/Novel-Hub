package models

import (
	"time"

	"novelhub/internal/dtos/response"
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

type JobEntities []*JobEntity

func (e *JobEntities) FromSqlc(rows []sqlc.Job) []*JobEntity {
	slice := make([]*JobEntity, len(rows))
	flat := make([]JobEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *JobEntity) ToResponse() *response.JobResponse {
	if e == nil {
		return nil
	}
	return &response.JobResponse{
		ID:          e.ID,
		Type:        e.Type,
		Status:      e.Status,
		Progress:    e.Progress,
		Total:       e.Total,
		ErrorMsg:    e.ErrorMsg,
		PayloadJSON: e.PayloadJSON,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func JobEntitiesToResponse(entities []*JobEntity) []*response.JobResponse {
	out := make([]*response.JobResponse, 0, len(entities))
	for _, j := range entities {
		if j == nil {
			continue
		}
		out = append(out, j.ToResponse())
	}
	return out
}
