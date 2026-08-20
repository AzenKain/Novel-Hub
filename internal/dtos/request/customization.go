package request

type ListCustomizationQueryDto struct {
	Cursor string `query:"cursor" validate:"omitempty"`
	Limit  int64  `query:"limit" validate:"omitempty,min=1,max=100"`
}

type UploadSoundscapeDto struct {
	Name     string  `json:"name" form:"name" validate:"required,min=1,max=100"`
	Category string  `json:"category" form:"category" validate:"omitempty,max=50"`
	Icon     string  `json:"icon" form:"icon" validate:"omitempty,max=50"`
	Volume   float64 `json:"volume" form:"volume" validate:"omitempty,min=0,max=1"`
	AudioURL string  `json:"audio_url" form:"audio_url" validate:"omitempty"`
	IsSystem bool    `json:"is_system" form:"is_system"`
}

type UploadFontDto struct {
	Name       string `json:"name" form:"name" validate:"required,min=1,max=100"`
	FontFamily string `json:"font_family" form:"font_family" validate:"required,min=1,max=100"`
	SourceType string `json:"source_type" form:"source_type" validate:"required,oneof=file url"`
	FontURL    string `json:"font_url" form:"font_url" validate:"omitempty"`
	IsSystem   bool   `json:"is_system" form:"is_system"`
}

type CreateCustomThemeDto struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	BgColor     string `json:"bg_color" validate:"required,min=3,max=30"`
	TextColor   string `json:"text_color" validate:"required,min=3,max=30"`
	AccentColor string `json:"accent_color" validate:"required,min=3,max=30"`
	CustomCss   string `json:"custom_css" validate:"omitempty,max=10000"`
	IsSystem    bool   `json:"is_system"`
}

type UpdateCustomThemeDto struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	BgColor     string `json:"bg_color" validate:"required,min=3,max=30"`
	TextColor   string `json:"text_color" validate:"required,min=3,max=30"`
	AccentColor string `json:"accent_color" validate:"required,min=3,max=30"`
	CustomCss   string `json:"custom_css" validate:"omitempty,max=10000"`
}
