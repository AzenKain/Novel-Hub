// Package mp3 implements an MPEG-1/2/2.5 Layer III audio decoder (ISO/IEC 11172-3 and 13818-3) in pure Go.
package mp3

import (
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

// Version is the decoder's cache-key version constant (ADR-0004): bump on any change that alters decoded samples.
const Version = "mp3-dec-1"

// HeaderLen is the fixed frame header size in bytes.
const HeaderLen = 4

// MPEGVersion distinguishes the three header generations.
type MPEGVersion uint8

const (
	MPEG1  MPEGVersion = 1
	MPEG2  MPEGVersion = 2
	MPEG25 MPEGVersion = 3
)

func (v MPEGVersion) String() string {
	switch v {
	case MPEG1:
		return "MPEG-1"
	case MPEG2:
		return "MPEG-2"
	case MPEG25:
		return "MPEG-2.5"
	default:
		return fmt.Sprintf("MPEGVersion(%d)", uint8(v))
	}
}

// ChannelMode is the header's 2-bit channel mode field; ModeExt qualifies joint stereo.
type ChannelMode uint8

const (
	ModeStereo ChannelMode = 0
	ModeJoint  ChannelMode = 1
	ModeDual   ChannelMode = 2
	ModeMono   ChannelMode = 3
)

// String names the mode for diagnostics.
func (m ChannelMode) String() string {
	switch m {
	case ModeStereo:
		return "stereo"
	case ModeJoint:
		return "joint-stereo"
	case ModeDual:
		return "dual-channel"
	case ModeMono:
		return "mono"
	default:
		return fmt.Sprintf("ChannelMode(%d)", uint8(m))
	}
}

// Header is a parsed MPEG audio frame header.
type Header struct {
	rateIdx int

	Version   MPEGVersion
	Rate      int
	Channels  int
	Mode      ChannelMode
	ModeExt   int
	Bitrate   int
	Padding   bool
	Protected bool
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "mp3: "+fmt.Sprintf(format, args...))
}

var bitrateKbps = [2][16]int{
	{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, -1},
	{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, -1},
}

var rateHz = [4]int{44100, 48000, 32000, -1}

// ParseHeader parses a Layer III frame header from the first HeaderLen bytes of b.
func ParseHeader(b []byte) (Header, error) {
	if len(b) < HeaderLen {
		return Header{}, malformed("truncated frame header")
	}
	if b[0] != 0xFF || b[1]&0xE0 != 0xE0 {
		return Header{}, malformed("missing frame sync")
	}
	var h Header
	switch b[1] >> 3 & 3 {
	case 0:
		h.Version = MPEG25
	case 2:
		h.Version = MPEG2
	case 3:
		h.Version = MPEG1
	default:
		return Header{}, malformed("reserved MPEG version")
	}
	if layer := b[1] >> 1 & 3; layer != 1 {
		return Header{}, malformed("not Layer III (layer bits %d)", layer)
	}
	h.Protected = b[1]&1 == 0

	lsf := 0
	if h.Version != MPEG1 {
		lsf = 1
	}
	bi := int(b[2] >> 4)
	if kbps := bitrateKbps[lsf][bi]; kbps < 0 {
		return Header{}, malformed("forbidden bit rate index")
	} else {
		h.Bitrate = kbps * 1000
	}
	ri := int(b[2] >> 2 & 3)
	if rateHz[ri] < 0 {
		return Header{}, malformed("reserved sample rate")
	}
	h.rateIdx = ri
	h.Rate = rateHz[ri]
	if h.Version != MPEG1 {
		h.Rate >>= 1
	}
	if h.Version == MPEG25 {
		h.Rate >>= 1
	}
	h.Padding = b[2]&2 != 0

	h.Mode = ChannelMode(b[3] >> 6)
	h.ModeExt = int(b[3] >> 4 & 3)
	h.Channels = 2
	if h.Mode == ModeMono {
		h.Channels = 1
	}
	if b[3]&3 == 2 {
		return Header{}, malformed("reserved emphasis")
	}
	return h, nil
}

// SamplesPerFrame is the PCM frame count one MP3 frame decodes to: 1152 for MPEG-1, 576 for the single-granule MPEG-2/2.5 layout.
func (h Header) SamplesPerFrame() int {
	if h.Version == MPEG1 {
		return 1152
	}
	return 576
}

// Size is the whole frame length in bytes, header included, or 0 for the free format (where the size is a property of the stream: the distance to the next sync).
func (h Header) Size() int {
	if h.Bitrate == 0 {
		return 0
	}
	n := h.SamplesPerFrame() / 8 * h.Bitrate / h.Rate
	if h.Padding {
		n++
	}
	return n
}

// SideInfoLen is the side information length in bytes for this header, following the header and the optional CRC-16.
func (h Header) SideInfoLen() int {
	if h.Version == MPEG1 {
		if h.Channels == 1 {
			return 17
		}
		return 32
	}
	if h.Channels == 1 {
		return 9
	}
	return 17
}

// PCMFormat is the pipeline format frames with this header decode to.
func (h Header) PCMFormat() audio.Format {
	return audio.Format{
		Rate:     h.Rate,
		Channels: h.Channels,
		Layout:   audio.DefaultLayout(h.Channels),
		Type:     audio.Float,
		BitDepth: 32,
	}
}

// Kin reports whether o plausibly belongs to the same stream: equal version, rate, and channel count.
func (h Header) Kin(o Header) bool {
	return h.Version == o.Version && h.Rate == o.Rate && h.Channels == o.Channels
}

func (h Header) rateRow() int {
	base := 0
	switch h.Version {
	case MPEG2:
		base = 3
	case MPEG1:
		base = 6
	}
	row := base + h.rateIdx
	if row != 0 {
		row--
	}
	return row
}
