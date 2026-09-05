package vorbis

import "math"

type encBook struct {
	dimensions int
	entries    int

	lengths  []uint8
	codeword []uint32
	codeLen  []uint8

	minimum      float64
	delta        float64
	lookupValues int
}

type bookSpec struct {
	dimensions int
	lengths    []uint8

	lookupType   int
	minimum      float64
	delta        float64
	valueBits    int
	sequenceP    bool
	multiplicand []uint32
}

func buildEncBook(s bookSpec) *encBook {
	entries := len(s.lengths)
	lookup := entries
	if s.lookupType == 1 {
		lookup = lookup1Values(entries, s.dimensions)
	}
	b := &encBook{
		dimensions:   s.dimensions,
		entries:      entries,
		lengths:      s.lengths,
		codeword:     make([]uint32, entries),
		codeLen:      make([]uint8, entries),
		minimum:      s.minimum,
		delta:        s.delta,
		lookupValues: lookup,
	}
	codes, ok := assignCodewords(s.lengths)
	if !ok {
		panic("vorbis: over-subscribed encoder codebook")
	}
	for e := 0; e < entries; e++ {
		l := s.lengths[e]
		b.codeLen[e] = l
		b.codeword[e] = reverseCodeword(codes[e], l)
	}
	return b
}

func reverseCodeword(code uint32, l uint8) uint32 {
	var v uint32
	for b := 0; b < int(l); b++ {
		v |= ((code >> (31 - uint(b))) & 1) << uint(b)
	}
	return v
}

func (b *encBook) emit(w *bitWriter, e int) {
	w.writeBits(uint(b.codeLen[e]), b.codeword[e])
}

func (b *encBook) latIndex(v float64) int {
	idx := int(math.Round((v - b.minimum) / b.delta))
	if idx < 0 {
		idx = 0
	}
	if idx >= b.lookupValues {
		idx = b.lookupValues - 1
	}
	return idx
}

func (b *encBook) latValue(idx int) float64 { return b.minimum + float64(idx)*b.delta }

func (b *encBook) latIndexSigned(v float64) int {
	idx := b.latIndex(v)
	switch {
	case v > 0 && b.latValue(idx) <= 0:
		for idx < b.lookupValues-1 && b.latValue(idx) <= 0 {
			idx++
		}
	case v < 0 && b.latValue(idx) >= 0:
		for idx > 0 && b.latValue(idx) >= 0 {
			idx--
		}
	}
	return idx
}

func (b *encBook) vectorEntry(vals []float64, sign []bool) int {
	entry, pow := 0, 1
	for k := 0; k < b.dimensions; k++ {
		idx := b.latIndex(vals[k])
		if sign != nil && sign[k] {
			idx = b.latIndexSigned(vals[k])
		}
		entry += idx * pow
		pow *= b.lookupValues
	}
	return entry
}

func writeCodebook(w *bitWriter, s bookSpec) {
	entries := len(s.lengths)
	w.writeBits(24, 0x564342)
	w.writeBits(16, uint32(s.dimensions))
	w.writeBits(24, uint32(entries))

	sparse := false
	for _, l := range s.lengths {
		if l == 0 {
			sparse = true
			break
		}
	}
	w.writeBit(0)
	if sparse {
		w.writeBit(1)
		for _, l := range s.lengths {
			if l == 0 {
				w.writeBit(0)
				continue
			}
			w.writeBit(1)
			w.writeBits(5, uint32(l-1))
		}
	} else {
		w.writeBit(0)
		for _, l := range s.lengths {
			w.writeBits(5, uint32(l-1))
		}
	}

	w.writeBits(4, uint32(s.lookupType))
	if s.lookupType == 0 {
		return
	}
	w.writeBits(32, float32Pack(s.minimum))
	w.writeBits(32, float32Pack(s.delta))
	w.writeBits(4, uint32(s.valueBits-1))
	w.writeBit(boolBit(s.sequenceP))
	for _, m := range s.multiplicand {
		w.writeBits(uint(s.valueBits), m)
	}
}

func boolBit(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func float32Pack(v float64) uint32 {
	if v == 0 {
		return 0
	}
	var sign uint32
	if v < 0 {
		sign = 0x80000000
		v = -v
	}
	frac, exp2 := math.Frexp(v)
	m := uint32(math.Round(frac * (1 << 21)))
	e := exp2 + 767
	if m >= 1<<21 {
		m >>= 1
		e++
	}
	if e < 0 {
		return sign
	}
	if e > 0x3ff {
		e = 0x3ff
	}
	return sign | uint32(e)<<21 | (m & 0x1fffff)
}
