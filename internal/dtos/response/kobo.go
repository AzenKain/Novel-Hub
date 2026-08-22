package response

import "novelhub/pkg/kobo"

// Kobo device responses. These are raw Kobo JSON, not CommonResponse envelopes: the device
// parses a fixed schema and ignores anything else, so wrapping would break it. The wire types
// themselves live in pkg/kobo alongside the protocol logic, same as pkg/opds holds Feed/Entry.
//
// The one exception is KoboSetupResponse, which is browser-facing and does go inside the
// normal envelope.

// KoboInitResponse is GET /v1/initialization. The device derives every other URL it calls from
// Resources.
//
// Resources stays a map rather than becoming a struct: it is 147 keys of upstream Kobo store
// endpoints (see pkg/kobo/kobo_resources.json), only four of which we rewrite, and two of which
// are nested objects rather than URLs. A 147-field struct would be a transcription of data that
// is already data.
type KoboInitResponse struct {
	Resources map[string]any `json:"Resources"`
}

// KoboUserProfileResponse is GET /v1/user/profile.
//
// calibre-web does not implement this locally — it forwards to the Kobo store and returns {}
// with proxying off, so there is no known-correct payload to copy. The keys are the ones the
// device sends in its own auth request, which is the only available evidence. Nothing in the
// sync flow reads it.
type KoboUserProfileResponse struct {
	User KoboProfileUserResponse `json:"User"`
}

type KoboProfileUserResponse struct {
	UserKey string `json:"UserKey"`
	UserID  string `json:"UserId"`
	IsGuest bool   `json:"IsGuest"`
}

// KoboAuthDeviceResponse is POST /v1/auth/device and /v1/auth/refresh.
//
// Deliberately a dummy, matching calibre-web's make_calibre_web_auth_response(): the device
// performs a login step before it will sync, but nothing it receives here is verified
// afterwards — the real credential is the token in the URL path.
type KoboAuthDeviceResponse struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
	TokenType    string `json:"TokenType"`
	TrackingID   string `json:"TrackingId"`
	UserKey      string `json:"UserKey"`
}

// KoboEntitlementResponse is one book as the device sees it. ReadingState is attached only when
// there is stored progress newer than the device's cursor; sending an older state would move
// the reader's position backwards.
type KoboEntitlementResponse struct {
	BookEntitlement kobo.BookEntitlement `json:"BookEntitlement"`
	BookMetadata    kobo.BookMetadata    `json:"BookMetadata"`
	ReadingState    *kobo.ReadingState   `json:"ReadingState,omitempty"`
}

// KoboSyncItemResponse is one element of the sync array. Exactly one field is ever set: the
// device distinguishes a book it has never seen from one that changed by which key is present,
// so both are omitempty rather than a single field plus a type discriminator.
type KoboSyncItemResponse struct {
	NewEntitlement     *KoboEntitlementResponse `json:"NewEntitlement,omitempty"`
	ChangedEntitlement *KoboEntitlementResponse `json:"ChangedEntitlement,omitempty"`
	NewTag             *kobo.Tag                `json:"NewTag,omitempty"`
	ChangedTag         *kobo.Tag                `json:"ChangedTag,omitempty"`
	DeletedTag         *kobo.Tag                `json:"DeletedTag,omitempty"`
}

// KoboSyncResponse is the result of one sync page. Items becomes the response body verbatim —
// a bare JSON array — while SyncToken and Continue become headers.
type KoboSyncResponse struct {
	Items     []KoboSyncItemResponse
	SyncToken string
	Continue  bool
}

// KoboSetupResponse is the browser-facing setup card payload, returned inside CommonResponse.
//
// IsLocalAddress reports that the endpoint points at loopback. A Kobo cannot resolve
// "localhost", and pasting such a URL is the most common way this setup silently fails —
// calibre-web warns about the same case. Returned as data rather than an error so the UI can
// show it inline next to the URL it refers to.
type KoboSetupResponse struct {
	EndpointURL    string `json:"endpoint_url"`
	IsLocalAddress bool   `json:"is_local_address"`
}
