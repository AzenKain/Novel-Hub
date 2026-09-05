package flac

import "math/bits"

type bitReader struct {
	data  []byte
	pos   int
	cache uint64
	n     uint
	err   bool
}

func (r *bitReader) ensure(k uint) bool {
	for r.n < k {
		if r.pos >= len(r.data) {
			r.err = true
			return false
		}
		r.cache = r.cache<<8 | uint64(r.data[r.pos])
		r.pos++
		r.n += 8
	}
	return true
}

func (r *bitReader) u(k uint) uint64 {
	if k == 0 {
		return 0
	}
	if !r.ensure(k) {
		return 0
	}
	r.n -= k
	return r.cache >> r.n & (1<<k - 1)
}

func (r *bitReader) s(k uint) int64 {
	v := r.u(k)
	return int64(v<<(64-k)) >> (64 - k)
}

func (r *bitReader) unary() int {
	n := 0
	for {
		if r.n == 0 && !r.ensure(1) {
			return n
		}
		window := r.cache << (64 - r.n)
		lz := uint(bits.LeadingZeros64(window))
		if lz >= r.n {
			n += int(r.n)
			r.n = 0
			continue
		}
		r.n -= lz + 1
		return n + int(lz)
	}
}

func (r *bitReader) align() {
	r.n -= r.n % 8
}

func (r *bitReader) consumed() int {
	return r.pos - int(r.n)/8
}
