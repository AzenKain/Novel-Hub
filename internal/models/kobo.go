package models

import (
	"time"

	"novelhub/internal/gen/sqlc"
)

// KoboAuthTokenEntity is the path token a Kobo device authenticates with. The token itself
// is a credential, so it is never included in a ToResponse payload aimed at anyone other
// than its owner — see internal/services/koboAuthService.go.
type KoboAuthTokenEntity struct {
	Token      string
	UserID     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func (e *KoboAuthTokenEntity) FromSqlc(row sqlc.KoboAuthToken) *KoboAuthTokenEntity {
	e.Token = row.Token
	e.UserID = row.UserID
	e.CreatedAt = row.CreatedAt
	if row.LastUsedAt.Valid {
		lastUsed := row.LastUsedAt.Time
		e.LastUsedAt = &lastUsed
	}
	return e
}
