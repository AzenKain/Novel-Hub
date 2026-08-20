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
	HardcoverEnabled        *bool                   `json:"hardcover.enabled"`
	HardcoverClientID       *string                 `json:"hardcover.client_id" validate:"omitempty,max=500"`
	HardcoverClientSecret   *string                 `json:"hardcover.client_secret" validate:"omitempty,max=1000"`
	EnableAutoEnrich        *bool                   `json:"metadata.auto_enrich_enabled"`
	EnableWebpCover         *bool                   `json:"metadata.webp_cover_enabled"`
	RequireEmailVerify      *bool                   `json:"auth.require_email_verify"`
	PasswordResetEnabled    *bool                   `json:"auth.password_reset_enabled"`
	UploadChunkBytes        *int64                  `json:"limits.upload_chunk_bytes"`
	UploadChunks            *int                    `json:"limits.upload_chunks"`
	UploadSessions          *int                    `json:"limits.upload_sessions"`
	UploadBytes             *int64                  `json:"limits.upload_bytes"`
	UploadSessionTTLSeconds *int64                  `json:"limits.upload_session_ttl_seconds"`
	CoverBytes              *int64                  `json:"limits.cover_bytes"`
	SiteAssetBytes          *int64                  `json:"limits.site_asset_bytes"`
	SoundscapeBytes         *int64                  `json:"limits.soundscape_bytes"`
	FontBytes               *int64                  `json:"limits.font_bytes"`

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
	SMTPMaxAttachmentMB      *int    `json:"smtp.max_attachment_mb" validate:"omitempty,min=1,max=500"`

	ProxyAuthEnabled        *bool     `json:"auth.proxy_auth_enabled"`
	ProxyAuthHeaders        *[]string `json:"auth.proxy_auth_headers"`
	ProxyAuthTrustedProxies *[]string `json:"auth.proxy_auth_trusted_proxies"`
	ProxyAuthAutoCreate     *bool     `json:"auth.proxy_auth_auto_create"`

	OAuthGoogleEnabled      *bool     `json:"oauth.google.enabled"`
	OAuthGoogleClientID     *string   `json:"oauth.google.client_id" validate:"omitempty,max=500"`
	OAuthGoogleClientSecret *string   `json:"oauth.google.client_secret" validate:"omitempty,max=1000"`
	OAuthGoogleRedirectURI  *string   `json:"oauth.google.redirect_uri" validate:"omitempty,max=2048"`

	OAuthGithubEnabled      *bool     `json:"oauth.github.enabled"`
	OAuthGithubClientID     *string   `json:"oauth.github.client_id" validate:"omitempty,max=500"`
	OAuthGithubClientSecret *string   `json:"oauth.github.client_secret" validate:"omitempty,max=1000"`
	OAuthGithubRedirectURI  *string   `json:"oauth.github.redirect_uri" validate:"omitempty,max=2048"`

	OAuthDiscordEnabled      *bool     `json:"oauth.discord.enabled"`
	OAuthDiscordClientID     *string   `json:"oauth.discord.client_id" validate:"omitempty,max=500"`
	OAuthDiscordClientSecret *string   `json:"oauth.discord.client_secret" validate:"omitempty,max=1000"`
	OAuthDiscordRedirectURI  *string   `json:"oauth.discord.redirect_uri" validate:"omitempty,max=2048"`

	OAuthOidcEnabled      *bool     `json:"oauth.oidc.enabled"`
	OAuthOidcName         *string   `json:"oauth.oidc.name" validate:"omitempty,max=200"`
	OAuthOidcIssuerURL    *string   `json:"oauth.oidc.issuer_url" validate:"omitempty,max=2048"`
	OAuthOidcClientID     *string   `json:"oauth.oidc.client_id" validate:"omitempty,max=500"`
	OAuthOidcClientSecret *string   `json:"oauth.oidc.client_secret" validate:"omitempty,max=1000"`
	OAuthOidcRedirectURI  *string   `json:"oauth.oidc.redirect_uri" validate:"omitempty,max=2048"`
	OAuthOidcScopes       *[]string `json:"oauth.oidc.scopes" validate:"omitempty,max=50"`

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
	putPtr(values, "hardcover.enabled", d.HardcoverEnabled)
	putPtr(values, "hardcover.client_id", d.HardcoverClientID)
	putPtr(values, "hardcover.client_secret", d.HardcoverClientSecret)
	putPtr(values, "metadata.auto_enrich_enabled", d.EnableAutoEnrich)
	putPtr(values, "metadata.webp_cover_enabled", d.EnableWebpCover)
	putPtr(values, "auth.require_email_verify", d.RequireEmailVerify)
	putPtr(values, "auth.password_reset_enabled", d.PasswordResetEnabled)
	putPtr(values, "limits.upload_chunk_bytes", d.UploadChunkBytes)
	putPtr(values, "limits.upload_chunks", d.UploadChunks)
	putPtr(values, "limits.upload_sessions", d.UploadSessions)
	putPtr(values, "limits.upload_bytes", d.UploadBytes)
	putPtr(values, "limits.upload_session_ttl_seconds", d.UploadSessionTTLSeconds)
	putPtr(values, "limits.cover_bytes", d.CoverBytes)
	putPtr(values, "limits.site_asset_bytes", d.SiteAssetBytes)
	putPtr(values, "limits.soundscape_bytes", d.SoundscapeBytes)
	putPtr(values, "limits.font_bytes", d.FontBytes)
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
	putPtr(values, "smtp.max_attachment_mb", d.SMTPMaxAttachmentMB)
	putPtr(values, "auth.proxy_auth_enabled", d.ProxyAuthEnabled)
	putPtr(values, "auth.proxy_auth_headers", d.ProxyAuthHeaders)
	putPtr(values, "auth.proxy_auth_trusted_proxies", d.ProxyAuthTrustedProxies)
	putPtr(values, "auth.proxy_auth_auto_create", d.ProxyAuthAutoCreate)

	putPtr(values, "oauth.google.enabled", d.OAuthGoogleEnabled)
	putPtr(values, "oauth.google.client_id", d.OAuthGoogleClientID)
	putPtr(values, "oauth.google.client_secret", d.OAuthGoogleClientSecret)
	putPtr(values, "oauth.google.redirect_uri", d.OAuthGoogleRedirectURI)

	putPtr(values, "oauth.github.enabled", d.OAuthGithubEnabled)
	putPtr(values, "oauth.github.client_id", d.OAuthGithubClientID)
	putPtr(values, "oauth.github.client_secret", d.OAuthGithubClientSecret)
	putPtr(values, "oauth.github.redirect_uri", d.OAuthGithubRedirectURI)

	putPtr(values, "oauth.discord.enabled", d.OAuthDiscordEnabled)
	putPtr(values, "oauth.discord.client_id", d.OAuthDiscordClientID)
	putPtr(values, "oauth.discord.client_secret", d.OAuthDiscordClientSecret)
	putPtr(values, "oauth.discord.redirect_uri", d.OAuthDiscordRedirectURI)

	putPtr(values, "oauth.oidc.enabled", d.OAuthOidcEnabled)
	putPtr(values, "oauth.oidc.name", d.OAuthOidcName)
	putPtr(values, "oauth.oidc.issuer_url", d.OAuthOidcIssuerURL)
	putPtr(values, "oauth.oidc.client_id", d.OAuthOidcClientID)
	putPtr(values, "oauth.oidc.client_secret", d.OAuthOidcClientSecret)
	putPtr(values, "oauth.oidc.redirect_uri", d.OAuthOidcRedirectURI)
	putPtr(values, "oauth.oidc.scopes", d.OAuthOidcScopes)

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
		"hardcover.enabled":        true,
		"hardcover.client_id":      true,
		"hardcover.client_secret":  true,
		"metadata.auto_enrich_enabled": true,
		"metadata.webp_cover_enabled":  true,
		"auth.require_email_verify": true, "auth.password_reset_enabled": true,
		"limits.upload_chunk_bytes": true, "limits.upload_chunks": true, "limits.upload_sessions": true,
		"limits.upload_bytes": true, "limits.upload_session_ttl_seconds": true,
		"limits.cover_bytes": true, "limits.site_asset_bytes": true,
		"limits.soundscape_bytes": true, "limits.font_bytes": true,
		"limits.rate_limit_auth": true, "limits.rate_limit_auth_window_seconds": true,
		"smtp.enabled": true, "smtp.host": true, "smtp.port": true, "smtp.username": true,
		"smtp.password": true, "smtp.from_email": true, "smtp.tls_mode": true,
		"smtp.allow_private_networks": true, "smtp.max_attachment_mb": true,
		"auth.proxy_auth_enabled":         true,
		"auth.proxy_auth_headers":         true,
		"auth.proxy_auth_trusted_proxies": true,
		"auth.proxy_auth_auto_create":      true,
		"oauth.google.enabled":            true,
		"oauth.google.client_id":          true,
		"oauth.google.client_secret":      true,
		"oauth.google.redirect_uri":       true,
		"oauth.github.enabled":            true,
		"oauth.github.client_id":          true,
		"oauth.github.client_secret":      true,
		"oauth.github.redirect_uri":       true,
		"oauth.discord.enabled":           true,
		"oauth.discord.client_id":         true,
		"oauth.discord.client_secret":     true,
		"oauth.discord.redirect_uri":      true,
		"oauth.oidc.enabled":              true,
		"oauth.oidc.name":                 true,
		"oauth.oidc.issuer_url":           true,
		"oauth.oidc.client_id":            true,
		"oauth.oidc.client_secret":        true,
		"oauth.oidc.redirect_uri":         true,
		"oauth.oidc.scopes":               true,
	}
	unknown := make([]string, 0)
	for key := range d.present {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}
