package request

import (
	"errors"
	"sort"

	"novelhub/pkg/jsonx"
)

type HomeSectionSettingsDto struct {
	RandomBooks *bool `json:"random_books"`
	TopBooks    *bool `json:"top_books"`
}

type UpdateSettingsDto struct {
	SiteTitle               *string                 `json:"site.title" validate:"omitempty,max=200"`
	SiteDescription         *string                 `json:"site.description" validate:"omitempty,max=1000"`
	SiteFavicon             *string                 `json:"site.favicon" validate:"omitempty,max=2048"`
	SiteLogo                *string                 `json:"site.logo" validate:"omitempty,max=2048"`
	SiteMetaDescription     *string                 `json:"site.meta_description" validate:"omitempty,max=1000"`
	ServerURL               *string                 `json:"server.url" validate:"omitempty,url,max=2048"`
	SidebarVisibleItems     *[]string               `json:"sidebar.visible_items" validate:"omitempty,max=100"`
	HomeSections            *HomeSectionSettingsDto `json:"home.sections"`
	RegistrationEnabled     *bool                   `json:"auth.registration_enabled"`
	LoginRequired           *bool                   `json:"auth.login_required"`
	GuestAccessMode         *string                 `json:"guest_access.mode" validate:"omitempty,oneof=all selected_libraries"`
	GuestAccessLibraryIDs   *[]string               `json:"guest_access.library_ids" validate:"omitempty,max=100"`
	EnableInBookSearch      *bool                   `json:"reader.enable_in_book_search"`
	EnableCustomFontUpload  *bool                   `json:"font.enable_custom_font_upload"`
	EnableAniListTracking   *bool                   `json:"tracker.anilist_enabled"`
	RequireEmailVerify      *bool                   `json:"auth.require_email_verify"`
	PasswordResetEnabled    *bool                   `json:"auth.password_reset_enabled"`
	UploadChunkBytes        *int64                  `json:"limits.upload_chunk_bytes"`
	UploadChunks            *int                    `json:"limits.upload_chunks"`
	UploadSessions          *int                    `json:"limits.upload_sessions"`
	UploadBytes             *int64                  `json:"limits.upload_bytes"`
	UploadSessionTTLSeconds *int64                  `json:"limits.upload_session_ttl_seconds"`
	CoverBytes              *int64                  `json:"limits.cover_bytes"`
	SiteAssetBytes          *int64                  `json:"limits.site_asset_bytes"`

	RateLimitAuth              *int   `json:"limits.rate_limit_auth"`
	RateLimitAuthWindowSeconds *int64 `json:"limits.rate_limit_auth_window_seconds"`

	SMTPEnabled              *bool   `json:"smtp.enabled"`
	SMTPHost                 *string `json:"smtp.host" validate:"omitempty,max=255"`
	SMTPPort                 *int    `json:"smtp.port" validate:"omitempty,min=1,max=65535"`
	SMTPUsername             *string `json:"smtp.username" validate:"omitempty,max=255"`
	SMTPPassword             *string `json:"smtp.password" validate:"omitempty,max=1024"`
	SMTPFromEmail            *string `json:"smtp.from_email" validate:"omitempty,max=255"`
	SMTPTLSMode              *string `json:"smtp.tls_mode" validate:"omitempty,oneof=none starttls implicit_tls"`
	SMTPAllowPrivateNetworks *bool   `json:"smtp.allow_private_networks"`

	present map[string]bool
}

type SMTPTestDto struct {
	Host                 *string `json:"host" validate:"omitempty,max=255"`
	Port                 *int    `json:"port" validate:"omitempty,min=1,max=65535"`
	Username             *string `json:"username" validate:"omitempty,max=255"`
	Password             *string `json:"password" validate:"omitempty,max=1024"`
	FromEmail            *string `json:"from_email" validate:"omitempty,max=255"`
	TLSMode              *string `json:"tls_mode" validate:"omitempty,oneof=none starttls implicit_tls"`
	AllowPrivateNetworks *bool   `json:"allow_private_networks"`
}

func (d *UpdateSettingsDto) UnmarshalJSON(data []byte) error {
	type plain UpdateSettingsDto
	var decoded plain
	if err := jsonx.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]any
	if err := jsonx.Unmarshal(data, &raw); err != nil {
		return err
	}
	*d = UpdateSettingsDto(decoded)
	d.present = make(map[string]bool, len(raw))
	for key := range raw {
		d.present[key] = true
	}
	if unknown := d.UnknownKeys(); len(unknown) > 0 {
		sort.Strings(unknown)
		return errors.New("unsupported setting: " + unknown[0])
	}
	return nil
}

// putPtr stores the pointed-to value, or nothing when the field was absent from
// the request body. It is generic on purpose: a `func(string, any)` helper would
// box a typed nil (*string)(nil) into a non-nil interface, so every unset field
// would be forwarded downstream and dereferenced into a panic.
func putPtr[T any](values map[string]any, key string, value *T) {
	if value != nil {
		values[key] = *value
	}
}

func (d *UpdateSettingsDto) Values() map[string]any {
	values := make(map[string]any)
	putPtr(values, "site.title", d.SiteTitle)
	putPtr(values, "site.description", d.SiteDescription)
	putPtr(values, "site.favicon", d.SiteFavicon)
	putPtr(values, "site.logo", d.SiteLogo)
	putPtr(values, "site.meta_description", d.SiteMetaDescription)
	putPtr(values, "server.url", d.ServerURL)
	putPtr(values, "sidebar.visible_items", d.SidebarVisibleItems)
	if d.HomeSections != nil {
		sections := make(map[string]any, 2)
		putPtr(sections, "random_books", d.HomeSections.RandomBooks)
		putPtr(sections, "top_books", d.HomeSections.TopBooks)
		values["home.sections"] = sections
	}
	putPtr(values, "auth.registration_enabled", d.RegistrationEnabled)
	putPtr(values, "auth.login_required", d.LoginRequired)
	putPtr(values, "guest_access.mode", d.GuestAccessMode)
	putPtr(values, "guest_access.library_ids", d.GuestAccessLibraryIDs)
	putPtr(values, "reader.enable_in_book_search", d.EnableInBookSearch)
	putPtr(values, "font.enable_custom_font_upload", d.EnableCustomFontUpload)
	putPtr(values, "tracker.anilist_enabled", d.EnableAniListTracking)
	putPtr(values, "auth.require_email_verify", d.RequireEmailVerify)
	putPtr(values, "auth.password_reset_enabled", d.PasswordResetEnabled)
	putPtr(values, "limits.upload_chunk_bytes", d.UploadChunkBytes)
	putPtr(values, "limits.upload_chunks", d.UploadChunks)
	putPtr(values, "limits.upload_sessions", d.UploadSessions)
	putPtr(values, "limits.upload_bytes", d.UploadBytes)
	putPtr(values, "limits.upload_session_ttl_seconds", d.UploadSessionTTLSeconds)
	putPtr(values, "limits.cover_bytes", d.CoverBytes)
	putPtr(values, "limits.site_asset_bytes", d.SiteAssetBytes)
	putPtr(values, "limits.rate_limit_auth", d.RateLimitAuth)
	putPtr(values, "limits.rate_limit_auth_window_seconds", d.RateLimitAuthWindowSeconds)
	putPtr(values, "smtp.enabled", d.SMTPEnabled)
	putPtr(values, "smtp.host", d.SMTPHost)
	putPtr(values, "smtp.port", d.SMTPPort)
	putPtr(values, "smtp.username", d.SMTPUsername)
	putPtr(values, "smtp.password", d.SMTPPassword)
	putPtr(values, "smtp.from_email", d.SMTPFromEmail)
	putPtr(values, "smtp.tls_mode", d.SMTPTLSMode)
	putPtr(values, "smtp.allow_private_networks", d.SMTPAllowPrivateNetworks)
	return values
}

func (d *UpdateSettingsDto) UnknownKeys() []string {
	known := map[string]bool{
		"site.title": true, "site.description": true, "site.favicon": true, "site.logo": true,
		"site.meta_description": true, "sidebar.visible_items": true, "home.sections": true,
		"server.url":                true,
		"auth.registration_enabled": true, "auth.login_required": true, "guest_access.mode": true, "guest_access.library_ids": true,
		"reader.enable_in_book_search": true, "font.enable_custom_font_upload": true,
		"tracker.anilist_enabled":   true,
		"auth.require_email_verify": true, "auth.password_reset_enabled": true,
		"limits.upload_chunk_bytes": true, "limits.upload_chunks": true, "limits.upload_sessions": true,
		"limits.upload_bytes": true, "limits.upload_session_ttl_seconds": true,
		"limits.cover_bytes": true, "limits.site_asset_bytes": true,
		"limits.rate_limit_auth": true, "limits.rate_limit_auth_window_seconds": true,
		"smtp.enabled": true, "smtp.host": true, "smtp.port": true, "smtp.username": true,
		"smtp.password": true, "smtp.from_email": true, "smtp.tls_mode": true,
		"smtp.allow_private_networks": true,
	}
	unknown := make([]string, 0)
	for key := range d.present {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}
