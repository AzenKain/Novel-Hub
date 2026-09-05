// Package audio defines WaxFlow's PCM model: sample formats, channel layouts, and the planar dual-domain Buffer that every decoder, DSP node, and encoder exchanges.
package audio

import (
	"fmt"

	"novelhub/pkg/waxflow/waxerr"
)

// SampleType selects a Buffer's populated domain.
type SampleType uint8

const (
	Int SampleType = iota
	Float
)

func (t SampleType) String() string {
	switch t {
	case Int:
		return "int"
	case Float:
		return "float"
	default:
		return fmt.Sprintf("SampleType(%d)", uint8(t))
	}
}

// MaxChannels is the widest layout the pipeline decodes (7.1).
const MaxChannels = 8

// Format describes PCM audio in the pipeline domain: what a Buffer holds, not how bytes are packed on the wire (wire packing is the pcm codec's concern).
type Format struct {
	Rate     int
	Channels int
	Layout   ChannelMask
	Type     SampleType
	BitDepth int
}

// Valid reports whether the format is internally consistent.
func (f Format) Valid() error {
	switch {
	case f.Rate <= 0:
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: rate %d must be positive", f.Rate))
	case f.Channels < 1 || f.Channels > MaxChannels:
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: %d channels outside 1..%d", f.Channels, MaxChannels))
	case f.Layout != 0 && f.Layout.Count() != f.Channels:
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: layout %v has %d positions for %d channels", f.Layout, f.Layout.Count(), f.Channels))
	}
	switch f.Type {
	case Int:
		if f.BitDepth < 1 || f.BitDepth > 32 {
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: int bit depth %d outside 1..32", f.BitDepth))
		}
	case Float:
		if f.BitDepth != 32 {
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: float bit depth must be 32, got %d", f.BitDepth))
		}
	default:
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("audio: unknown sample type %d", f.Type))
	}
	return nil
}

func (f Format) String() string {
	return fmt.Sprintf("%dHz %dch %s%d", f.Rate, f.Channels, f.Type, f.BitDepth)
}
