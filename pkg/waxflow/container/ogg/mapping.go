package ogg

import (
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
)

type mapping interface {
	codecID() codec.ID

	parseID(pkt []byte) (extraHeaders int, err error)

	parseHeader(pkt []byte) error

	isAudio(pkt []byte) bool

	finalizeTrack(lastGranule func() int64) (container.Track, error)

	packetTiming(pkt []byte, running int64) (pts, dur int64, sync, ok bool)

	selfTiming() bool

	// preroll is how many samples before a seek target the demuxer should land so the decoder reconverges: 0 for FLAC, a block for Vorbis, 80 ms for Opus (RFC 7845).
	preroll() int64

	resetTiming()
}

const detectHeaders = -1
