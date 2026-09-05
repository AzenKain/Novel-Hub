package aac

import "math"

type imdctPlan struct {
	n          int
	aRe, aIm   []float64
	bpRe, bpIm []float64
	twRe, twIm []float64
}

var (
	planLong  = newIMDCTPlan(2048)
	planShort = newIMDCTPlan(256)
)

func newIMDCTPlan(n int) *imdctPlan {
	p := &imdctPlan{n: n}
	n0 := (float64(n)/2 + 1) / 2
	p.aRe = make([]float64, n/2)
	p.aIm = make([]float64, n/2)
	for k := 0; k < n/2; k++ {
		a := 2 * math.Pi * n0 * float64(k) / float64(n)
		p.aRe[k], p.aIm[k] = math.Cos(a), math.Sin(a)
	}
	scale := 2.0 / float64(n)
	bAng := math.Pi * n0 / float64(n)
	bRe, bIm := math.Cos(bAng), math.Sin(bAng)
	p.bpRe = make([]float64, n)
	p.bpIm = make([]float64, n)
	for m := 0; m < n; m++ {
		a := math.Pi * float64(m) / float64(n)
		pRe, pIm := math.Cos(a), math.Sin(a)
		p.bpRe[m] = scale * (bRe*pRe - bIm*pIm)
		p.bpIm[m] = scale * (bRe*pIm + bIm*pRe)
	}
	for length := 2; length <= n; length <<= 1 {
		for k := 0; k < length/2; k++ {
			a := 2 * math.Pi * float64(k) / float64(length)
			p.twRe = append(p.twRe, math.Cos(a))
			p.twIm = append(p.twIm, math.Sin(a))
		}
	}
	return p
}

func (p *imdctPlan) imdct(spec, out []float64) {
	n := p.n
	var reBuf, imBuf [2048]float64
	cr, ci := reBuf[:n], imBuf[:n]
	for k := 0; k < n/2; k++ {
		cr[k] = spec[k] * p.aRe[k]
		ci[k] = spec[k] * p.aIm[k]
	}
	p.fft(cr, ci)
	for m := 0; m < n; m++ {
		out[m] = p.bpRe[m]*cr[m] - p.bpIm[m]*ci[m]
	}
}

func (p *imdctPlan) fft(re, im []float64) {
	n := len(re)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	tw := 0
	for length := 2; length <= n; length <<= 1 {
		half := length / 2
		base := tw
		for i := 0; i < n; i += length {
			for k := 0; k < half; k++ {
				wr, wi := p.twRe[base+k], p.twIm[base+k]
				a, b := i+k, i+k+half
				vr := re[b]*wr - im[b]*wi
				vi := re[b]*wi + im[b]*wr
				re[b], im[b] = re[a]-vr, im[a]-vi
				re[a], im[a] = re[a]+vr, im[a]+vi
			}
		}
		tw += half
	}
}
