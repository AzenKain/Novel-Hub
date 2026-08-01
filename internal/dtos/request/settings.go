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
	SidebarVisibleItems     *[]string               `json:"sidebar.visible_items" validate:"omitempty,max=100"`
	HomeSections            *HomeSectionSettingsDto `json:"home.sections"`
	RegistrationEnabled     *bool                   `json:"auth.registration_enabled"`
	GuestAccessMode         *string                 `json:"guest_access.mode" validate:"omitempty,oneof=all selected_libraries login_required"`
	GuestAccessLibraryIDs   *[]string               `json:"guest_access.library_ids" validate:"omitempty,max=100"`
	EnableInBookSearch      *bool                   `json:"reader.enable_in_book_search"`
	EnableCustomFontUpload  *bool                   `json:"font.enable_custom_font_upload"`
	EnableAniListTracking   *bool                   `json:"tracker.anilist_enabled"`
	UploadChunkBytes        *int64                  `json:"limits.upload_chunk_bytes"`
	UploadChunks            *int                    `json:"limits.upload_chunks"`
	UploadSessions          *int                    `json:"limits.upload_sessions"`
	UploadBytes             *int64                  `json:"limits.upload_bytes"`
	UploadSessionTTLSeconds *int64                  `json:"limits.upload_session_ttl_seconds"`
	CoverBytes              *int64                  `json:"limits.cover_bytes"`
	SiteAssetBytes          *int64                  `json:"limits.site_asset_bytes"`
	present                 map[string]bool
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

func (d *UpdateSettingsDto) Values() map[string]any {
	values := make(map[string]any)
	put := func(key string, value any) {
		if value != nil {
			values[key] = value
		}
	}
	put("site.title", d.SiteTitle)
	put("site.description", d.SiteDescription)
	put("site.favicon", d.SiteFavicon)
	put("site.logo", d.SiteLogo)
	put("site.meta_description", d.SiteMetaDescription)
	put("sidebar.visible_items", d.SidebarVisibleItems)
	if d.HomeSections != nil {
		sections := make(map[string]any, 2)
		putSection := func(key string, value *bool) {
			if value != nil {
				sections[key] = *value
			}
		}
		putSection("random_books", d.HomeSections.RandomBooks)
		putSection("top_books", d.HomeSections.TopBooks)
		values["home.sections"] = sections
	}
	put("auth.registration_enabled", d.RegistrationEnabled)
	put("guest_access.mode", d.GuestAccessMode)
	put("guest_access.library_ids", d.GuestAccessLibraryIDs)
	put("reader.enable_in_book_search", d.EnableInBookSearch)
	put("font.enable_custom_font_upload", d.EnableCustomFontUpload)
	put("tracker.anilist_enabled", d.EnableAniListTracking)
	put("limits.upload_chunk_bytes", d.UploadChunkBytes)
	put("limits.upload_chunks", d.UploadChunks)
	put("limits.upload_sessions", d.UploadSessions)
	put("limits.upload_bytes", d.UploadBytes)
	put("limits.upload_session_ttl_seconds", d.UploadSessionTTLSeconds)
	put("limits.cover_bytes", d.CoverBytes)
	put("limits.site_asset_bytes", d.SiteAssetBytes)
	return values
}

func (d *UpdateSettingsDto) UnknownKeys() []string {
	known := map[string]bool{
		"site.title": true, "site.description": true, "site.favicon": true, "site.logo": true,
		"site.meta_description": true, "sidebar.visible_items": true, "home.sections": true,
		"auth.registration_enabled": true, "guest_access.mode": true, "guest_access.library_ids": true,
		"reader.enable_in_book_search": true, "font.enable_custom_font_upload": true,
		"tracker.anilist_enabled":   true,
		"limits.upload_chunk_bytes": true, "limits.upload_chunks": true, "limits.upload_sessions": true,
		"limits.upload_bytes": true, "limits.upload_session_ttl_seconds": true,
		"limits.cover_bytes": true, "limits.site_asset_bytes": true,
	}
	unknown := make([]string, 0)
	for key := range d.present {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}
