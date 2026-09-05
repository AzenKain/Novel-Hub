package response

import "novelhub/pkg/kobo"

// KoboInitResponse is GET /v1/initialization.
type KoboInitResponse struct {
	Resources map[string]any `json:"Resources"`
}

// KoboUserProfileResponse is GET /v1/user/profile.
type KoboUserProfileResponse struct {
	User KoboProfileUserResponse `json:"User"`
}

type KoboProfileUserResponse struct {
	UserKey string `json:"UserKey"`
	UserID  string `json:"UserId"`
	IsGuest bool   `json:"IsGuest"`
}

// KoboAuthDeviceResponse is POST /v1/auth/device and /v1/auth/refresh.
type KoboAuthDeviceResponse struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
	TokenType    string `json:"TokenType"`
	TrackingID   string `json:"TrackingId"`
	UserKey      string `json:"UserKey"`
}

// KoboEntitlementResponse is one book as the device sees it.
type KoboEntitlementResponse struct {
	BookEntitlement kobo.BookEntitlement `json:"BookEntitlement"`
	BookMetadata    kobo.BookMetadata    `json:"BookMetadata"`
	ReadingState    *kobo.ReadingState   `json:"ReadingState,omitempty"`
}

// KoboSyncItemResponse is one element of the sync array.
type KoboSyncItemResponse struct {
	NewEntitlement     *KoboEntitlementResponse `json:"NewEntitlement,omitempty"`
	ChangedEntitlement *KoboEntitlementResponse `json:"ChangedEntitlement,omitempty"`
	NewTag             *kobo.Tag                `json:"NewTag,omitempty"`
	ChangedTag         *kobo.Tag                `json:"ChangedTag,omitempty"`
	DeletedTag         *kobo.Tag                `json:"DeletedTag,omitempty"`
}

// KoboSyncResponse is the result of one sync page.
type KoboSyncResponse struct {
	Items     []KoboSyncItemResponse
	SyncToken string
	Continue  bool
}

// KoboSetupResponse is the browser-facing setup card payload, returned inside CommonResponse.
type KoboSetupResponse struct {
	EndpointURL    string `json:"endpoint_url"`
	IsLocalAddress bool   `json:"is_local_address"`
}
