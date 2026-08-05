package response

type AuthResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TOTPRequired bool   `json:"totp_required,omitempty"`
}

type OTPRequestResponse struct {
	ExpiresInSeconds int `json:"expires_in_seconds"`
	CooldownSeconds  int `json:"cooldown_seconds"`
}

type OTPVerifyResponse struct {
	OTPTicket        string `json:"otp_ticket"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}
