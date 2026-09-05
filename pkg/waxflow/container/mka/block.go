package mka

import (
	"novelhub/pkg/waxflow/container/internal/srcwin"
	"novelhub/pkg/waxflow/waxerr"
)

type frameLoc struct {
	off  int64
	size int
}

type blockHeader struct {
	track   uint64
	relTime int64
	frames  []frameLoc
}

const (
	laceNone  = 0
	laceXiph  = 1
	laceFixed = 2
	laceEBML  = 3
)

func parseBlock(w *srcwin.Window, dataOff, size int64) (blockHeader, error) {
	var bh blockHeader
	if size < 4 {
		return bh, malformed("block of %d bytes too short", size)
	}
	if size > maxBlockData {
		return bh, malformed("block of %d bytes exceeds the %d cap", size, int64(maxBlockData))
	}
	end := dataOff + size

	head := w.BytesAt(dataOff, int(min(8, size)))
	if len(head) == 0 {
		return bh, readErr(w, "block track number")
	}
	track, tlen, _, ok := parseVint(head, false)
	if !ok {
		return bh, malformed("block track number is not a valid vint")
	}
	pos := dataOff + int64(tlen)
	if pos+3 > end {
		return bh, malformed("block header truncated")
	}
	ht := w.BytesAt(pos, 3)
	if len(ht) < 3 {
		return bh, readErr(w, "block timestamp")
	}
	bh.track = track
	bh.relTime = int64(int16(uint16(ht[0])<<8 | uint16(ht[1])))
	flags := ht[2]
	pos += 3

	lacing := int(flags>>1) & 0x3
	if lacing == laceNone {
		bh.frames = []frameLoc{{off: pos, size: int(end - pos)}}
		return bh, nil
	}

	cb := w.BytesAt(pos, 1)
	if len(cb) < 1 {
		return bh, readErr(w, "block lace count")
	}
	n := int(cb[0]) + 1
	pos++
	if n < 1 || n > maxLaceFrames {
		return bh, malformed("block laces %d frames", n)
	}

	sizes := make([]int, n)
	var err error
	switch lacing {
	case laceFixed:
		total := end - pos
		if total < 0 || total%int64(n) != 0 {
			return bh, malformed("fixed-lace block of %d bytes not divisible by %d frames", total, n)
		}
		each := int(total / int64(n))
		for i := 0; i < n-1; i++ {
			sizes[i] = each
		}
	case laceXiph:
		pos, err = readXiphSizes(w, pos, end, sizes)
	case laceEBML:
		pos, err = readEBMLSizes(w, pos, end, sizes)
	}
	if err != nil {
		return bh, err
	}

	used := int64(0)
	for i := 0; i < n-1; i++ {
		if sizes[i] < 0 {
			return bh, malformed("negative lace size")
		}
		used += int64(sizes[i])
	}
	if pos+used > end {
		return bh, malformed("laced frame sizes overrun the block")
	}
	sizes[n-1] = int(end - pos - used)

	bh.frames = make([]frameLoc, n)
	off := pos
	for i, s := range sizes {
		bh.frames[i] = frameLoc{off: off, size: s}
		off += int64(s)
	}
	return bh, nil
}

func readXiphSizes(w *srcwin.Window, pos, end int64, sizes []int) (int64, error) {
	for i := 0; i < len(sizes)-1; i++ {
		total := 0
		for {
			if pos >= end {
				return 0, malformed("xiph lace size runs past block")
			}
			b := w.BytesAt(pos, 1)
			if len(b) < 1 {
				return 0, readErr(w, "xiph lace size")
			}
			pos++
			total += int(b[0])
			if b[0] < 255 {
				break
			}
		}
		sizes[i] = total
	}
	return pos, nil
}

func readEBMLSizes(w *srcwin.Window, pos, end int64, sizes []int) (int64, error) {
	prev := int64(0)
	for i := 0; i < len(sizes)-1; i++ {
		avail := int(min(8, end-pos))
		b := w.BytesAt(pos, avail)
		if len(b) == 0 {
			return 0, readErr(w, "ebml lace size")
		}
		val, vlen, _, ok := parseVint(b, false)
		if !ok {
			return 0, malformed("ebml lace size is not a valid vint")
		}
		pos += int64(vlen)
		if i == 0 {
			prev = int64(val)
		} else {
			bias := int64(1)<<(7*vlen-1) - 1
			prev += int64(val) - bias
		}
		if prev < 0 {
			return 0, malformed("ebml lace produced a negative frame size")
		}
		sizes[i] = int(prev)
	}
	return pos, nil
}

func readErr(w *srcwin.Window, what string) error {
	if err := w.Err(); err != nil {
		return err
	}
	return waxerr.New(waxerr.CodeUnsupportedFormat, "mka: "+what+" runs past end of data")
}
