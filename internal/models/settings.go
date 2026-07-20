package models

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
	SetupCompleted          bool                 `json:"setup_completed"`
	AvailableSidebarItems   []string             `json:"available_sidebar_items"`
	AvailableHomeSections   []string             `json:"available_home_sections"`
	AvailablePolicyModes    []string             `json:"available_policy_modes"`
	AvailableGuestModes     []string             `json:"available_guest_modes"`
}
