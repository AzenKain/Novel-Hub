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

type RequestMagicCodeResponse struct {
	Code             string `json:"code"`
	PollToken        string `json:"poll_token"`
	ActivateURL      string `json:"activate_url"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type PollMagicCodeResponse struct {
	Status       string        `json:"status"`
	AuthResponse *AuthResponse `json:"auth,omitempty"`
}

type OAuthCallbackResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RedirectURL  string `json:"redirect_url"`
}
