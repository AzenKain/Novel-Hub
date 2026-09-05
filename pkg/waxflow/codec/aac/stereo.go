package aac

import "math"

func isIntensity(cb uint8) bool { return cb == intensityHCB || cb == intensityHCB2 }

func (d *Decoder) applyPNS(cd *channelData) {
	info := &cd.info
	gs := groupStarts(info)
	for g := 0; g < info.numWindowGroups; g++ {
		for sfb := 0; sfb < info.maxSfb; sfb++ {
			if cd.sfbCb[g][sfb] != noiseHCB {
				continue
			}
			scale := math.Exp2(0.25 * float64(cd.sf[g][sfb]-sfOffset))
			start, end := int(info.swb[sfb]), int(info.swb[sfb+1])
			for w := 0; w < info.windowGroupLen[g]; w++ {
				base := (gs[g] + w) * 128
				for k := start; k < end; k++ {
					cd.spec[base+k] = d.pnsNext() * scale
				}
			}
		}
	}
}

func (d *Decoder) pnsNext() float64 {
	d.pnsState ^= d.pnsState << 13
	d.pnsState ^= d.pnsState >> 17
	d.pnsState ^= d.pnsState << 5
	return float64(int32(d.pnsState)) / 2147483648.0
}

func applyMS(left, right *channelData, msMask int, msUsed *[maxWindowGroups][maxSFBCount]bool) {
	info := &left.info
	gs := groupStarts(info)
	for g := 0; g < info.numWindowGroups; g++ {
		for sfb := 0; sfb < info.maxSfb; sfb++ {
			on := msMask == 2 || (msMask == 1 && msUsed[g][sfb])
			if !on || left.sfbCb[g][sfb] >= noiseHCB || right.sfbCb[g][sfb] >= noiseHCB {
				continue
			}
			start, end := int(info.swb[sfb]), int(info.swb[sfb+1])
			for w := 0; w < info.windowGroupLen[g]; w++ {
				base := (gs[g] + w) * 128
				for k := start; k < end; k++ {
					m, s := left.spec[base+k], right.spec[base+k]
					left.spec[base+k] = m + s
					right.spec[base+k] = m - s
				}
			}
		}
	}
}

func applyIntensity(left, right *channelData, msMask int, msUsed *[maxWindowGroups][maxSFBCount]bool) {
	info := &right.info
	gs := groupStarts(info)
	for g := 0; g < info.numWindowGroups; g++ {
		for sfb := 0; sfb < info.maxSfb; sfb++ {
			cb := right.sfbCb[g][sfb]
			if !isIntensity(cb) {
				continue
			}
			scale := math.Exp2(-0.25 * float64(right.sf[g][sfb]))
			if cb == intensityHCB2 {
				scale = -scale
			}
			if msMask == 1 && msUsed[g][sfb] {
				scale = -scale
			}
			start, end := int(info.swb[sfb]), int(info.swb[sfb+1])
			for w := 0; w < info.windowGroupLen[g]; w++ {
				base := (gs[g] + w) * 128
				for k := start; k < end; k++ {
					right.spec[base+k] = left.spec[base+k] * scale
				}
			}
		}
	}
}
