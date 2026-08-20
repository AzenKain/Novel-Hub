package models

import (
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/pkg/convert"
)

type SoundscapeEntity struct {
	ID        string  `json:"id"`
	UserID    *string `json:"user_id,omitempty"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	FilePath  string  `json:"file_path"`
	Icon      string  `json:"icon"`
	Volume    float64 `json:"volume"`
	IsSystem  bool    `json:"is_system"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func (e *SoundscapeEntity) FromSqlc(r sqlc.Soundscape) *SoundscapeEntity {
	e.ID = r.ID
	e.UserID = convert.NullStringToStrPtr(r.UserID)
	e.Name = r.Name
	e.Category = r.Category
	e.FilePath = r.FilePath
	e.Icon = r.Icon
	e.Volume = r.Volume
	e.IsSystem = r.IsSystem == 1
	e.CreatedAt = r.CreatedAt
	e.UpdatedAt = r.UpdatedAt
	return e
}

func (e *SoundscapeEntity) ToResponse() *response.SoundscapeResponse {
	return &response.SoundscapeResponse{
		ID:        e.ID,
		UserID:    e.UserID,
		Name:      e.Name,
		Category:  e.Category,
		FilePath:  e.FilePath,
		StreamURL: "/api/v1/soundscapes/" + e.ID + "/stream",
		Icon:      e.Icon,
		Volume:    e.Volume,
		IsSystem:  e.IsSystem,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type SoundscapeEntities []*SoundscapeEntity

func (e *SoundscapeEntities) FromSqlc(rows []sqlc.Soundscape) []*SoundscapeEntity {
	slice := make([]*SoundscapeEntity, len(rows))
	flat := make([]SoundscapeEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

type CustomFontEntity struct {
	ID         string  `json:"id"`
	UserID     *string `json:"user_id,omitempty"`
	Name       string  `json:"name"`
	FontFamily string  `json:"font_family"`
	SourceType string  `json:"source_type"`
	FilePath   string  `json:"file_path"`
	FontURL    string  `json:"font_url"`
	IsSystem   bool    `json:"is_system"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func (e *CustomFontEntity) FromSqlc(r sqlc.CustomFont) *CustomFontEntity {
	e.ID = r.ID
	e.UserID = convert.NullStringToStrPtr(r.UserID)
	e.Name = r.Name
	e.FontFamily = r.FontFamily
	e.SourceType = r.SourceType
	e.FilePath = r.FilePath
	e.FontURL = r.FontUrl
	e.IsSystem = r.IsSystem == 1
	e.CreatedAt = r.CreatedAt
	e.UpdatedAt = r.UpdatedAt
	return e
}

func (e *CustomFontEntity) ToResponse() *response.CustomFontResponse {
	fileURL := ""
	if e.SourceType == "file" && e.FilePath != "" {
		fileURL = "/api/v1/fonts/" + e.ID + "/file"
	}
	return &response.CustomFontResponse{
		ID:         e.ID,
		UserID:     e.UserID,
		Name:       e.Name,
		FontFamily: e.FontFamily,
		SourceType: e.SourceType,
		FileURL:    fileURL,
		FontURL:    e.FontURL,
		IsSystem:   e.IsSystem,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

type CustomFontEntities []*CustomFontEntity

func (e *CustomFontEntities) FromSqlc(rows []sqlc.CustomFont) []*CustomFontEntity {
	slice := make([]*CustomFontEntity, len(rows))
	flat := make([]CustomFontEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

type CustomThemeEntity struct {
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

func (e *CustomThemeEntity) FromSqlc(r sqlc.CustomTheme) *CustomThemeEntity {
	e.ID = r.ID
	e.UserID = convert.NullStringToStrPtr(r.UserID)
	e.Name = r.Name
	e.BgColor = r.BgColor
	e.TextColor = r.TextColor
	e.AccentColor = r.AccentColor
	e.CustomCss = r.CustomCss
	e.IsSystem = r.IsSystem == 1
	e.CreatedAt = r.CreatedAt
	e.UpdatedAt = r.UpdatedAt
	return e
}

func (e *CustomThemeEntity) ToResponse() *response.CustomThemeResponse {
	return &response.CustomThemeResponse{
		ID:          e.ID,
		UserID:      e.UserID,
		Name:        e.Name,
		BgColor:     e.BgColor,
		TextColor:   e.TextColor,
		AccentColor: e.AccentColor,
		CustomCss:   e.CustomCss,
		IsSystem:    e.IsSystem,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

type CustomThemeEntities []*CustomThemeEntity

func (e *CustomThemeEntities) FromSqlc(rows []sqlc.CustomTheme) []*CustomThemeEntity {
	slice := make([]*CustomThemeEntity, len(rows))
	flat := make([]CustomThemeEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}
