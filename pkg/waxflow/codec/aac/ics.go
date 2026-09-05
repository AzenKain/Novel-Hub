package aac

import "math"

type pulseInfo struct {
	numPulse int
	startSfb int
	offset   [4]int
	amp      [4]int
}

func (d *Decoder) parseICSInfo(r *bitReader, info *icsInfo) bool {
	r.bit()
	seq := int(r.read(2))
	shape := int(r.bit())
	*info = icsInfo{windowSequence: seq, windowShape: shape}
	if info.windowSequence == eightShort {
		info.maxSfb = int(r.read(4))
		grouping := int(r.read(7))
		info.numWindows = 8
		info.numWindowGroups = 1
		info.windowGroupLen[0] = 1
		for i := 0; i < 7; i++ {
			if grouping&(1<<uint(6-i)) != 0 {
				info.windowGroupLen[info.numWindowGroups-1]++
			} else {
				info.numWindowGroups++
				info.windowGroupLen[info.numWindowGroups-1] = 1
			}
		}
		info.swb = swbOffsetShort[d.rateIdx]
		info.numSwb = swbCountShort(d.rateIdx)
	} else {
		info.maxSfb = int(r.read(6))
		if r.bit() != 0 {
			return false
		}
		info.numWindows = 1
		info.numWindowGroups = 1
		info.windowGroupLen[0] = 1
		info.swb = swbOffsetLong[d.rateIdx]
		info.numSwb = swbCountLong(d.rateIdx)
	}
	return info.maxSfb <= info.numSwb && info.maxSfb <= maxSFBCount
}

func (d *Decoder) decodeChannelData(r *bitReader, cd *channelData, common bool) error {
	cd.globalGain = int(r.read(8))
	if !common {
		if !d.parseICSInfo(r, &cd.info) {
			return malformed("bad ics_info")
		}
	}
	if !d.sectionData(r, cd) {
		return malformed("bad section data")
	}
	if !d.scaleFactorData(r, cd) {
		return malformed("bad scalefactor data")
	}
	cd.hasPulse = r.bit() != 0
	if cd.hasPulse {
		parsePulse(r, cd)
	}
	cd.hasTNS = r.bit() != 0
	if cd.hasTNS {
		parseTNS(r, cd)
	}
	if r.bit() != 0 {
		return malformed("gain control not allowed in AAC-LC")
	}
	clear(cd.spec[:])
	if !d.spectralData(r, cd) {
		return malformed("bad spectral data")
	}
	return nil
}

func groupStarts(info *icsInfo) [maxWindowGroups]int {
	var gs [maxWindowGroups]int
	acc := 0
	for g := 0; g < info.numWindowGroups; g++ {
		gs[g] = acc
		acc += info.windowGroupLen[g]
	}
	return gs
}

func (d *Decoder) sectionData(r *bitReader, cd *channelData) bool {
	info := &cd.info
	lenBits := uint(5)
	if info.windowSequence == eightShort {
		lenBits = 3
	}
	esc := uint32(1)<<lenBits - 1
	for g := 0; g < info.numWindowGroups; g++ {
		k := 0
		for k < info.maxSfb {
			if r.overrun() {
				return false
			}
			cb := uint8(r.read(4))
			if cb == reservedHCB {
				return false
			}
			length := 0
			for {
				incr := r.read(lenBits)
				length += int(incr)
				if incr != esc {
					break
				}
			}
			if length == 0 {
				return false
			}
			for i := 0; i < length; i++ {
				if k >= info.maxSfb {
					return false
				}
				cd.sfbCb[g][k] = cb
				k++
			}
		}
	}
	return true
}

func (d *Decoder) scaleFactorData(r *bitReader, cd *channelData) bool {
	info := &cd.info
	scale := cd.globalGain
	isPos := 0
	noiseEnergy := cd.globalGain - 90
	firstNoise := true
	for g := 0; g < info.numWindowGroups; g++ {
		for sfb := 0; sfb < info.maxSfb; sfb++ {
			switch cb := cd.sfbCb[g][sfb]; {
			case cb == zeroHCB:
				cd.sf[g][sfb] = 0
			case cb == intensityHCB || cb == intensityHCB2:
				delta, ok := decodeScalefactor(r)
				if !ok {
					return false
				}
				isPos += delta
				cd.sf[g][sfb] = isPos
			case cb == noiseHCB:
				if firstNoise {
					firstNoise = false
					noiseEnergy += int(r.read(9)) - 256
				} else {
					delta, ok := decodeScalefactor(r)
					if !ok {
						return false
					}
					noiseEnergy += delta
				}
				cd.sf[g][sfb] = noiseEnergy
			default:
				delta, ok := decodeScalefactor(r)
				if !ok {
					return false
				}
				scale += delta
				if scale < 0 || scale > 255 {
					return false
				}
				cd.sf[g][sfb] = scale
			}
		}
	}
	return true
}

func parsePulse(r *bitReader, cd *channelData) {
	p := &cd.pulse
	p.numPulse = int(r.read(2)) + 1
	p.startSfb = int(r.read(6))
	for i := 0; i < p.numPulse; i++ {
		p.offset[i] = int(r.read(5))
		p.amp[i] = int(r.read(4))
	}
}

func (d *Decoder) spectralData(r *bitReader, cd *channelData) bool {
	info := &cd.info
	gs := groupStarts(info)
	var tuple [4]int
	var pos [1024]int
	for g := 0; g < info.numWindowGroups; g++ {
		L := info.windowGroupLen[g]
		sfb := 0
		for sfb < info.maxSfb {
			cb := int(cd.sfbCb[g][sfb])
			end := sfb
			for end < info.maxSfb && int(cd.sfbCb[g][end]) == cb {
				end++
			}
			if cb == zeroHCB || cb == noiseHCB || cb == intensityHCB || cb == intensityHCB2 || cb >= 12 {
				sfb = end
				continue
			}
			n := 0
			for s := sfb; s < end; s++ {
				width := int(info.swb[s+1]) - int(info.swb[s])
				for w := 0; w < L; w++ {
					base := (gs[g]+w)*128 + int(info.swb[s])
					for k := 0; k < width; k++ {
						pos[n] = base + k
						n++
					}
				}
			}
			dim := hcbDim[cb-1]
			for read := 0; read < n; read += dim {
				if !decodeSpectral(r, cb, tuple[:dim]) {
					return false
				}
				for t := 0; t < dim && read+t < n; t++ {
					cd.spec[pos[read+t]] = float64(tuple[t])
				}
			}
			sfb = end
		}
	}
	return true
}

func (d *Decoder) dequant(cd *channelData) {
	info := &cd.info
	if cd.hasPulse && info.windowSequence != eightShort {
		applyPulse(cd)
	}
	gs := groupStarts(info)
	for g := 0; g < info.numWindowGroups; g++ {
		for sfb := 0; sfb < info.maxSfb; sfb++ {
			cb := cd.sfbCb[g][sfb]
			if cb == zeroHCB || cb == noiseHCB || cb == intensityHCB || cb == intensityHCB2 || cb >= 12 {
				continue
			}
			gain := math.Exp2(0.25 * float64(cd.sf[g][sfb]-sfOffset))
			start, end := int(info.swb[sfb]), int(info.swb[sfb+1])
			for w := 0; w < info.windowGroupLen[g]; w++ {
				base := (gs[g] + w) * 128
				for k := start; k < end; k++ {
					cd.spec[base+k] = iquant(cd.spec[base+k]) * gain
				}
			}
		}
	}
}

func applyPulse(cd *channelData) {
	p := &cd.pulse
	info := &cd.info
	if p.startSfb >= info.numSwb {
		return
	}
	k := int(info.swb[p.startSfb])
	for i := 0; i < p.numPulse; i++ {
		k += p.offset[i]
		if k >= 1024 {
			return
		}
		if cd.spec[k] > 0 {
			cd.spec[k] += float64(p.amp[i])
		} else {
			cd.spec[k] -= float64(p.amp[i])
		}
	}
}

var iqTable [8192]float64

func init() {
	for i := range iqTable {
		iqTable[i] = float64(i) * math.Cbrt(float64(i))
	}
}

func iquant(q float64) float64 {
	a := q
	if a < 0 {
		a = -a
	}
	var v float64
	if a < float64(len(iqTable)) {
		v = iqTable[int(a)]
	} else {
		v = a * math.Cbrt(a)
	}
	if q < 0 {
		return -v
	}
	return v
}
