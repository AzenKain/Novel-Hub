package mka

import (
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

func vintWidth(first byte) int {
	for i := 0; i < 8; i++ {
		if first&(0x80>>i) != 0 {
			return i + 1
		}
	}
	return 0
}

func parseVint(b []byte, keepMarker bool) (val uint64, n int, unknown bool, ok bool) {
	if len(b) == 0 {
		return 0, 0, false, false
	}
	w := vintWidth(b[0])
	if w == 0 || w > len(b) {
		return 0, 0, false, false
	}
	if keepMarker {
		return beUint(b[:w]), w, false, true
	}
	first := b[0] & (0xFF >> w)
	allOnes := first == (0xFF >> w)
	val = uint64(first)
	for _, x := range b[1:w] {
		val = val<<8 | uint64(x)
		if x != 0xFF {
			allOnes = false
		}
	}
	return val, w, allOnes, true
}

type element struct {
	id          uint32
	dataOff     int64
	size        int64
	unknownSize bool
}

func (e element) dataEnd() int64 { return e.dataOff + e.size }

func (d *Demuxer) readElement(off, end int64) (element, error) {
	if off+2 > end {
		return element{}, malformed("element header at %d runs past %d", off, end)
	}
	var hdr [12]byte
	n := int64(len(hdr))
	if off+n > end {
		n = end - off
	}
	if err := container.ReadFull(d.src, hdr[:n], off); err != nil {
		return element{}, waxerr.Wrap(waxerr.CodeSourceUnreadable, "mka: reading element header", err)
	}
	buf := hdr[:n]
	id64, idLen, _, ok := parseVint(buf, true)
	if !ok || id64 > 0xFFFFFFFF {
		return element{}, malformed("invalid element ID at %d", off)
	}
	size, sizeLen, unknown, ok := parseVint(buf[idLen:], false)
	if !ok {
		return element{}, malformed("invalid element size at %d", off)
	}
	e := element{id: uint32(id64), dataOff: off + int64(idLen+sizeLen)}
	if unknown {
		e.unknownSize = true
		e.size = -1
		return e, nil
	}
	if e.dataOff > end || int64(size) > end-e.dataOff {
		return element{}, malformed("element %#x at %d (size %d) runs past %d", e.id, off, size, end)
	}
	e.size = int64(size)
	return e, nil
}

func walkElements(buf []byte, fn func(id uint32, data []byte) error) error {
	for len(buf) >= 2 {
		id64, idLen, _, ok := parseVint(buf, true)
		if !ok || id64 > 0xFFFFFFFF {
			return malformed("invalid element ID in master body")
		}
		size, sizeLen, unknown, ok := parseVint(buf[idLen:], false)
		if !ok {
			return malformed("invalid element size in master body")
		}
		hdr := idLen + sizeLen
		body := buf[hdr:]
		n := len(body)
		if !unknown && size < uint64(n) {
			n = int(size)
		}
		if err := fn(uint32(id64), body[:n]); err != nil {
			return err
		}
		buf = body[n:]
	}
	return nil
}
