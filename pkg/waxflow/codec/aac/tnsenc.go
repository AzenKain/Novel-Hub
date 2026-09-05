package aac

import "math"

const (
	tnsMaxOrderLC = 12
	tnsMinGain    = 1.4
	tnsStartHz    = 1500.0
)

type tnsEnc struct {
	present bool
	order   int
	length  int
	coef    [tnsMaxOrderLC]int
}

func analyzeTNS(spec *[1024]float64, swb []uint16, numSwb, maxSfb, rateIdx, rate int) tnsEnc {
	maxBands := tnsMaxBandsLong[rateIdx]
	startSfb := 0
	lineHz := float64(rate) / 2048
	for startSfb < numSwb && float64(swb[startSfb])*lineHz < tnsStartHz {
		startSfb++
	}
	topSfb := min(min(numSwb, maxBands), maxSfb)
	start, end := int(swb[min(startSfb, topSfb)]), int(swb[topSfb])
	if end-start < 32 {
		return tnsEnc{}
	}
	region := spec[start:end]

	var r [tnsMaxOrderLC + 1]float64
	for lag := 0; lag <= tnsMaxOrderLC; lag++ {
		s := 0.0
		for i := lag; i < len(region); i++ {
			s += region[i] * region[i-lag]
		}
		r[lag] = s
	}
	if r[0] <= 0 {
		return tnsEnc{}
	}
	var lpc [tnsMaxOrderLC + 1]float64
	var refl [tnsMaxOrderLC]float64
	lpc[0] = 1
	errE := r[0]
	order := 0
	for m := 1; m <= tnsMaxOrderLC; m++ {
		acc := r[m]
		for i := 1; i < m; i++ {
			acc += lpc[i] * r[m-i]
		}
		k := -acc / errE
		if math.Abs(k) >= 1 {
			break
		}
		var next [tnsMaxOrderLC + 1]float64
		copy(next[:], lpc[:])
		for i := 1; i < m; i++ {
			next[i] = lpc[i] + k*lpc[m-i]
		}
		next[m] = k
		lpc = next
		refl[m-1] = k
		errE *= 1 - k*k
		order = m
	}
	if order == 0 || errE <= 0 || r[0]/errE < tnsMinGain {
		return tnsEnc{}
	}
	for order > 1 && math.Abs(refl[order-1]) < 0.05 {
		order--
	}

	const resBits = 4
	iqfac := (float64(int(1)<<uint(resBits-1)) - 0.5) / (math.Pi / 2)
	iqfacM := (float64(int(1)<<uint(resBits-1)) + 0.5) / (math.Pi / 2)
	t := tnsEnc{present: true, order: order, length: numSwb - startSfb}
	var qrefl [tnsMaxOrderLC]float64
	for i := 0; i < order; i++ {
		fac := iqfac
		if refl[i] < 0 {
			fac = iqfacM
		}
		c := int(math.Round(math.Asin(refl[i]) * fac))
		if c > 7 {
			c = 7
		} else if c < -8 {
			c = -8
		}
		t.coef[i] = c
		if c < 0 {
			qrefl[i] = math.Sin(float64(c) / iqfacM)
		} else {
			qrefl[i] = math.Sin(float64(c) / iqfac)
		}
	}
	var a [tnsMaxOrderLC]float64
	reflToLPC(qrefl[:order], a[:order])

	var hist [tnsMaxOrderLC]float64
	for i := range region {
		x := region[i]
		y := x
		for j := 0; j < order && j < i; j++ {
			y += a[j] * hist[j]
		}
		for j := order - 1; j > 0; j-- {
			hist[j] = hist[j-1]
		}
		hist[0] = x
		region[i] = y
	}
	return t
}

func (t *tnsEnc) sideBits() int {
	if !t.present {
		return 0
	}
	return 2 + 1 + 6 + 5 + 1 + 1 + 4*t.order
}

func (t *tnsEnc) write(w *bitWriter) {
	w.writeBits(2, 1)
	w.writeBits(1, 1)
	w.writeBits(6, uint64(t.length))
	w.writeBits(5, uint64(t.order))
	w.writeBits(1, 0)
	w.writeBits(1, 0)
	for i := 0; i < t.order; i++ {
		w.writeBits(4, uint64(t.coef[i]&0xF))
	}
}
