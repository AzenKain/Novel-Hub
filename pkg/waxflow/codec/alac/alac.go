// Package alac implements an Apple Lossless (ALAC) decoder.
package alac

import (
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

// Version is the decoder's cache-key version constant (ADR-0004): bump on any change that alters decoded samples.
const Version = "alac-dec-1"

// CookieLen is the byte length of an ALACSpecificConfig (the magic cookie), the blob carried in container.Track.CodecConfig.
const CookieLen = 24

const maxFrameLength = 16384

// Config is a parsed ALACSpecificConfig.
type Config struct {
	FrameLength   uint32
	BitDepth      int
	PB            uint32
	MB            uint32
	KB            uint32
	Channels      int
	MaxRun        uint32
	MaxFrameBytes uint32
	AvgBitRate    uint32
	SampleRate    int

	Cookie []byte
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "alac: "+fmt.Sprintf(format, args...))
}

// ParseMagicCookie parses the 24-byte ALACSpecificConfig at the head of b (trailing channel-layout bytes are ignored).
func ParseMagicCookie(b []byte) (Config, error) {
	if len(b) < CookieLen {
		return Config{}, malformed("magic cookie of %d bytes, want at least %d", len(b), CookieLen)
	}
	c := Config{
		FrameLength:   be32(b[0:]),
		BitDepth:      int(b[5]),
		PB:            uint32(b[6]),
		MB:            uint32(b[7]),
		KB:            uint32(b[8]),
		Channels:      int(b[9]),
		MaxRun:        uint32(be16(b[10:])),
		MaxFrameBytes: be32(b[12:]),
		AvgBitRate:    be32(b[16:]),
		SampleRate:    int(be32(b[20:])),
	}
	switch c.BitDepth {
	case 16, 20, 24, 32:
	default:
		return Config{}, malformed("bit depth %d, want 16/20/24/32", c.BitDepth)
	}
	switch {
	case c.FrameLength == 0 || c.FrameLength > maxFrameLength:
		return Config{}, malformed("frame length %d outside 1..%d", c.FrameLength, maxFrameLength)
	case c.Channels < 1 || c.Channels > 2:
		return Config{}, malformed("channel count %d: only mono and stereo are supported", c.Channels)
	case c.SampleRate <= 0:
		return Config{}, malformed("sample rate %d", c.SampleRate)
	}
	c.Cookie = append([]byte(nil), b[:CookieLen]...)
	return c, nil
}

// Format is the pipeline format the decoder emits for this stream: the int domain, right-justified at the stream's bit depth.
func (c Config) Format() audio.Format {
	return audio.Format{
		Rate:     c.SampleRate,
		Channels: c.Channels,
		Layout:   audio.DefaultLayout(c.Channels),
		Type:     audio.Int,
		BitDepth: c.BitDepth,
	}
}

func be16(b []byte) uint16 { return uint16(b[0])<<8 | uint16(b[1]) }

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}
