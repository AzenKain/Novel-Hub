package constants

import "time"

const (
	NormalCacheDuration  = 15 * time.Minute
	ListCacheDuration    = 10 * time.Minute
	AccessTokenDuration  = 30 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour

	OTPDuration         = 10 * time.Minute
	OTPCooldown         = 60 * time.Second
	OTPVerifiedDuration = 15 * time.Minute
	OTPMaxAttempts      = 5

	// Covers the three TOTP steps a code is accepted across, so a used code cannot be replayed.
	TOTPReplayWindow = 2 * time.Minute
)
