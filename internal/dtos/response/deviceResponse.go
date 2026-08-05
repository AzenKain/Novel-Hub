package response

import "time"

type UserDeviceResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	DeviceType    string    `json:"device_type"`
	TargetAddress string    `json:"target_address"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
