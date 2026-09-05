// Package mka demuxes Matroska and WebM (ISO/IEC 14496 EBML) audio: the .mka/.mkv/.webm family carrying Opus, Vorbis, FLAC, AAC-LC, or PCM.
package mka

import (
	"fmt"
	"math"

	"novelhub/pkg/waxflow/waxerr"
)

const (
	maxHeaderElement    = 64 << 20
	maxTracks           = 1 << 10
	maxCodecPrivate     = 1 << 20
	maxLaceFrames       = 256
	maxBlockData        = 1 << 26
	maxTopLevelElements = 1 << 20
	maxClusters         = 1 << 20
	maxFrames           = 1 << 26
	maxCuePoints        = 1 << 16
	maxCuesElement      = 8 << 20
)

const (
	idEBML    = 0x1A45DFA3
	idDocType = 0x4282
	idSegment = 0x18538067

	idSeekHead     = 0x114D9B74
	idSeek         = 0x4DBB
	idSeekID       = 0x53AB
	idSeekPosition = 0x53AC

	idInfo           = 0x1549A966
	idTimestampScale = 0x2AD7B1
	idDuration       = 0x4489

	idTracks       = 0x1654AE6B
	idTrackEntry   = 0xAE
	idTrackNumber  = 0xD7
	idTrackType    = 0x83
	idFlagDefault  = 0x88
	idCodecID      = 0x86
	idCodecPrivate = 0x63A2
	idCodecDelay   = 0x56AA
	idSeekPreRoll  = 0x56BB
	idAudio        = 0xE1
	idSamplingFreq = 0xB5
	idChannels     = 0x9F
	idBitDepth     = 0x6264

	idCues               = 0x1C53BB6B
	idCuePoint           = 0xBB
	idCueTime            = 0xB3
	idCueTrackPositions  = 0xB7
	idCueTrack           = 0xF7
	idCueClusterPosition = 0xF1

	idVoid = 0xEC

	idCluster        = 0x1F43B675
	idTimestamp      = 0xE7
	idSimpleBlock    = 0xA3
	idBlockGroup     = 0xA0
	idBlock          = 0xA1
	idDiscardPadding = 0x75A2
)

const trackTypeAudio = 2

const defaultTimestampScale = 1_000_000

var ebmlMagic = [4]byte{0x1A, 0x45, 0xDF, 0xA3}

// Match reports whether head begins with the EBML signature.
func Match(head []byte) bool {
	return len(head) >= 4 &&
		head[0] == ebmlMagic[0] && head[1] == ebmlMagic[1] &&
		head[2] == ebmlMagic[2] && head[3] == ebmlMagic[3]
}

// MatchNeed is how many leading bytes Match inspects.
const MatchNeed = 4

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "mka: "+fmt.Sprintf(format, args...))
}

func beUint(b []byte) uint64 {
	var v uint64
	for _, x := range b {
		v = v<<8 | uint64(x)
	}
	return v
}

func beInt(b []byte) int64 {
	if len(b) == 0 {
		return 0
	}
	v := int64(0)
	if b[0]&0x80 != 0 {
		v = -1
	}
	for _, x := range b {
		v = v<<8 | int64(x)
	}
	return v
}

func beFloat(b []byte) (v float64, ok bool) {
	switch len(b) {
	case 0:
		return 0, true
	case 4:
		return float64(math.Float32frombits(uint32(beUint(b)))), true
	case 8:
		return math.Float64frombits(beUint(b)), true
	default:
		return 0, false
	}
}

func nsToSamples(ns int64, rate int) int64 {
	if ns <= 0 || rate <= 0 {
		return 0
	}
	sec := ns / 1_000_000_000
	rem := ns % 1_000_000_000
	return sec*int64(rate) + (rem*int64(rate)+500_000_000)/1_000_000_000
}
