package request

type SignInDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code" validate:"omitempty,min=6,max=16"`
}

type RegisterDto struct {
	Email     string `json:"email" validate:"required,min=5,max=255,email"`
	Password  string `json:"password" validate:"required"`
	FullName  string `json:"full_name"`
	OTPTicket string `json:"otp_ticket" validate:"omitempty,uuid"`
}

type RequestOTPDto struct {
	Email   string `json:"email" validate:"required,min=5,max=255,email"`
	Purpose string `json:"purpose" validate:"required,oneof=email_verify password_reset"`
}

type VerifyOTPDto struct {
	Email   string `json:"email" validate:"required,min=5,max=255,email"`
	Purpose string `json:"purpose" validate:"required,oneof=email_verify password_reset"`
	Code    string `json:"code" validate:"required,len=6,numeric"`
}

type ResetPasswordWithOTPDto struct {
	Email       string `json:"email" validate:"required,min=5,max=255,email"`
	OTPTicket   string `json:"otp_ticket" validate:"required,uuid"`
	NewPassword string `json:"new_password" validate:"required"`
}

type SetupDto struct {
	Username            string   `json:"username" validate:"required,min=1,max=255"`
	Email               string   `json:"email" validate:"required,min=5,max=255,email"`
	Password            string   `json:"password" validate:"required"`
	SiteTitle           string   `json:"site_title"`
	SiteDescription     string   `json:"site_description"`
	Favicon             string   `json:"favicon"`
	Logo                string   `json:"logo"`
	Registration        bool     `json:"registration"`
	LoginRequired       bool     `json:"login_required"`
	GuestMode           string   `json:"guest_mode"`
	GuestLibraryIDs     []string `json:"guest_library_ids"`
	DownloadMode        string   `json:"download_mode"`
	BookmarkMode        string   `json:"bookmark_mode"`
	CollectionMode      string   `json:"collection_mode"`
	ReviewMode          string   `json:"review_mode"`
	ShareMode           string   `json:"share_mode"`
	ShareLibraryIDs     []string `json:"share_library_ids"`
	ReadMode            string   `json:"read_mode"`
	ReadLibraryIDs      []string `json:"read_library_ids"`
	SidebarVisibleItems []string `json:"sidebar_visible_items"`
}
