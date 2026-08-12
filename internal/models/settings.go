package models

import "novelhub/internal/gen/sqlc"

type AppSettingEntity struct {
	Key       string `json:"key"`
	ValueJSON string `json:"value_json"`
	UpdatedAt string `json:"updated_at"`
}

type SiteSettings struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Favicon         string `json:"favicon"`
	Logo            string `json:"logo"`
	MetaDescription string `json:"meta_description"`
}

type HomeSectionSettings struct {
	RandomBooks bool `json:"random_books"`
	TopBooks    bool `json:"top_books"`
}

type LibraryPolicy struct {
	Mode         string   `json:"mode"`
	LibraryIDs   []string `json:"library_ids"`
	VisibleStats []string `json:"visible_stats,omitempty"`
}

type PublicSettings struct {
	Site                   SiteSettings        `json:"site"`
	SidebarVisibleItems    []string            `json:"sidebar_visible_items"`
	HomeSections           HomeSectionSettings `json:"home_sections"`
	RegistrationEnabled    bool                `json:"registration_enabled"`
	GuestLoginRequired     bool                `json:"guest_login_required"`
	GuestAccess            LibraryPolicy       `json:"guest_access"`
	GuestPermissions       []string            `json:"guest_permissions"`
	EnableInBookSearch     bool                `json:"enable_in_book_search"`
	EnableCustomFontUpload bool                `json:"enable_custom_font_upload"`
	EnableAniListTracking  bool                `json:"enable_anilist_tracking"`
	EnableAutoEnrich       bool                `json:"enable_auto_enrich"`
	EnableWebpCover        bool                `json:"enable_webp_cover"`
	RequireEmailVerify     bool                `json:"require_email_verify"`
	PasswordResetEnabled   bool                `json:"password_reset_enabled"`
	SMTPEnabled            bool                `json:"smtp_enabled"`
	SetupCompleted         bool                `json:"setup_completed"`
	AvailableSidebarItems  []string            `json:"available_sidebar_items"`
	AvailableHomeSections  []string            `json:"available_home_sections"`
	AvailableGuestModes    []string            `json:"available_guest_modes"`
	ProxyAuthEnabled       bool                `json:"proxy_auth_enabled"`
	OAuth                  OAuthSettingsPublic `json:"oauth"`
}

type OAuthProviderPublic struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Enabled     bool   `json:"enabled"`
}

type OAuthSettingsPublic struct {
	Providers []OAuthProviderPublic `json:"providers"`
}

type OAuthProviderAdmin struct {
	Enabled         bool     `json:"enabled"`
	ClientID        string   `json:"client_id"`
	ClientSecretSet bool     `json:"client_secret_set"`
	RedirectURI     string   `json:"redirect_uri"`
	Name            string   `json:"name,omitempty"`
	IssuerURL       string   `json:"issuer_url,omitempty"`
	Scopes          []string `json:"scopes,omitempty"`
}

type OAuthSettingsAdmin struct {
	Google  OAuthProviderAdmin `json:"google"`
	Github  OAuthProviderAdmin `json:"github"`
	Discord OAuthProviderAdmin `json:"discord"`
	Oidc    OAuthProviderAdmin `json:"oidc"`
}

type ProxyAuthSettings struct {
	Enabled        bool     `json:"enabled"`
	HeaderNames    []string `json:"header_names"`
	TrustedProxies []string `json:"trusted_proxies"`
	AutoCreate     bool     `json:"auto_create"`
}

type RuntimeLimits struct {
	UploadChunkBytes        int64 `json:"upload_chunk_bytes"`
	UploadChunks            int   `json:"upload_chunks"`
	UploadSessions          int   `json:"upload_sessions"`
	UploadBytes             int64 `json:"upload_bytes"`
	UploadSessionTTLSeconds int64 `json:"upload_session_ttl_seconds"`
	CoverBytes              int64 `json:"cover_bytes"`
	SiteAssetBytes          int64 `json:"site_asset_bytes"`
	RateLimitAuth              int   `json:"rate_limit_auth"`
	RateLimitAuthWindowSeconds int64 `json:"rate_limit_auth_window_seconds"`
}

type RuntimeLimitBounds struct {
	Min RuntimeLimits `json:"min"`
	Max RuntimeLimits `json:"max"`
}

type SMTPSettings struct {
	Enabled              bool     `json:"enabled"`
	Host                 string   `json:"host"`
	Port                 int      `json:"port"`
	Username             string   `json:"username"`
	FromEmail            string   `json:"from_email"`
	TLSMode              string   `json:"tls_mode"`
	AllowPrivateNetworks bool     `json:"allow_private_networks"`
	MaxAttachmentMB      int      `json:"max_attachment_mb"`
	PasswordConfigured   bool     `json:"password_configured"`
	AvailableTLSModes    []string `json:"available_tls_modes"`
}

type AdminSettings struct {
	PublicSettings
	Limits    RuntimeLimits      `json:"limits"`
	Bounds    RuntimeLimitBounds `json:"bounds"`
	SMTP      SMTPSettings       `json:"smtp"`
	ServerURL string             `json:"server_url"`
	ProxyAuth ProxyAuthSettings  `json:"proxy_auth"`
	OAuth     OAuthSettingsAdmin `json:"oauth"`
}

func (s *AppSettingEntity) FromSqlc(row sqlc.AppSetting) *AppSettingEntity {
	s.Key = row.Key
	s.ValueJSON = row.ValueJson
	s.UpdatedAt = row.UpdatedAt
	return s
}

type AppSettingEntities []*AppSettingEntity

func (e *AppSettingEntities) FromSqlc(rows []sqlc.AppSetting) []*AppSettingEntity {
	slice := make([]*AppSettingEntity, len(rows))
	flat := make([]AppSettingEntity, len(rows))
	for i, row := range rows {
		slice[i] = flat[i].FromSqlc(row)
	}
	return slice
}

type OAuthProviderConfig struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"`
	Name         string   `json:"name,omitempty"`
	IssuerURL    string   `json:"issuer_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}
