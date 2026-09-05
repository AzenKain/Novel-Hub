package mp3

type bitReader struct {
	data []byte
	pos  int
	err  bool
}

func (r *bitReader) bitLen() int { return len(r.data) * 8 }

func (r *bitReader) bit() uint32 {
	if r.pos >= len(r.data)*8 {
		r.err = true
		return 0
	}
	b := r.data[r.pos>>3] >> (7 - r.pos&7) & 1
	r.pos++
	return uint32(b)
}

func (r *bitReader) bits(k uint) uint32 {
	if k == 0 {
		return 0
	}
	if r.pos+int(k) > len(r.data)*8 {
		r.err = true
		r.pos = len(r.data) * 8
		return 0
	}
	i := r.pos >> 3
	off := uint(r.pos & 7)
	var v uint32
	for j := 0; j < 4; j++ {
		v <<= 8
		if i+j < len(r.data) {
			v |= uint32(r.data[i+j])
		}
	}
	r.pos += int(k)
	return v << off >> (32 - k)
}

func (r *bitReader) bitPos() int { return r.pos }

func (r *bitReader) setPos(p int) {
	if p > len(r.data)*8 {
		r.err = true
		p = len(r.data) * 8
	}
	r.pos = p
}
