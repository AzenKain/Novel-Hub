package aac

import "math"

const maxTNSOrder = 20

type tnsFilter struct {
	length    int
	order     int
	direction int
	coef      [maxTNSOrder]float64
}

type tnsInfo struct {
	nFilt   [8]int
	coefRes [8]int
	filt    [8][4]tnsFilter
}

func parseTNS(r *bitReader, cd *channelData) {
	info := &cd.info
	t := &cd.tns
	nFiltBits, lengthBits, orderBits := uint(2), uint(6), uint(5)
	if info.windowSequence == eightShort {
		nFiltBits, lengthBits, orderBits = 1, 4, 3
	}
	for w := 0; w < info.numWindows; w++ {
		nfilt := int(r.read(nFiltBits))
		t.nFilt[w] = nfilt
		if nfilt == 0 {
			continue
		}
		coefRes := int(r.bit())
		t.coefRes[w] = coefRes
		for f := 0; f < nfilt && f < 4; f++ {
			filt := &t.filt[w][f]
			filt.length = int(r.read(lengthBits))
			rawOrder := int(r.read(orderBits))
			if rawOrder == 0 {
				filt.order = 0
				continue
			}
			filt.direction = int(r.bit())
			coefCompress := int(r.bit())
			resBits := coefRes + 3
			coefBits := resBits - coefCompress
			signMask := 1 << uint(coefBits-1)
			negMask := ^((1 << uint(coefBits)) - 1)
			iqfac := (float64(int(1)<<uint(resBits-1)) - 0.5) / (math.Pi / 2)
			iqfacM := (float64(int(1)<<uint(resBits-1)) + 0.5) / (math.Pi / 2)
			filt.order = min(rawOrder, maxTNSOrder)
			var refl [maxTNSOrder]float64
			for i := 0; i < rawOrder; i++ {
				c := int(r.read(uint(coefBits)))
				if i >= maxTNSOrder {
					continue
				}
				if c&signMask != 0 {
					c |= negMask
				}
				fac := iqfac
				if c < 0 {
					fac = iqfacM
				}
				refl[i] = math.Sin(float64(c) / fac)
			}
			reflToLPC(refl[:filt.order], filt.coef[:filt.order])
		}
	}
}

func reflToLPC(refl, a []float64) {
	order := len(refl)
	var lpc [maxTNSOrder + 1]float64
	lpc[0] = 1
	for m := 1; m <= order; m++ {
		var b [maxTNSOrder + 1]float64
		for i := 1; i < m; i++ {
			b[i] = lpc[i] + refl[m-1]*lpc[m-i]
		}
		for i := 1; i < m; i++ {
			lpc[i] = b[i]
		}
		lpc[m] = refl[m-1]
	}
	for i := 0; i < order; i++ {
		a[i] = lpc[i+1]
	}
}

func applyTNS(cd *channelData, rateIdx int) {
	info := &cd.info
	t := &cd.tns
	maxBands := tnsMaxBandsLong[rateIdx]
	if info.windowSequence == eightShort {
		maxBands = tnsMaxBandsShort[rateIdx]
	}
	for w := 0; w < info.numWindows; w++ {
		if t.nFilt[w] == 0 {
			continue
		}
		top := info.numSwb
		base := w * 128
		for f := 0; f < t.nFilt[w]; f++ {
			filt := &t.filt[w][f]
			bottom := max(top-filt.length, 0)
			if filt.order == 0 {
				top = bottom
				continue
			}
			start := int(info.swb[min(min(bottom, maxBands), info.maxSfb)])
			end := int(info.swb[min(min(top, maxBands), info.maxSfb)])
			size := end - start
			if size <= 0 {
				top = bottom
				continue
			}
			arFilter(cd.spec[base+start:base+end], size, filt)
			top = bottom
		}
	}
}

func arFilter(spec []float64, size int, filt *tnsFilter) {
	inc := 1
	off := 0
	if filt.direction != 0 {
		inc = -1
		off = size - 1
	}
	var state [maxTNSOrder]float64
	pos := off
	for i := 0; i < size; i++ {
		y := spec[pos]
		for j := 0; j < filt.order; j++ {
			if i-1-j >= 0 {
				y -= filt.coef[j] * state[j]
			}
		}
		for j := filt.order - 1; j > 0; j-- {
			state[j] = state[j-1]
		}
		state[0] = y
		spec[pos] = y
		pos += inc
	}
}
