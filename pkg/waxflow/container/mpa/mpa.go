// Package mpa demuxes the MP3 elementary stream: a bare sequence of Layer III frames, usually wrapped in ID3 tags, often led by a Xing, Info, or VBRI metadata frame.
package mpa

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/codec/mp3"
)

// Match reports whether head begins with an MP3 elementary stream: a parsable Layer III header confirmed by a second one right behind it (the sync word alone false-positives on arbitrary binaries, which is also why this driver sniffs last).
func Match(head []byte) bool {
	for off := 0; off < maxLeadingJunk && off+mp3.HeaderLen <= len(head); off++ {
		h, err := mp3.ParseHeader(head[off:])
		if err != nil || h.Size() == 0 {
			continue
		}
		next := off + h.Size()
		if next+mp3.HeaderLen > len(head) {
			return off == 0
		}
		if n, err := mp3.ParseHeader(head[next:]); err == nil && h.Kin(n) {
			return true
		}
	}
	return false
}

const maxLeadingJunk = 8 << 10

// MatchNeed is the sniff window Match wants: room for leading junk plus two maximum-size frames.
const MatchNeed = maxLeadingJunk + 2*1441 + mp3.HeaderLen

const decoderDelay = 529

type vbrTag struct {
	frames         int64
	delay, padding int64
}

func parseVBRTag(h mp3.Header, frame []byte) (vbrTag, bool) {
	tag := vbrTag{delay: -1, padding: -1}

	off := mp3.HeaderLen + h.SideInfoLen()
	if h.Protected {
		off += 2
	}
	if len(frame) >= off+8 {
		magic := string(frame[off : off+4])
		if magic == "Xing" || magic == "Info" {
			flags := binary.BigEndian.Uint32(frame[off+4:])
			p := off + 8
			take := func(n int) []byte {
				if p+n > len(frame) {
					p = len(frame)
					return nil
				}
				b := frame[p : p+n]
				p += n
				return b
			}
			if flags&1 != 0 {
				if b := take(4); b != nil {
					tag.frames = int64(binary.BigEndian.Uint32(b))
				}
			}
			if flags&2 != 0 {
				take(4)
			}
			if flags&4 != 0 {
				take(100)
			}
			if flags&8 != 0 {
				take(4)
			}
			if enc := take(9); enc != nil {
				switch string(enc[:4]) {
				case "LAME", "Lavc", "Lavf", "WaxF":
					if p+12+3 <= len(frame) {
						b := frame[p+12:]
						tag.delay = int64(b[0])<<4 | int64(b[1])>>4
						tag.padding = int64(b[1]&0xF)<<8 | int64(b[2])
					}
				}
			}
			return tag, true
		}
	}

	off = mp3.HeaderLen + 32
	if len(frame) >= off+26 && string(frame[off:off+4]) == "VBRI" {
		tag.frames = int64(binary.BigEndian.Uint32(frame[off+14:]))
		return tag, true
	}
	return vbrTag{}, false
}
