package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type JobScheduleEntity struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	TaskType        string     `json:"taskType"`
	PayloadJSON     *string    `json:"payloadJson"`
	IntervalMinutes int64      `json:"intervalMinutes"`
	Enabled         bool       `json:"enabled"`
	NextRunAt       time.Time  `json:"nextRunAt"`
	LastRunAt       *time.Time `json:"lastRunAt"`
	LastJobID       *string    `json:"lastJobId"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (e *JobScheduleEntity) FromSqlc(res sqlc.JobSchedule) *JobScheduleEntity {
	e.ID = res.ID
	e.Name = res.Name
	e.TaskType = res.TaskType
	e.PayloadJSON = convert.NullStringToStrPtr(res.PayloadJson)
	e.IntervalMinutes = res.IntervalMinutes
	e.Enabled = res.Enabled == 1
	e.NextRunAt = res.NextRunAt
	e.LastRunAt = convert.NullTimeToTimePtr(res.LastRunAt)
	e.LastJobID = convert.NullStringToStrPtr(res.LastJobID)
	e.CreatedAt = res.CreatedAt
	e.UpdatedAt = res.UpdatedAt
	return e
}

type JobScheduleEntities []*JobScheduleEntity

func (e *JobScheduleEntities) FromSqlc(rows []sqlc.JobSchedule) []*JobScheduleEntity {
	out := make([]*JobScheduleEntity, len(rows))
	for i, row := range rows {
		out[i] = (&JobScheduleEntity{}).FromSqlc(row)
	}
	return out
}

func (e *JobScheduleEntity) ToResponse() *response.JobScheduleResponse {
	if e == nil {
		return nil
	}
	return &response.JobScheduleResponse{
		ID:              e.ID,
		Name:            e.Name,
		TaskType:        e.TaskType,
		PayloadJSON:     e.PayloadJSON,
		IntervalMinutes: e.IntervalMinutes,
		Enabled:         e.Enabled,
		NextRunAt:       e.NextRunAt,
		LastRunAt:       e.LastRunAt,
		LastJobID:       e.LastJobID,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func JobSchedulesToResponse(entities []*JobScheduleEntity) []*response.JobScheduleResponse {
	out := make([]*response.JobScheduleResponse, 0, len(entities))
	for _, entity := range entities {
		if entity != nil {
			out = append(out, entity.ToResponse())
		}
	}
	return out
}
