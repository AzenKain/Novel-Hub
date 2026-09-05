// Package vorbis decodes Vorbis I audio (the Xiph "Vorbis I specification").
package vorbis

import (
	"errors"
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

var errEndOfPacket = errors.New("vorbis: end of packet")

// Version is the decoder's cache-key version constant (ADR-0004): bump on any change that alters decoded samples.
const Version = "vorbis-dec-1"

const (
	maxChannels    = audio.MaxChannels
	maxCodebooks   = 1 << 16
	maxCodewordLen = 32
	maxFloors      = 1 << 6
	maxResidues    = 1 << 6
	maxMappings    = 1 << 6
	maxModes       = 1 << 6
	maxSubmaps     = 16
	maxFloor1Parts = 31
	maxFloor1Class = 65
	maxFloor1Xs    = 65
	maxBlockSize   = 8192
	minBlockLog    = 6
	maxBlockLog    = 13
)

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "vorbis: "+fmt.Sprintf(format, args...))
}

type bitReader struct {
	data  []byte
	bytex int
	bitx  uint
	eof   bool
}

func newBitReader(data []byte) *bitReader { return &bitReader{data: data} }

func (r *bitReader) bit() uint32 {
	if r.bytex >= len(r.data) {
		r.eof = true
		return 0
	}
	b := (uint32(r.data[r.bytex]) >> r.bitx) & 1
	r.bitx++
	if r.bitx == 8 {
		r.bitx = 0
		r.bytex++
	}
	return b
}

func (r *bitReader) read(n int) uint32 {
	var result uint32
	got := 0
	for got < n {
		if r.bytex >= len(r.data) {
			r.eof = true
			return result
		}
		avail := 8 - int(r.bitx)
		take := n - got
		if take > avail {
			take = avail
		}
		chunk := (uint32(r.data[r.bytex]) >> r.bitx) & ((1 << take) - 1)
		result |= chunk << got
		got += take
		r.bitx += uint(take)
		if r.bitx == 8 {
			r.bitx = 0
			r.bytex++
		}
	}
	return result
}

// Config is the decoder configuration carried out of band by the container (the concatenated identification and setup headers).
type Config struct {
	Version  uint32
	Channels int
	Rate     int
	Bitrate  int

	blockSizes [2]int

	codebooks []codebook
	floors    []floor
	residues  []residue
	mappings  []mapping
	modes     []mode
}

// Format returns the PCM format this stream decodes to.
func (c Config) Format() audio.Format {
	return audio.Format{
		Rate:     c.Rate,
		Channels: c.Channels,
		Layout:   audio.DefaultLayout(c.Channels),
		Type:     audio.Float,
		BitDepth: 32,
	}
}

// LongBlock returns the long transform size, used by the Ogg mapping to size a seek pre-roll.
func (c Config) LongBlock() int { return c.blockSizes[1] }

// ModeBits returns the number of bits a packet's mode number occupies, which the Ogg mapping needs to read a packet's block size without a full decode.
func ModeBits(c Config) int { return ilog(len(c.modes) - 1) }

// PacketBlockSize reads an audio packet's block size (its mode's transform length) without decoding it.
func PacketBlockSize(c Config, modeBits int, pkt []byte) (block int, ok bool) {
	if len(pkt) == 0 || pkt[0]&1 != 0 {
		return 0, false
	}
	r := newBitReader(pkt)
	if r.bit() != 0 {
		return 0, false
	}
	modeNum := int(r.read(modeBits))
	if modeNum >= len(c.modes) {
		return 0, false
	}
	if c.modes[modeNum].blockflag {
		return c.blockSizes[1], true
	}
	return c.blockSizes[0], true
}

func waveFromVorbis(n int) []int {
	switch n {
	case 3:
		return []int{0, 2, 1}
	case 4:
		return []int{0, 1, 2, 3}
	case 5:
		return []int{0, 2, 1, 3, 4}
	case 6:
		return []int{0, 2, 1, 5, 3, 4}
	case 7:
		return []int{0, 2, 1, 6, 5, 3, 4}
	case 8:
		return []int{0, 2, 1, 7, 5, 6, 3, 4}
	default:
		id := make([]int, n)
		for i := range id {
			id[i] = i
		}
		return id
	}
}
