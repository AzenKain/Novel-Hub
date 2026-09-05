package aac

func (d *Decoder) finishChannel(cd *channelData, outCh int) {
	if cd.hasTNS {
		applyTNS(cd, d.rateIdx)
	}
	info := &cd.info
	curShape := info.windowShape
	prevShape := d.prevWin[outCh]
	var cur [2048]float64
	if info.windowSequence == eightShort {
		shortFilterbank(cd, prevShape, curShape, &cur)
	} else {
		var z [2048]float64
		planLong.imdct(cd.spec[:1024], z[:])
		longWindowApply(&z, &cur, info.windowSequence, prevShape, curShape)
	}
	out := d.buf.ChanF(outCh)[:1024]
	ov := &d.overlap[outCh]
	const norm = 1.0 / 32768.0
	for i := 0; i < 1024; i++ {
		out[i] = float32((cur[i] + ov[i]) * norm)
		ov[i] = cur[1024+i]
	}
	d.prevWin[outCh] = curShape
}

func longWindowApply(z, cur *[2048]float64, seq, prevShape, curShape int) {
	if seq == longStop {
		wl := &shortWindow[prevShape]
		for n := 0; n < 448; n++ {
			cur[n] = 0
		}
		for n := 0; n < 128; n++ {
			cur[448+n] = z[448+n] * wl[n]
		}
		for n := 576; n < 1024; n++ {
			cur[n] = z[n]
		}
	} else {
		wl := &longWindow[prevShape]
		for n := 0; n < 1024; n++ {
			cur[n] = z[n] * wl[n]
		}
	}
	if seq == longStart {
		for n := 1024; n < 1472; n++ {
			cur[n] = z[n]
		}
		wr := &shortWindow[curShape]
		for n := 0; n < 128; n++ {
			cur[1472+n] = z[1472+n] * wr[128+n]
		}
		for n := 1600; n < 2048; n++ {
			cur[n] = 0
		}
	} else {
		wr := &longWindow[curShape]
		for n := 1024; n < 2048; n++ {
			cur[n] = z[n] * wr[n]
		}
	}
}

func shortFilterbank(cd *channelData, prevShape, curShape int, cur *[2048]float64) {
	*cur = [2048]float64{}
	for i := 0; i < 8; i++ {
		var z [256]float64
		planShort.imdct(cd.spec[i*128:i*128+128], z[:])
		lShape := curShape
		if i == 0 {
			lShape = prevShape
		}
		wl := &shortWindow[lShape]
		wr := &shortWindow[curShape]
		off := 448 + i*128
		for n := 0; n < 128; n++ {
			cur[off+n] += z[n] * wl[n]
			cur[off+128+n] += z[128+n] * wr[128+n]
		}
	}
}
