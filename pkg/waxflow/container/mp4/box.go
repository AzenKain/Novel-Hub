package mp4

import (
	"encoding/binary"
	"math"

	"novelhub/pkg/waxflow/container"
)

type box struct {
	typ    string
	off    int64
	hdrLen int64
	size   int64
	toEnd  bool
}

func (b box) payloadOff() int64 { return b.off + b.hdrLen }

func (b box) payloadLen() int64 { return b.size - b.hdrLen }

func readBox(src container.Source, off, end int64) (box, error) {
	if off+8 > end {
		return box{}, malformed("box header at %d runs past end", off)
	}
	var hdr [16]byte
	if err := container.ReadFull(src, hdr[:8], off); err != nil {
		return box{}, err
	}
	b := box{typ: string(hdr[4:8]), off: off, hdrLen: 8}
	size := int64(be32(hdr[:]))
	switch size {
	case 1:
		if off+16 > end {
			return box{}, malformed("64-bit box size at %d runs past end", off)
		}
		if err := container.ReadFull(src, hdr[8:16], off+8); err != nil {
			return box{}, err
		}
		size = int64(be64(hdr[8:]))
		b.hdrLen = 16
	case 0:
		size = end - off
		b.toEnd = true
	}
	if size < b.hdrLen {
		return box{}, malformed("box %q size %d smaller than its header", b.typ, size)
	}
	if size > end-off {
		return box{}, malformed("box %q at %d (size %d) runs past end %d", b.typ, off, size, end)
	}
	b.size = size
	return b, nil
}

func walkBoxes(body []byte, fn func(typ string, payload []byte) error) error {
	for len(body) >= 8 {
		size := int64(be32(body))
		typ := string(body[4:8])
		hdr := int64(8)
		switch size {
		case 1:
			if len(body) < 16 {
				return malformed("64-bit box %q size truncated", typ)
			}
			size = int64(be64(body[8:16]))
			hdr = 16
		case 0:
			size = int64(len(body))
		}
		if size < hdr {
			return malformed("box %q size %d smaller than its header", typ, size)
		}
		if size > int64(len(body)) {
			return malformed("box %q size %d exceeds %d remaining bytes", typ, size, len(body))
		}
		if err := fn(typ, body[hdr:size]); err != nil {
			return err
		}
		body = body[size:]
	}
	return nil
}

func fullBox(payload []byte) (version byte, flags uint32, rest []byte, ok bool) {
	if len(payload) < 4 {
		return 0, 0, nil, false
	}
	return payload[0], be32(payload[:4]) & 0xFFFFFF, payload[4:], true
}

func makeBox(typ string, parts ...[]byte) []byte {
	n := 8
	for _, p := range parts {
		n += len(p)
	}
	b := make([]byte, 8, n)
	binary.BigEndian.PutUint32(b, uint32(n))
	copy(b[4:], typ)
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

func makeFullBox(typ string, version byte, flags uint32, parts ...[]byte) []byte {
	head := []byte{version, byte(flags >> 16), byte(flags >> 8), byte(flags)}
	return makeBox(typ, append([][]byte{head}, parts...)...)
}

func durVersion(dur int64) byte {
	if dur > math.MaxUint32 {
		return 1
	}
	return 0
}

func zeroTimes(version byte) []byte {
	if version == 1 {
		return make([]byte, 16)
	}
	return make([]byte, 8)
}

func durField(version byte, dur int64) []byte {
	if dur < 0 {
		dur = 0
	}
	if version == 1 {
		return u64(uint64(dur))
	}
	return u32(uint32(dur))
}

func u16(v uint16) []byte { return []byte{byte(v >> 8), byte(v)} }

func u32(v uint32) []byte {
	return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}

func u64(v uint64) []byte {
	return []byte{byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32),
		byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
}
