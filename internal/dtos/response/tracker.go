package response

import "time"

type UserTrackerResponse struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Provider  string     `json:"provider"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type BookTrackerMappingResponse struct {
	ID               int64     `json:"id"`
	BookID           int64     `json:"book_id"`
	Provider         string    `json:"provider"`
	ExternalSeriesID string    `json:"external_series_id"`
	CreatedAt        time.Time `json:"created_at"`
}
