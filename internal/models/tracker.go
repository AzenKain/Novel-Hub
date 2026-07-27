package models

import (
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/crypto"
)

type UserTrackerEntity struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	Provider     string     `json:"provider"`
	AccessToken  string     `json:"-"`
	RefreshToken *string    `json:"-"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (e *UserTrackerEntity) FromSqlc(res sqlc.UserTracker) (*UserTrackerEntity, error) {
	e.ID = res.ID
	e.UserID = res.UserID
	e.Provider = res.Provider
	decrypted, err := crypto.DecryptAES(res.AccessToken)
	if err != nil {
		return nil, err
	}
	e.AccessToken = decrypted
	if res.RefreshToken.Valid {
		refreshToken, err := crypto.DecryptAES(res.RefreshToken.String)
		if err != nil {
			return nil, err
		}
		e.RefreshToken = &refreshToken
	}
	if res.ExpiresAt.Valid {
		e.ExpiresAt = &res.ExpiresAt.Time
	}
	e.CreatedAt = res.CreatedAt
	e.UpdatedAt = res.UpdatedAt
	return e, nil
}

func (e *UserTrackerEntity) ToResponse() *response.UserTrackerResponse {
	if e == nil {
		return nil
	}
	return &response.UserTrackerResponse{
		ID:        e.ID,
		UserID:    e.UserID,
		Provider:  e.Provider,
		ExpiresAt: e.ExpiresAt,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type BookTrackerMappingEntity struct {
	ID               string    `json:"id"`
	BookID           string    `json:"bookId"`
	Provider         string    `json:"provider"`
	ExternalSeriesID string    `json:"externalSeriesId"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (e *BookTrackerMappingEntity) FromSqlc(res sqlc.BookTrackerMapping) *BookTrackerMappingEntity {
	e.ID = res.ID
	e.BookID = res.BookID
	e.Provider = res.Provider
	e.ExternalSeriesID = res.ExternalSeriesID
	e.CreatedAt = res.CreatedAt
	return e
}

func (e *BookTrackerMappingEntity) ToResponse() *response.BookTrackerMappingResponse {
	if e == nil {
		return nil
	}
	return &response.BookTrackerMappingResponse{
		ID:               e.ID,
		BookID:           e.BookID,
		Provider:         e.Provider,
		ExternalSeriesID: e.ExternalSeriesID,
		CreatedAt:        e.CreatedAt,
	}
}
