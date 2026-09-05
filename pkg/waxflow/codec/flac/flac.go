// Package flac implements a FLAC decoder (RFC 9639), written from the specification.
package flac

import (
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

// Version is the decoder's cache-key version constant (ADR-0004): bump on any change that alters decoded samples.
const Version = "flac-dec-1"

// StreamInfoLen is the byte length of a STREAMINFO metadata block body, which is also the codec configuration blob (codec.Encoder.CodecConfig shape) carried in container.Track.CodecConfig.
const StreamInfoLen = 34

// MaxBlockSize is the largest legal FLAC block size in samples.
const MaxBlockSize = 65535

// MaxRate is the largest sample rate STREAMINFO's 20-bit field can carry.
const MaxRate = 1<<20 - 1

// StreamInfo is a parsed STREAMINFO metadata block.
type StreamInfo struct {
	MinBlock, MaxBlock int
	MinFrame, MaxFrame int
	Rate               int
	Channels           int
	Bits               int
	Samples            int64
	MD5                [16]byte
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "flac: "+fmt.Sprintf(format, args...))
}

// ParseStreamInfo parses a 34-byte STREAMINFO block body.
func ParseStreamInfo(b []byte) (StreamInfo, error) {
	var si StreamInfo
	if len(b) != StreamInfoLen {
		return si, malformed("STREAMINFO of %d bytes, want %d", len(b), StreamInfoLen)
	}
	si.MinBlock = int(b[0])<<8 | int(b[1])
	si.MaxBlock = int(b[2])<<8 | int(b[3])
	si.MinFrame = int(b[4])<<16 | int(b[5])<<8 | int(b[6])
	si.MaxFrame = int(b[7])<<16 | int(b[8])<<8 | int(b[9])
	si.Rate = int(b[10])<<12 | int(b[11])<<4 | int(b[12])>>4
	si.Channels = (int(b[12])>>1)&0x7 + 1
	si.Bits = (int(b[12])&0x1)<<4 | int(b[13])>>4
	si.Bits++
	si.Samples = int64(b[13]&0xF)<<32 | int64(b[14])<<24 | int64(b[15])<<16 | int64(b[16])<<8 | int64(b[17])
	copy(si.MD5[:], b[18:])
	switch {
	case si.Rate == 0:
		return si, malformed("STREAMINFO sample rate 0")
	case si.Bits < 4:
		return si, malformed("STREAMINFO bit depth %d, want at least 4", si.Bits)
	case si.MaxBlock < 16:
		return si, malformed("STREAMINFO max block size %d, want at least 16", si.MaxBlock)
	}
	return si, nil
}

// MarshalBinary packs si into the 34-byte STREAMINFO wire form, the inverse of ParseStreamInfo.
func (si StreamInfo) MarshalBinary() ([]byte, error) {
	switch {
	case si.Rate < 1 || si.Rate > MaxRate:
		return nil, malformed("STREAMINFO rate %d outside 1..%d", si.Rate, MaxRate)
	case si.Channels < 1 || si.Channels > 8:
		return nil, malformed("STREAMINFO channels %d outside 1..8", si.Channels)
	case si.Bits < 4 || si.Bits > 32:
		return nil, malformed("STREAMINFO bit depth %d outside 4..32", si.Bits)
	case si.MinBlock < 16 || si.MaxBlock > MaxBlockSize || si.MinBlock > si.MaxBlock:
		return nil, malformed("STREAMINFO block bounds %d..%d invalid", si.MinBlock, si.MaxBlock)
	case si.MinFrame < 0 || si.MinFrame >= 1<<24 || si.MaxFrame < 0 || si.MaxFrame >= 1<<24:
		return nil, malformed("STREAMINFO frame bounds %d..%d overflow 24 bits", si.MinFrame, si.MaxFrame)
	case si.Samples < 0 || si.Samples >= 1<<36:
		return nil, malformed("STREAMINFO sample count %d overflows 36 bits", si.Samples)
	}
	b := make([]byte, StreamInfoLen)
	b[0], b[1] = byte(si.MinBlock>>8), byte(si.MinBlock)
	b[2], b[3] = byte(si.MaxBlock>>8), byte(si.MaxBlock)
	b[4], b[5], b[6] = byte(si.MinFrame>>16), byte(si.MinFrame>>8), byte(si.MinFrame)
	b[7], b[8], b[9] = byte(si.MaxFrame>>16), byte(si.MaxFrame>>8), byte(si.MaxFrame)
	b[10] = byte(si.Rate >> 12)
	b[11] = byte(si.Rate >> 4)
	b[12] = byte(si.Rate&0xF)<<4 | byte(si.Channels-1)<<1 | byte((si.Bits-1)>>4)
	b[13] = byte((si.Bits-1)&0xF)<<4 | byte(si.Samples>>32)
	b[14], b[15], b[16], b[17] = byte(si.Samples>>24), byte(si.Samples>>16), byte(si.Samples>>8), byte(si.Samples)
	copy(b[18:], si.MD5[:])
	return b, nil
}

// PCMFormat is the pipeline format the decoder emits for this stream.
func (si StreamInfo) PCMFormat() audio.Format {
	return audio.Format{
		Rate:     si.Rate,
		Channels: si.Channels,
		Layout:   audio.DefaultLayout(si.Channels),
		Type:     audio.Int,
		BitDepth: si.Bits,
	}
}
