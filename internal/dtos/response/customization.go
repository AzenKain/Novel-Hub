package response

type SoundscapeResponse struct {
	ID        string  `json:"id"`
	UserID    *string `json:"user_id,omitempty"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	FilePath  string  `json:"file_path"`
	StreamURL string  `json:"stream_url"`
	Icon      string  `json:"icon"`
	Volume    float64 `json:"volume"`
	IsSystem  bool    `json:"is_system"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type CustomFontResponse struct {
	ID         string  `json:"id"`
	UserID     *string `json:"user_id,omitempty"`
	Name       string  `json:"name"`
	FontFamily string  `json:"font_family"`
	SourceType string  `json:"source_type"`
	FileURL    string  `json:"file_url,omitempty"`
	FontURL    string  `json:"font_url,omitempty"`
	IsSystem   bool    `json:"is_system"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type CustomThemeResponse struct {
	ID          string  `json:"id"`
	UserID      *string `json:"user_id,omitempty"`
	Name        string  `json:"name"`
	BgColor     string  `json:"bg_color"`
	TextColor   string  `json:"text_color"`
	AccentColor string  `json:"accent_color"`
	CustomCss   string  `json:"custom_css"`
	IsSystem    bool    `json:"is_system"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
