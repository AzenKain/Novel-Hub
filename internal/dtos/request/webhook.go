package request

type CreateWebhookDto struct {
	Name          string   `json:"name" validate:"required,min=1,max=100"`
	URL           string   `json:"url" validate:"required,url"`
	TemplateType  string   `json:"template_type" validate:"oneof=generic discord telegram slack"`
	Secret        *string  `json:"secret,omitempty"`
	CustomHeaders *string  `json:"custom_headers,omitempty"`
	Events        []string `json:"events" validate:"required,min=1"`
	IsActive      *bool    `json:"is_active,omitempty"`
}

type UpdateWebhookDto struct {
	Name          string   `json:"name" validate:"required,min=1,max=100"`
	URL           string   `json:"url" validate:"required,url"`
	TemplateType  string   `json:"template_type" validate:"oneof=generic discord telegram slack"`
	Secret        *string  `json:"secret,omitempty"`
	CustomHeaders *string  `json:"custom_headers,omitempty"`
	Events        []string `json:"events" validate:"required,min=1"`
	IsActive      bool     `json:"is_active"`
}
