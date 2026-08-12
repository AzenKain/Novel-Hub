package response

import "time"

type ContentWarningResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type KidsModeInfoResponse struct {
	ID                  string `json:"id"`
	IsKidsMode          bool   `json:"is_kids_mode"`
	HasPin              bool   `json:"has_pin"`
	MaxAllowedAgeRating string `json:"max_allowed_age_rating"`
}
