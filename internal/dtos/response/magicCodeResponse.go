package response

import "time"

type MagicCodeResponse struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	PollToken  string    `json:"poll_token"`
	DeviceInfo string    `json:"device_info"`
	UserID     *string   `json:"user_id,omitempty"`
	JWTToken   string    `json:"jwt_token,omitempty"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}
