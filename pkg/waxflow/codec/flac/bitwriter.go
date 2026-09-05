package flac

type bitWriter struct {
	buf   []byte
	cache uint64
	n     uint
}

func (w *bitWriter) reset() {
	w.buf = w.buf[:0]
	w.cache = 0
	w.n = 0
}

func (w *bitWriter) writeBits(k uint, v uint64) {
	if k == 0 {
		return
	}
	w.cache = w.cache<<k | v&(1<<k-1)
	w.n += k
	for w.n >= 8 {
		w.n -= 8
		w.buf = append(w.buf, byte(w.cache>>w.n))
	}
}

func (w *bitWriter) writeSigned(k uint, v int64) {
	w.writeBits(k, uint64(v))
}

func (w *bitWriter) writeUnary(q uint64) {
	for q >= 32 {
		w.writeBits(32, 0)
		q -= 32
	}
	w.writeBits(uint(q)+1, 1)
}

func (w *bitWriter) align() {
	if w.n > 0 {
		w.writeBits(8-w.n, 0)
	}
}
