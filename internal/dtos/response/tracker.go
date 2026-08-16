package response

import "time"

type UserTrackerResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Provider  string     `json:"provider"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Connection state per provider; tokens are never included.
type TrackerConnectionResponse struct {
	Provider  string     `json:"provider"`
	Connected bool       `json:"connected"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// One selectable AniList media candidate.
type TrackerSearchResultResponse struct {
	ExternalSeriesID string `json:"external_series_id"`
	TitleEnglish     string `json:"title_english,omitempty"`
	TitleRomaji      string `json:"title_romaji,omitempty"`
	MediaType        string `json:"media_type"`
}

type BookTrackerMappingResponse struct {
	ID               string    `json:"id"`
	BookID           string    `json:"book_id"`
	Provider         string    `json:"provider"`
	ExternalSeriesID string    `json:"external_series_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type TrackerSearchListResponse struct {
	Results []TrackerSearchResultResponse `json:"results"`
}
