// Package adts demuxes the ADTS elementary stream framing for AAC (ISO/IEC 14496-3 1.A).
package adts

import (
	"fmt"

	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/waxerr"
)

const (
	headerLen       = 7
	maxResync       = 1 << 20
	maxID3Tags      = 8
	samplesPerFrame = 1024
)

// MatchNeed is how many leading bytes Match inspects: two full frame headers so a stray syncword in other data does not false-positive.
const MatchNeed = 9

// Match reports whether head begins with a valid ADTS frame whose declared length points at a second valid frame (or the end of the buffer).
func Match(head []byte) bool {
	h, ok := parseHeader(head)
	if !ok {
		return false
	}
	if h.frameLen >= len(head) {
		return true
	}
	nh, ok := parseHeader(head[h.frameLen:])
	return ok && h.kin(nh)
}

type header struct {
	profile  int
	rateIdx  int
	channels int
	frameLen int
	hdrLen   int
	blocks   int
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "adts: "+fmt.Sprintf(format, args...))
}

func parseHeader(b []byte) (header, bool) {
	if len(b) < headerLen {
		return header{}, false
	}
	if b[0] != 0xFF || b[1]&0xF0 != 0xF0 || (b[1]>>1)&0x3 != 0 {
		return header{}, false
	}
	protectionAbsent := b[1] & 1
	var h header
	h.profile = int(b[2]>>6) & 0x3
	h.rateIdx = int(b[2]>>2) & 0xF
	h.channels = int(b[2]&1)<<2 | int(b[3]>>6)
	h.frameLen = int(b[3]&0x3)<<11 | int(b[4])<<3 | int(b[5]>>5)
	h.blocks = int(b[6] & 0x3)
	h.hdrLen = headerLen
	if protectionAbsent == 0 {
		h.hdrLen = 9
	}
	if h.rateIdx >= 13 || h.frameLen < h.hdrLen {
		return header{}, false
	}
	return h, true
}

func (h header) kin(o header) bool {
	return h.rateIdx == o.rateIdx && h.channels == o.channels && h.profile == o.profile
}

func (h header) asc() []byte {
	aot := h.profile + 1
	return []byte{
		byte(aot<<3) | byte(h.rateIdx>>1),
		byte(h.rateIdx&1)<<7 | byte(h.channels)<<3,
	}
}

func (h header) config() (aac.Config, error) {
	cfg, err := aac.ParseASC(h.asc())
	if err != nil {
		return aac.Config{}, err
	}
	if cfg.Channels == 0 {
		return aac.Config{}, malformed("channel configuration 0 (in-band PCE) is unsupported")
	}
	return cfg, nil
}
