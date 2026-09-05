// Package mp4 demuxes ISO base media files (ISO/IEC 14496-12) and their QuickTime kin: the .m4a/.m4b/.mp4 family carrying AAC-LC or ALAC audio.
package mp4

import (
	"fmt"

	"novelhub/pkg/waxflow/waxerr"
)

const (
	maxDepth         = 32
	maxMoovBytes     = 64 << 20
	maxTracks        = 1 << 10
	maxSamples       = 1 << 26
	maxChapters      = 1 << 16
	maxDescriptorLen = 1 << 16
)

// Match reports whether head is an ISO base media file: a leading box whose type is ftyp.
func Match(head []byte) bool {
	if len(head) < 8 {
		return false
	}
	size := be32(head)
	if size < 8 || size > 1024 {
		return false
	}
	return string(head[4:8]) == "ftyp"
}

// MatchNeed is how many leading bytes Match inspects.
const MatchNeed = 8

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "mp4: "+fmt.Sprintf(format, args...))
}

func be16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func be64(b []byte) uint64 {
	return uint64(be32(b))<<32 | uint64(be32(b[4:]))
}
