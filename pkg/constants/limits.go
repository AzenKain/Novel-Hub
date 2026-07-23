package constants

import "time"

const (
	MaxPaginationLimit  = 100
	MaxUploadChunkBytes = 10 << 20
	MaxUploadChunks     = 100
	MaxUploadSessions   = 8
	MaxUploadBytes      = 1 << 30
	MaxCoverBytes       = 20 << 20
	MaxSiteAssetBytes   = 5 << 20
	MaxArchiveEntries   = 10000
	MaxArchiveAssetSize = 64 << 20
	UploadSessionTTL    = 2 * time.Hour
)
