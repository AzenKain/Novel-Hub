package vorbis

import (
	"math"

	"novelhub/pkg/waxflow/dsp/fft"
)

func fwdScale(n int) float64 { return 4.0 / float64(n) }

type mdctForward struct {
	n            int
	plan         *fft.Plan
	preRe, preIm []float32
	cRe, cIm     []float32
	inRe, inIm   []float32
	outRe, outIm []float32
}

func newMDCTForward(n int) *mdctForward {
	m := &mdctForward{
		n:     n,
		plan:  fft.NewPlan(n),
		preRe: make([]float32, n),
		preIm: make([]float32, n),
		cRe:   make([]float32, n/2),
		cIm:   make([]float32, n/2),
		inRe:  make([]float32, n),
		inIm:  make([]float32, n),
		outRe: make([]float32, n),
		outIm: make([]float32, n),
	}
	for i := 0; i < n; i++ {
		a := math.Pi * float64(i) / float64(n)
		m.preRe[i] = float32(math.Cos(a))
		m.preIm[i] = float32(-math.Sin(a))
	}
	n0 := (float64(n)/2 + 1) / 2
	s := fwdScale(n)
	for k := 0; k < n/2; k++ {
		a := math.Pi * n0 / float64(n) * float64(2*k+1)
		m.cRe[k] = float32(s * math.Cos(a))
		m.cIm[k] = float32(s * math.Sin(a))
	}
	return m
}

func fullWindow(n int) []float32 {
	rise := getPlan(n).window
	w := make([]float32, n)
	for i := 0; i < n/2; i++ {
		w[i] = rise[i]
		w[n-1-i] = rise[i]
	}
	return w
}

func (m *mdctForward) forward(windowed []float32, spec []float32) {
	n := m.n
	for i := 0; i < n; i++ {
		x := windowed[i]
		m.inRe[i] = x * m.preRe[i]
		m.inIm[i] = x * m.preIm[i]
	}
	m.plan.Transform(m.outRe, m.outIm, m.inRe, m.inIm)
	for k := 0; k < n/2; k++ {
		spec[k] = m.cRe[k]*m.outRe[k] + m.cIm[k]*m.outIm[k]
	}
}
