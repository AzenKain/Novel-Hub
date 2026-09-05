package alac

import (
	"encoding/binary"
	"math/bits"
)

type bitReader struct {
	data      []byte
	pos       int
	validBits int
}

func (r *bitReader) byteAt(i int) uint32 {
	if i < len(r.data) {
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

func (r *bitReader) one() uint32 { return r.read(1) }

func (r *bitReader) advance(n int) { r.pos += n }

func (r *bitReader) byteAlign() {
	if rem := r.pos & 7; rem != 0 {
		r.pos += 8 - rem
	}
}

func (r *bitReader) overrun() bool { return r.pos > r.validBits }

func (r *bitReader) read32(i int) uint32 {
	if i >= 0 && i+4 <= len(r.data) {
		return binary.BigEndian.Uint32(r.data[i:])
	}
	return r.byteAt(i)<<24 | r.byteAt(i+1)<<16 | r.byteAt(i+2)<<8 | r.byteAt(i+3)
}

func (r *bitReader) peek32(p int) uint32 {
	return r.read32(p>>3) << uint(p&7)
}

func (r *bitReader) streamBits(p int, numbits uint) uint32 {
	load1 := r.read32(p >> 3)
	bo := uint(p & 7)
	var result uint32
	if numbits+bo > 32 {
		result = load1 << bo
		load2 := r.byteAt((p >> 3) + 4)
		load2 >>= 8 - (numbits + bo - 32)
		result >>= 32 - numbits
		result |= load2
	} else {
		result = load1 >> (32 - numbits - bo)
	}
	if numbits != 32 {
		result &= ^(^uint32(0) << numbits)
	}
	return result
}

const (
	qbShift           = 9
	qb                = 1 << qbShift
	mmulShift         = 2
	mDenShift         = qbShift - mmulShift - 1
	mOff              = 1 << (mDenShift - 2)
	bitOff            = 24
	maxPrefix16       = 9
	maxPrefix32       = 9
	maxDataTypeBits16 = 16
	nMaxMeanClamp     = 0xFFFF
	nMeanClampVal     = 0xFFFF
)

func lead(x uint32) uint32 { return uint32(bits.LeadingZeros32(x)) }

func lg3a(x uint32) uint32 { return 31 - uint32(bits.LeadingZeros32(x+3)) }

func (r *bitReader) dynGet32(p *int, m, k, maxbits uint32) uint32 {
	stream := r.peek32(*p)
	result := uint32(bits.LeadingZeros32(^stream))
	if result >= maxPrefix32 {
		result = r.streamBits(*p+maxPrefix32, uint(maxbits))
		*p += maxPrefix32 + int(maxbits)
		return result
	}
	*p += int(result) + 1
	if k != 1 {
		stream <<= result + 1
		v := stream >> (32 - k)
		*p += int(k) - 1
		result *= m
		if v >= 2 {
			result += v - 1
			*p++
		}
	}
	return result
}

func (r *bitReader) dynGet16(p *int, m, k uint32) uint32 {
	stream := r.peek32(*p)
	pre := uint32(bits.LeadingZeros32(^stream))
	if pre >= maxPrefix16 {
		pre = maxPrefix16
		*p += int(pre)
		stream <<= pre
		result := stream >> (32 - maxDataTypeBits16)
		*p += maxDataTypeBits16
		return result
	}
	*p += int(pre) + 1
	stream <<= pre + 1
	v := stream >> (32 - k)
	*p += int(k)
	result := pre*m + v - 1
	if v < 2 {
		result -= v - 1
		*p--
	}
	return result
}
