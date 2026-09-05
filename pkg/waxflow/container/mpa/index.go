package mpa

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/codec/mp3"
	"novelhub/pkg/waxflow/container"
)

var _ container.Indexer = (*Demuxer)(nil)

const idxMagic = "WXMPAIDX1\x00"

const idxMinFrames = 4096

const idxProbes = 8

// IndexSnapshot implements container.Indexer for the lazy frame index.
func (d *Demuxer) IndexSnapshot() []byte {
	if len(d.idx) < idxMinFrames || !d.grew {
		return nil
	}
	buf := make([]byte, 0, len(idxMagic)+2+10+len(d.idx)*2)
	buf = append(buf, idxMagic...)
	if d.done {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	buf = binary.AppendUvarint(buf, uint64(len(d.idx)))
	prev := int64(0)
	for _, off := range d.idx {
		buf = binary.AppendUvarint(buf, uint64(off-prev))
		prev = off
	}
	return buf
}

// RestoreIndex implements container.Indexer.
func (d *Demuxer) RestoreIndex(blob []byte) bool {
	if len(d.idx) > 1 || d.cur != 0 {
		return false
	}
	if len(blob) < len(idxMagic)+2 || string(blob[:len(idxMagic)]) != idxMagic {
		return false
	}
	rest := blob[len(idxMagic):]
	done := rest[0] == 1
	rest = rest[1:]
	count, n := binary.Uvarint(rest)
	if n <= 0 || count == 0 || count > uint64(d.w.DataEnd()/minFrameLen)+2 {
		return false
	}
	rest = rest[n:]
	idx := make([]int64, 0, min(count, 4096))
	prev := int64(-1)
	pos := int64(0)
	for i := uint64(0); i < count; i++ {
		delta, n := binary.Uvarint(rest)
		if n <= 0 || delta > uint64(d.w.DataEnd()) {
			return false
		}
		rest = rest[n:]
		pos += int64(delta)
		if pos <= prev || pos > d.w.DataEnd()-mp3.HeaderLen {
			return false
		}
		prev = pos
		idx = append(idx, pos)
	}
	if idx[0] != d.firstFrame {
		return false
	}
	for i := 0; i <= idxProbes; i++ {
		probe := idx[i*(len(idx)-1)/idxProbes]
		if _, ok := d.frameAt(probe); !ok {
			return false
		}
	}
	if done {
		last := idx[len(idx)-1]
		if h, ok := d.frameAt(last); ok {
			if next := last + int64(h.Size()); next <= d.w.DataEnd()-mp3.HeaderLen {
				if _, more := d.frameAt(next); more {
					done = false
				}
			}
		}
	}
	d.idx = idx
	d.done = done
	d.grew = false
	return true
}
