package opus

import (
	"math"

	"novelhub/pkg/waxflow/dsp/fft"
)

// The CELT inverse MDCT.

type mdctPlan struct {
	n  int
	n2 int
	n4 int
	tr []float64
	fp *fft.Plan
}

type mdctPlanCache struct {
	mplans map[int]*mdctPlan
}

func (c *mdctPlanCache) planFor(n int) *mdctPlan {
	if p, ok := c.mplans[n]; ok {
		return p
	}
	if c.mplans == nil {
		c.mplans = map[int]*mdctPlan{}
	}
	p := newMDCTPlan(n)
	c.mplans[n] = p
	return p
}

func newMDCTPlan(n int) *mdctPlan {
	p := &mdctPlan{n: n, n2: n / 2, n4: n / 4}
	p.tr = make([]float64, p.n2)
	for j := range p.tr {
		p.tr[j] = math.Cos(2 * math.Pi * (float64(j) + 0.125) / float64(n))
	}
	p.fp = fft.NewPlan(p.n4)
	return p
}

func celtWindow(overlap int) []float64 {
	w := make([]float64, overlap)
	for i := range w {
		s := math.Sin(0.5 * math.Pi * (float64(i) + 0.5) / float64(overlap))
		w[i] = math.Sin(0.5 * math.Pi * s * s)
	}
	return w
}

type mdctScratch struct {
	fr, fi, gr, gi []float32
}

func newMDCTScratch(maxN4 int) *mdctScratch {
	return &mdctScratch{
		fr: make([]float32, maxN4),
		fi: make([]float32, maxN4),
		gr: make([]float32, maxN4),
		gi: make([]float32, maxN4),
	}
}

func (p *mdctPlan) backward(in []float32, stride int, out []float32, window []float64, overlap int, s *mdctScratch) {
	n2, n4 := p.n2, p.n4
	tr := p.tr
	fr, fi := s.fr[:n4], s.fi[:n4]
	gr, gi := s.gr[:n4], s.gi[:n4]

	for i := 0; i < n4; i++ {
		x1 := float64(in[(2*i)*stride])
		x2 := float64(in[(n2-1-2*i)*stride])
		yr := x2*tr[i] + x1*tr[n4+i]
		yi := x1*tr[i] - x2*tr[n4+i]
		fr[i] = float32(yi)
		fi[i] = float32(yr)
	}

	p.fp.Transform(gr, gi, fr, fi)

	mid := overlap / 2
	half := (n4 + 1) >> 1
	for i := 0; i < half; i++ {
		re := float64(gi[i])
		im := float64(gr[i])
		t0 := tr[i]
		t1 := tr[n4+i]
		yr0 := re*t0 + im*t1
		yi0 := re*t1 - im*t0
		re2 := float64(gi[n4-1-i])
		im2 := float64(gr[n4-1-i])
		t0b := tr[n4-1-i]
		t1b := tr[n2-1-i]
		yr1 := re2*t0b + im2*t1b
		yi1 := re2*t1b - im2*t0b
		out[mid+2*i] = float32(yr0)
		out[mid+n2-1-2*i] = float32(yi0)
		out[mid+n2-2-2*i] = float32(yr1)
		out[mid+2*i+1] = float32(yi1)
	}

	for i := 0; i < overlap/2; i++ {
		x1 := float64(out[overlap-1-i])
		x2 := float64(out[i])
		w1 := window[i]
		w2 := window[overlap-1-i]
		out[i] = float32(x2*w2 - x1*w1)
		out[overlap-1-i] = float32(x2*w1 + x1*w2)
	}
}

func (p *mdctPlan) forward(in []float32, out []float32, stride int, window []float64, overlap int, s *mdctScratch) {
	n2, n4 := p.n2, p.n4
	tr := p.tr
	fr, fi := s.fr[:n4], s.fi[:n4]
	gr, gi := s.gr[:n4], s.gi[:n4]
	scale := 1.0 / float64(n4)
	o2 := overlap / 2
	half1 := (overlap + 3) >> 2

	for i := 0; i < n4; i++ {
		xp1 := o2 + 2*i
		xp2 := n2 - 1 + o2 - 2*i
		var re, im float64
		switch {
		case i < half1:
			wp1 := o2 + 2*i
			wp2 := o2 - 1 - 2*i
			re = float64(in[xp1+n2])*window[wp2] + float64(in[xp2])*window[wp1]
			im = float64(in[xp1])*window[wp1] - float64(in[xp2-n2])*window[wp2]
		case i < n4-half1:
			re = float64(in[xp2])
			im = float64(in[xp1])
		default:
			k := i - (n4 - half1)
			wp1 := 2 * k
			wp2 := overlap - 1 - 2*k
			re = -float64(in[xp1-n2])*window[wp1] + float64(in[xp2])*window[wp2]
			im = float64(in[xp1])*window[wp2] + float64(in[xp2+n2])*window[wp1]
		}
		t0 := tr[i]
		t1 := tr[n4+i]
		fr[i] = float32((re*t0 - im*t1) * scale)
		fi[i] = float32((im*t0 + re*t1) * scale)
	}

	p.fp.Transform(gr, gi, fr, fi)

	for i := 0; i < n4; i++ {
		t0 := tr[i]
		t1 := tr[n4+i]
		yr := float64(gi[i])*t1 - float64(gr[i])*t0
		yi := float64(gr[i])*t1 + float64(gi[i])*t0
		out[stride*(2*i)] = float32(yr)
		out[stride*(n2-1-2*i)] = float32(yi)
	}
}
