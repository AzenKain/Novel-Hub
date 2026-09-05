package vorbis

type bitWriter struct {
	buf   []byte
	cache uint64
	n     uint
	total int64
}

func (w *bitWriter) reset() {
	w.buf = w.buf[:0]
	w.cache = 0
	w.n = 0
	w.total = 0
}

func (w *bitWriter) writeBits(k uint, v uint32) {
	if k == 0 {
		return
	}
	w.cache |= uint64(v&mask32(k)) << w.n
	w.n += k
	w.total += int64(k)
	for w.n >= 8 {
		w.buf = append(w.buf, byte(w.cache))
		w.cache >>= 8
		w.n -= 8
	}
}

func (w *bitWriter) writeBit(v uint32) { w.writeBits(1, v) }

func mask32(k uint) uint32 {
	if k >= 32 {
		return 0xffffffff
	}
	return 1<<k - 1
}

func (w *bitWriter) bits() int64 { return w.total }

func (w *bitWriter) bytes() []byte {
	if w.n > 0 {
		w.buf = append(w.buf, byte(w.cache))
		w.cache = 0
		w.n = 0
	}
	return w.buf
}
