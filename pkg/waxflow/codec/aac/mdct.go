package aac

import "math"

type mdctPlan struct {
	inv            *imdctPlan
	preRe, preIm   []float64
	postRe, postIm []float64
}

var (
	mdctLong  = newMDCTPlan(planLong)
	mdctShort = newMDCTPlan(planShort)
)

func newMDCTPlan(inv *imdctPlan) *mdctPlan {
	n := inv.n
	p := &mdctPlan{inv: inv,
		preRe: make([]float64, n), preIm: make([]float64, n),
		postRe: make([]float64, n/2), postIm: make([]float64, n/2)}
	for m := 0; m < n; m++ {
		a := math.Pi * float64(m) / float64(n)
		p.preRe[m], p.preIm[m] = math.Cos(a), math.Sin(a)
	}
	n0 := (float64(n)/2 + 1) / 2
	for k := 0; k < n/2; k++ {
		a := 2 * math.Pi * n0 * (float64(k) + 0.5) / float64(n)
		p.postRe[k], p.postIm[k] = 2*math.Cos(a), -2*math.Sin(a)
	}
	return p
}

func (p *mdctPlan) mdct(x, spec []float64) {
	n := p.inv.n
	if n <= 256 {
		var reBuf, imBuf [256]float64
		p.run(reBuf[:n], imBuf[:n], x, spec)
		return
	}
	var reBuf, imBuf [2048]float64
	p.run(reBuf[:n], imBuf[:n], x, spec)
}

func (p *mdctPlan) run(cr, ci, x, spec []float64) {
	n := p.inv.n
	for m := 0; m < n; m++ {
		cr[m] = x[m] * p.preRe[m]
		ci[m] = x[m] * p.preIm[m]
	}
	p.inv.fft(cr, ci)
	for k := 0; k < n/2; k++ {
		spec[k] = p.postRe[k]*cr[k] + p.postIm[k]*ci[k]
	}
}

func windowedLong(t *[2048]float64, seq int, out *[2048]float64) {
	wl := &longWindow[shapeSine]
	ws := &shortWindow[shapeSine]
	if seq == longStop {
		for n := 0; n < 448; n++ {
			out[n] = 0
		}
		for n := 0; n < 128; n++ {
			out[448+n] = t[448+n] * ws[n]
		}
		for n := 576; n < 1024; n++ {
			out[n] = t[n]
		}
	} else {
		for n := 0; n < 1024; n++ {
			out[n] = t[n] * wl[n]
		}
	}
	if seq == longStart {
		for n := 1024; n < 1472; n++ {
			out[n] = t[n]
		}
		for n := 0; n < 128; n++ {
			out[1472+n] = t[1472+n] * ws[128+n]
		}
		for n := 1600; n < 2048; n++ {
			out[n] = 0
		}
	} else {
		for n := 1024; n < 2048; n++ {
			out[n] = t[n] * wl[n]
		}
	}
}

func mdctFrame(t *[2048]float64, seq int, spec *[1024]float64) {
	if seq != eightShort {
		var w [2048]float64
		windowedLong(t, seq, &w)
		mdctLong.mdct(w[:], spec[:1024])
		return
	}
	ws := &shortWindow[shapeSine]
	var w [256]float64
	for i := 0; i < 8; i++ {
		off := 448 + i*128
		for n := 0; n < 256; n++ {
			w[n] = t[off+n] * ws[n]
		}
		mdctShort.mdct(w[:], spec[i*128:i*128+128])
	}
}
