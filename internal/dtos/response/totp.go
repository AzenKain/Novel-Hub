package response

import "time"

type TOTPStatusResponse struct {
	Enabled                bool       `json:"enabled"`
	PendingEnrollment      bool       `json:"pending_enrollment,omitempty"`
	ConfirmedAt            *time.Time `json:"confirmed_at,omitempty"`
	RecoveryCodesRemaining int        `json:"recovery_codes_remaining"`
}

type TOTPEnrollResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioning_uri"`
}

type TOTPRecoveryCodesResponse struct {
	Codes []string `json:"codes"`
}
