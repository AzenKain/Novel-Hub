package constants

import "time"

const (
	MaxPaginationLimit = 100

	MaxUploadChunkBytes         = 32 << 20
	MaxUploadChunks             = 256
	MaxUploadSessions           = 16
	MaxUploadBytes              = int64(8) << 30
	MaxCoverBytes               = 32 << 20
	MaxSiteAssetBytes           = 10 << 20
	MaxArchiveEntries           = 20000
	MaxArchiveAssetSize         = 128 << 20
	MaxArchiveUncompressedBytes = int64(2) << 30
	UploadSessionTTL            = 6 * time.Hour

	MinRuntimeUploadChunkBytes = 1 << 20
	MinRuntimeUploadChunks     = 1
	MinRuntimeUploadSessions   = 1
	MinRuntimeUploadBytes      = int64(1) << 20
	MinRuntimeUploadSessionTTL = 5 * time.Minute
	MinRuntimeCoverBytes       = 1 << 20
	MinRuntimeSiteAssetBytes   = 1 << 20

	HardMaxUploadChunkBytes = 64 << 20
	HardMaxUploadChunks     = 1024
	HardMaxUploadSessions   = 64
	HardMaxUploadBytes      = int64(32) << 30
	HardMaxUploadSessionTTL = 24 * time.Hour
	HardMaxCoverBytes       = 64 << 20
	HardMaxSiteAssetBytes   = 16 << 20

	MaxDefaultRequestBody = 16 << 20
	MultipartBodyOverhead = 1 << 20

	MaxRateLimitAuth              = 5
	MaxRateLimitAuthWindowSeconds = 60

	MinRuntimeRateLimitAuth              = 1
	MinRuntimeRateLimitAuthWindowSeconds = 1

	HardMaxRateLimitAuth              = 1000
	HardMaxRateLimitAuthWindowSeconds = 3600
)
