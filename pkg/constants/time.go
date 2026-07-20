package constants

import "time"

const (
	NormalCacheDuration  = 15 * time.Minute
	ListCacheDuration    = 10 * time.Minute
	AccessTokenDuration  = 30 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)
