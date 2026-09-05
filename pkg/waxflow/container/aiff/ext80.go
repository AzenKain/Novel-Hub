package aiff

import "math"

func fromExt80(b []byte) float64 {
	sign := 1.0
	if b[0]&0x80 != 0 {
		sign = -1
	}
	exp := int(b[0]&0x7F)<<8 | int(b[1])
	mant := be.Uint64(b[2:10])
	if exp == 0 && mant == 0 {
		return 0
	}
	if exp == 0x7FFF {
		if mant<<1 == 0 {
			return sign * math.Inf(1)
		}
		return math.NaN()
	}
	return sign * math.Ldexp(float64(mant), exp-16383-63)
}

func toExt80(v float64) [10]byte {
	var b [10]byte
	if v <= 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return b
	}
	frac, e := math.Frexp(v)
	exp := e + 16382
	if exp <= 0 || exp >= 0x7FFF {
		return b
	}
	mant := uint64(frac * (1 << 63) * 2)
	b[0] = byte(exp >> 8)
	b[1] = byte(exp)
	be.PutUint64(b[2:], mant)
	return b
}
