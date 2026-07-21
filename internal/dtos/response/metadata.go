package response

import "time"

type SeriesResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type PublisherResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type LanguageResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type MetadataCountResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BookCount int64  `json:"bookCount"`
	CoverURL  string `json:"coverUrl,omitempty"`
}

type AuthorResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Bio       *string   `json:"bio,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TagResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
