package opus

// Laplace-model decoding for CELT coarse band energy (RFC 6716 section 4.3.2.1).

const (
	laplaceLogMinP = 0
	laplaceMinP    = 1 << laplaceLogMinP
	laplaceNMin    = 16
)

func laplaceGetFreq1(fs0 uint32, decay int) uint32 {
	ft := uint32(32768-laplaceMinP*(2*laplaceNMin)) - fs0
	return uint32((uint64(ft) * uint64(16384-decay)) >> 15)
}

func (d *rangeDecoder) laplaceDecode(fs uint32, decay int) int {
	val := 0
	fm := d.decodeBin(15)
	fl := uint32(0)
	if fm >= fs {
		val++
		fl = fs
		fs = laplaceGetFreq1(fs, decay) + laplaceMinP
		for fs > laplaceMinP && fm >= fl+2*fs {
			fs *= 2
			fl += fs
			fs = uint32((uint64(fs-2*laplaceMinP) * uint64(decay)) >> 15)
			fs += laplaceMinP
			val++
		}
		if fs <= laplaceMinP {
			di := int((fm - fl) >> (laplaceLogMinP + 1))
			val += di
			fl += uint32(2*di) * laplaceMinP
		}
		if fm < fl+fs {
			val = -val
		} else {
			fl += fs
		}
	}
	hi := fl + fs
	if hi > 32768 {
		hi = 32768
	}
	d.update(fl, hi, 32768)
	return val
}

func (e *rangeEncoder) laplaceEncode(value int, fs uint32, decay int) int {
	fl := uint32(0)
	val := value
	if val != 0 {
		s := 0
		if val < 0 {
			s = -1
		}
		val = (val + s) ^ s
		fl = fs
		fs = laplaceGetFreq1(fs, decay)
		i := 1
		for ; fs > 0 && i < val; i++ {
			fs *= 2
			fl += fs + 2*laplaceMinP
			fs = uint32((int64(fs) * int64(decay)) >> 15)
		}
		if fs == 0 {
			ndiMax := int(32768-fl+laplaceMinP-1) >> laplaceLogMinP
			ndiMax = (ndiMax - s) >> 1
			di := min(val-i, ndiMax-1)
			fl += uint32(2*di+1+s) * laplaceMinP
			fs = uint32(min(int(laplaceMinP), int(32768-fl)))
			value = (i + di + s) ^ s
		} else {
			fs += laplaceMinP
			fl += fs &^ uint32(s)
		}
	}
	e.encodeBin(fl, fl+fs, 15)
	return value
}
