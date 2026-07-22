package models

import (
	"strings"
	"time"

	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type WebhookEntity struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	TemplateType  string     `json:"template_type"`
	Secret        *string    `json:"secret,omitempty"`
	CustomHeaders *string    `json:"custom_headers,omitempty"`
	Events        []string   `json:"events"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func WebhookFromSqlc(row sqlc.Webhook) *WebhookEntity {
	var events []string
	if strings.TrimSpace(row.Events) != "" {
		parts := strings.Split(row.Events, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				events = append(events, trimmed)
			}
		}
	}
	if events == nil {
		events = []string{}
	}

	return &WebhookEntity{
		ID:            row.ID,
		Name:          row.Name,
		URL:           row.Url,
		TemplateType:  row.TemplateType,
		Secret:        convert.NullStringToStrPtr(row.Secret),
		CustomHeaders: convert.NullStringToStrPtr(row.CustomHeaders),
		Events:        events,
		IsActive:      row.IsActive == 1,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}
