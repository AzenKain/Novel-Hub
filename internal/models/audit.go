package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type AuditLogEntity struct {
	ID          string    `json:"id"`
	ActorID     *string   `json:"actor_id,omitempty"`
	ActorEmail  string    `json:"actor_email"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    *string   `json:"target_id,omitempty"`
	TargetLabel string    `json:"target_label"`
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}

func AuditLogFromSqlc(row sqlc.AuditLog) *AuditLogEntity {
	return &AuditLogEntity{
		ID:          row.ID,
		ActorID:     convert.NullStringToStrPtr(row.ActorID),
		ActorEmail:  row.ActorEmail,
		Action:      row.Action,
		TargetType:  row.TargetType,
		TargetID:    convert.NullStringToStrPtr(row.TargetID),
		TargetLabel: row.TargetLabel,
		IP:          row.Ip,
		CreatedAt:   row.CreatedAt.Time,
	}
}

func (e *AuditLogEntity) ToResponse() *response.AuditLogResponse {
	return &response.AuditLogResponse{
		ID:          e.ID,
		ActorID:     e.ActorID,
		ActorEmail:  e.ActorEmail,
		Action:      e.Action,
		TargetType:  e.TargetType,
		TargetID:    e.TargetID,
		TargetLabel: e.TargetLabel,
		IP:          e.IP,
		CreatedAt:   e.CreatedAt,
	}
}

func AuditLogEntitiesToResponse(entities []*AuditLogEntity) []*response.AuditLogResponse {
	out := make([]*response.AuditLogResponse, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		out = append(out, entity.ToResponse())
	}
	return out
}
