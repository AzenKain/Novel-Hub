package aac

var specBookMax = [11]int{1, 1, 2, 2, 4, 4, 7, 7, 12, 12, 16}

const escMaxValue = 8191

func specTupleBits(cb int, v []int) int {
	dim, mod, off := hcbDim[cb-1], hcbMod[cb-1], hcbOff[cb-1]
	idx := 0
	extra := 0
	if hcbUnsigned[cb-1] {
		for d := 0; d < dim; d++ {
			m := v[d]
			if m < 0 {
				m = -m
			}
			if m != 0 {
				extra++
			}
			if cb == escHCB {
				if m > escMaxValue {
					return -1
				}
				if m >= 16 {
					extra += escBits(m)
					m = 16
				}
			} else if m > specBookMax[cb-1] {
				return -1
			}
			idx = idx*mod + m
		}
	} else {
		for d := 0; d < dim; d++ {
			s := v[d] + off
			if s < 0 || s >= mod {
				return -1
			}
			idx = idx*mod + s
		}
	}
	return int(spectralBits[cb-1][idx]) + extra
}

func escBits(m int) int {
	n := 0
	for m >= 1<<uint(n+5) {
		n++
	}
	return n + 1 + n + 4
}

func (w *bitWriter) writeSpecTuple(cb int, v []int) {
	dim, mod, off := hcbDim[cb-1], hcbMod[cb-1], hcbOff[cb-1]
	idx := 0
	var mag [4]int
	if hcbUnsigned[cb-1] {
		for d := 0; d < dim; d++ {
			m := v[d]
			if m < 0 {
				m = -m
			}
			mag[d] = m
			c := m
			if cb == escHCB && c >= 16 {
				c = 16
			}
			idx = idx*mod + c
		}
	} else {
		for d := 0; d < dim; d++ {
			idx = idx*mod + v[d] + off
		}
	}
	w.writeBits(uint(spectralBits[cb-1][idx]), uint64(spectralCodes[cb-1][idx]))
	if !hcbUnsigned[cb-1] {
		return
	}
	for d := 0; d < dim; d++ {
		if mag[d] != 0 {
			s := uint64(0)
			if v[d] < 0 {
				s = 1
			}
			w.writeBits(1, s)
		}
	}
	if cb == escHCB {
		for d := 0; d < dim; d++ {
			if mag[d] >= 16 {
				n := 0
				for mag[d] >= 1<<uint(n+5) {
					n++
				}
				w.writeBits(uint(n), 1<<uint(n)-1)
				w.writeBits(1, 0)
				w.writeBits(uint(n+4), uint64(mag[d]-1<<uint(n+4)))
			}
		}
	}
}

func specRunBits(cb int, q []int) int {
	dim := hcbDim[cb-1]
	total := 0
	for i := 0; i+dim <= len(q); i += dim {
		b := specTupleBits(cb, q[i:i+dim])
		if b < 0 {
			return -1
		}
		total += b
	}
	return total
}

func (w *bitWriter) writeSpecRun(cb int, q []int) {
	dim := hcbDim[cb-1]
	for i := 0; i+dim <= len(q); i += dim {
		w.writeSpecTuple(cb, q[i:i+dim])
	}
}

func sfDeltaBits(delta int) int { return int(scalefactorBits[delta+60]) }

func (w *bitWriter) writeSFDelta(delta int) {
	w.writeBits(uint(scalefactorBits[delta+60]), uint64(scalefactorCodes[delta+60]))
}
