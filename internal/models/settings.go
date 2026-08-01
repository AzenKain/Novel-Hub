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
	GuestAccess            LibraryPolicy       `json:"guest_access"`
	GuestPermissions       []string            `json:"guest_permissions"`
	EnableInBookSearch     bool                `json:"enable_in_book_search"`
	EnableCustomFontUpload bool                `json:"enable_custom_font_upload"`
	EnableAniListTracking  bool                `json:"enable_anilist_tracking"`
	SetupCompleted         bool                `json:"setup_completed"`
	AvailableSidebarItems  []string            `json:"available_sidebar_items"`
	AvailableHomeSections  []string            `json:"available_home_sections"`
	AvailableGuestModes    []string            `json:"available_guest_modes"`
}

type RuntimeLimits struct {
	UploadChunkBytes        int64 `json:"upload_chunk_bytes"`
	UploadChunks            int   `json:"upload_chunks"`
	UploadSessions          int   `json:"upload_sessions"`
	UploadBytes             int64 `json:"upload_bytes"`
	UploadSessionTTLSeconds int64 `json:"upload_session_ttl_seconds"`
	CoverBytes              int64 `json:"cover_bytes"`
	SiteAssetBytes          int64 `json:"site_asset_bytes"`
}

type RuntimeLimitBounds struct {
	Min RuntimeLimits `json:"min"`
	Max RuntimeLimits `json:"max"`
}

type AdminSettings struct {
	PublicSettings
	Limits RuntimeLimits      `json:"limits"`
	Bounds RuntimeLimitBounds `json:"bounds"`
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
