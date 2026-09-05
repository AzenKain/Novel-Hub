package mka

import (
	"encoding/binary"
	"math"
)

func appendID(dst []byte, id uint32) []byte {
	switch {
	case id <= 0xFF:
		return append(dst, byte(id))
	case id <= 0xFFFF:
		return append(dst, byte(id>>8), byte(id))
	case id <= 0xFFFFFF:
		return append(dst, byte(id>>16), byte(id>>8), byte(id))
	default:
		return append(dst, byte(id>>24), byte(id>>16), byte(id>>8), byte(id))
	}
}

func vintLen(v uint64) int {
	for w := 1; w <= 8; w++ {
		if v < (uint64(1)<<(7*w))-1 {
			return w
		}
	}
	return 8
}

func appendVint(dst []byte, v uint64) []byte {
	w := vintLen(v)
	var b [8]byte
	for i := w - 1; i >= 0; i-- {
		b[i] = byte(v)
		v >>= 8
	}
	b[0] |= 0x80 >> (w - 1)
	return append(dst, b[:w]...)
}

func appendElement(dst []byte, id uint32, body []byte) []byte {
	dst = appendID(dst, id)
	dst = appendVint(dst, uint64(len(body)))
	return append(dst, body...)
}

func appendUint(dst []byte, id uint32, v uint64) []byte {
	return appendElement(dst, id, beUintBytes(v))
}

func appendUintFixed(dst []byte, id uint32, v uint64, width int) []byte {
	if width < 1 || width > 8 {
		panic("mka: appendUintFixed width outside 1..8")
	}
	var full [8]byte
	binary.BigEndian.PutUint64(full[:], v)
	return appendElement(dst, id, full[8-width:])
}

const maxVoidSpan = 128

func appendVoid(dst []byte, total int) []byte {
	if total < 2 || total > maxVoidSpan {
		panic("mka: appendVoid span outside 2..128")
	}
	dst = appendID(dst, idVoid)
	dst = appendVint(dst, uint64(total-2))
	return append(dst, make([]byte, total-2)...)
}

func appendString(dst []byte, id uint32, s string) []byte {
	return appendElement(dst, id, []byte(s))
}

func appendFloat(dst []byte, id uint32, v float64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], math.Float64bits(v))
	return appendElement(dst, id, b[:])
}

func beUintBytes(v uint64) []byte {
	var full [8]byte
	binary.BigEndian.PutUint64(full[:], v)
	i := 0
	for i < 7 && full[i] == 0 {
		i++
	}
	return append([]byte(nil), full[i:]...)
}

func beIntBytes(v int64) []byte {
	var full [8]byte
	binary.BigEndian.PutUint64(full[:], uint64(v))
	i := 0
	if v >= 0 {
		for i < 7 && full[i] == 0x00 && full[i+1]&0x80 == 0 {
			i++
		}
	} else {
		for i < 7 && full[i] == 0xFF && full[i+1]&0x80 != 0 {
			i++
		}
	}
	return append([]byte(nil), full[i:]...)
}

func samplesToNs(samples int64, rate int) int64 {
	if samples <= 0 || rate <= 0 {
		return 0
	}
	r := int64(rate)
	sec := samples / r
	rem := samples % r
	return sec*1_000_000_000 + (rem*1_000_000_000+r/2)/r
}
