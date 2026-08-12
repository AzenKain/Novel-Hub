package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type MagicCodeEntity struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	PollToken  string    `json:"poll_token"`
	DeviceInfo string    `json:"device_info"`
	UserID     *string   `json:"user_id,omitempty"`
	JWTToken   string    `json:"jwt_token"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (e *MagicCodeEntity) FromSqlc(row sqlc.MagicCode) *MagicCodeEntity {
	e.ID = row.ID
	e.Code = row.Code
	e.PollToken = row.PollToken
	e.DeviceInfo = row.DeviceInfo
	e.UserID = convert.NullStringToStrPtr(row.UserID)
	e.JWTToken = row.JwtToken
	e.Status = row.Status
	e.ExpiresAt = row.ExpiresAt
	e.CreatedAt = row.CreatedAt
	return e
}

func (e *MagicCodeEntity) ToResponse() *response.MagicCodeResponse {
	if e == nil {
		return nil
	}
	return &response.MagicCodeResponse{
		ID:         e.ID,
		Code:       e.Code,
		PollToken:  e.PollToken,
		DeviceInfo: e.DeviceInfo,
		UserID:     e.UserID,
		JWTToken:   e.JWTToken,
		Status:     e.Status,
		ExpiresAt:  e.ExpiresAt,
		CreatedAt:  e.CreatedAt,
	}
}
