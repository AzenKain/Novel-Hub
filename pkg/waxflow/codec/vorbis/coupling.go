package vorbis

import "math"

func coupleForward(a, b float32) (m, ang float32) {
	if abs32(a) > abs32(b) {
		if a > 0 {
			return a, a - b
		}
		return a, b - a
	}
	if b > 0 {
		return b, a - b
	}
	return b, b - a
}

func abs32(x float32) float32 {
	return math.Float32frombits(math.Float32bits(x) &^ (1 << 31))
}

func coupleResidues(resid [][]float32, n2 int) {
	m, a := resid[0], resid[1]
	for i := 0; i < n2; i++ {
		m[i], a[i] = coupleForward(m[i], a[i])
	}
}

func deriveCoupledClasses(magCls, angCls []int, ang []float32, n2 int) {
	for p := range magCls {
		band := magCls[p]
		if angCls[p] > band {
			band = angCls[p]
		}
		magCls[p] = band
		angCls[p] = classSkip
		if band == classSkip {
			continue
		}
		for l := p * resPartSize; l < (p+1)*resPartSize && l < n2; l++ {
			if ang[l] != 0 {
				angCls[p] = band
				break
			}
		}
	}
}
