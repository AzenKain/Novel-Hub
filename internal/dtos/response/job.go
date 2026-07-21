package response

import "time"

type JobResponse struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      *string   `json:"status,omitempty"`
	Progress    *int64    `json:"progress,omitempty"`
	Total       *int64    `json:"total,omitempty"`
	ErrorMsg    *string   `json:"errorMsg,omitempty"`
	PayloadJSON *string   `json:"payloadJson,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
