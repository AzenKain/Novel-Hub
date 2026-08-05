package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
)

type TOTPEntity struct {
	UserID      string     `json:"user_id"`
	Secret      string     `json:"secret"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (e *TOTPEntity) IsConfirmed() bool {
	return e != nil && e.ConfirmedAt != nil
}

func TOTPFromSqlc(row sqlc.UserTotp) *TOTPEntity {
	entity := &TOTPEntity{
		UserID:    row.UserID,
		Secret:    row.Secret,
		CreatedAt: row.CreatedAt,
	}
	if row.ConfirmedAt.Valid {
		confirmed := row.ConfirmedAt.Time
		entity.ConfirmedAt = &confirmed
	}
	return entity
}
