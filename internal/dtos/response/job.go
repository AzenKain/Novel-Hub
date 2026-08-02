package response

import "time"

type JobResponse struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      *string   `json:"status,omitempty"`
	Progress    *int64    `json:"progress,omitempty"`
	Total       *int64    `json:"total,omitempty"`
	ErrorMsg    *string   `json:"error_msg,omitempty"`
	PayloadJSON *string   `json:"payload_json,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
