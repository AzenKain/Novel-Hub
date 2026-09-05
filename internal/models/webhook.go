package models

import (
	"strings"
	"time"

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type WebhookEntity struct {
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

func (e *WebhookEntity) FromSqlc(row sqlc.Webhook) *WebhookEntity {
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

	e.ID = row.ID
	e.Name = row.Name
	e.URL = row.Url
	e.TemplateType = row.TemplateType
	e.Secret = convert.NullStringToStrPtr(row.Secret)
	e.CustomHeaders = convert.NullStringToStrPtr(row.CustomHeaders)
	e.Events = events
	e.IsActive = row.IsActive == 1
	e.CreatedAt = row.CreatedAt.Time
	e.UpdatedAt = row.UpdatedAt.Time

	return e
}

func (e *WebhookEntity) ToResponse() *response.WebhookResponse {
	return &response.WebhookResponse{
		ID:            e.ID,
		Name:          e.Name,
		URL:           e.URL,
		TemplateType:  e.TemplateType,
		Secret:        e.Secret,
		CustomHeaders: e.CustomHeaders,
		Events:        e.Events,
		IsActive:      e.IsActive,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

type WebhookEntities []*WebhookEntity

func (e *WebhookEntities) FromSqlc(rows []sqlc.Webhook) []*WebhookEntity {
	slice := make([]*WebhookEntity, len(rows))
	flat := make([]WebhookEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}
