package request

type SignInDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required"`
	FullName string `json:"full_name"`
}

type SetupDto struct {
	Username        string                 `json:"username" validate:"required,min=1,max=255"`
	Email           string                 `json:"email" validate:"required,min=5,max=255,email"`
	Password        string                 `json:"password" validate:"required"`
	SiteTitle       string                 `json:"site_title"`
	SiteDescription string                 `json:"site_description"`
	Favicon         string                 `json:"favicon"`
	Logo            string                 `json:"logo"`
	Registration    bool                   `json:"registration"`
	GuestMode       string                 `json:"guest_mode"`
	GuestLibraryIDs []string               `json:"guest_library_ids"`
	DownloadMode    string                 `json:"download_mode"`
	BookmarkMode    string                 `json:"bookmark_mode"`
	CollectionMode  string                 `json:"collection_mode"`
	ReviewMode          string                 `json:"review_mode"`
	SidebarVisibleItems []string               `json:"sidebar_visible_items"`
}
