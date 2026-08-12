package response

import "time"

type WebhookResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	TemplateType  string    `json:"template_type"`
	Secret        *string   `json:"secret,omitempty"`
	CustomHeaders *string   `json:"custom_headers,omitempty"`
	Events        []string  `json:"events"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
