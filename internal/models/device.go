package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type UserDeviceEntity struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Name          string     `json:"name"`
	DeviceType    string     `json:"device_type"`
	TargetAddress string     `json:"target_address"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

func (e *UserDeviceEntity) FromSqlc(res sqlc.UserDevice) *UserDeviceEntity {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Name = res.Name
	e.DeviceType = res.DeviceType
	e.TargetAddress = res.TargetAddress
	e.CreatedAt = convert.NullTimeToTimePtr(res.CreatedAt)
	e.UpdatedAt = convert.NullTimeToTimePtr(res.UpdatedAt)
	return e
}

type UserDeviceEntities []*UserDeviceEntity

func (e *UserDeviceEntities) FromSqlc(rows []sqlc.UserDevice) []*UserDeviceEntity {
	slice := make([]*UserDeviceEntity, len(rows))
	flat := make([]UserDeviceEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

func (e *UserDeviceEntity) ToResponse() *response.UserDeviceResponse {
	if e == nil {
		return nil
	}
	var createdAt, updatedAt time.Time
	if e.CreatedAt != nil {
		createdAt = *e.CreatedAt
	}
	if e.UpdatedAt != nil {
		updatedAt = *e.UpdatedAt
	}
	return &response.UserDeviceResponse{
		ID:            e.ID,
		UserID:        e.UserID,
		Name:          e.Name,
		DeviceType:    e.DeviceType,
		TargetAddress: e.TargetAddress,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func UserDeviceEntitiesToResponse(entities []*UserDeviceEntity) []*response.UserDeviceResponse {
	out := make([]*response.UserDeviceResponse, 0, len(entities))
	for _, d := range entities {
		if d == nil {
			continue
		}
		out = append(out, d.ToResponse())
	}
	return out
}
