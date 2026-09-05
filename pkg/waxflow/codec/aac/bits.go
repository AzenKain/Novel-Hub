package aac

import "encoding/binary"

type bitReader struct {
	data []byte
	pos  int
	n    int
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data, n: len(data) * 8}
}

func (r *bitReader) byteAt(i int) uint32 {
	if i >= 0 && i < len(r.data) {
		return uint32(r.data[i])
	}
	return 0
}

func (r *bitReader) read(n uint) uint32 {
	if n == 0 {
		return 0
	}
	byteOff := r.pos >> 3
	bitOff := uint(r.pos & 7)
	var acc uint64
	if byteOff+8 <= len(r.data) {
		acc = binary.BigEndian.Uint64(r.data[byteOff:])
	} else {
		for i := 0; i < 8; i++ {
			acc = acc<<8 | uint64(r.byteAt(byteOff+i))
		}
	}
	r.pos += int(n)
	return uint32((acc >> (64 - bitOff - n)) & (1<<n - 1))
}

func (r *bitReader) bit() uint32 {
	b := r.byteAt(r.pos>>3) >> (7 - uint(r.pos&7)) & 1
	r.pos++
	return b
}

func (r *bitReader) skip(n int) { r.pos += n }

func (r *bitReader) byteAlign() {
	if rem := r.pos & 7; rem != 0 {
		r.pos += 8 - rem
	}
}

func (r *bitReader) left() int { return r.n - r.pos }

func (r *bitReader) overrun() bool { return r.pos > r.n }
