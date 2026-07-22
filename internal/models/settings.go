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
	Mode       string   `json:"mode"`
	LibraryIDs []string `json:"library_ids"`
}

type PublicSettings struct {
	Site                    SiteSettings         `json:"site"`
	SidebarVisibleItems     []string             `json:"sidebar_visible_items"`
	HomeSections            HomeSectionSettings  `json:"home_sections"`
	RegistrationEnabled     bool                 `json:"registration_enabled"`
	GuestAccess             LibraryPolicy        `json:"guest_access"`
	Download                LibraryPolicy        `json:"download"`
	Bookmark                LibraryPolicy        `json:"bookmark"`
	Collection              LibraryPolicy        `json:"collection"`
	Review                  LibraryPolicy        `json:"review"`
	EnableInBookSearch      bool                 `json:"enable_in_book_search"`
	EnableCustomFontUpload  bool                 `json:"enable_custom_font_upload"`
	SetupCompleted          bool                 `json:"setup_completed"`
	AvailableSidebarItems   []string             `json:"available_sidebar_items"`
	AvailableHomeSections   []string             `json:"available_home_sections"`
	AvailablePolicyModes    []string             `json:"available_policy_modes"`
	AvailableGuestModes     []string             `json:"available_guest_modes"`
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
