package loudness

import (
	"math"

	"novelhub/pkg/waxflow/dsp/internal/firwin"
)

const tpTaps = 12

type truePeak struct {
	phases [][tpTaps]float64
	hist   [][]float64
	pos    []int
}

func newTruePeak(rate, channels int) *truePeak {
	factor := 4
	switch {
	case rate > 192000:
		return nil
	case rate >= 96000:
		factor = 2
	}
	tp := &truePeak{
		phases: make([][tpTaps]float64, factor-1),
		hist:   make([][]float64, channels),
		pos:    make([]int, channels),
	}
	for c := range tp.hist {
		tp.hist[c] = make([]float64, 2*tpTaps)
	}
	const beta = 6.0
	const half = tpTaps / 2
	i0 := firwin.BesselI0(beta)
	for p := range tp.phases {
		frac := float64(p+1) / float64(factor)
		ph := &tp.phases[p]
		var sum float64
		for t := 0; t < tpTaps; t++ {
			x := frac - float64(t-half+1)
			u := x / half
			w := firwin.BesselI0(beta*math.Sqrt(1-u*u)) / i0
			ph[t] = firwin.Sinc(x) * w
			sum += ph[t]
		}
		for t := range ph {
			ph[t] /= sum
		}
	}
	return tp
}

func (tp *truePeak) push(c int, x float64) float64 {
	buf := tp.hist[c]
	p := tp.pos[c]
	buf[p] = x
	buf[p+tpTaps] = x
	if p++; p == tpTaps {
		p = 0
	}
	tp.pos[c] = p
	win := buf[p : p+tpTaps]
	peak := math.Abs(x)
	for p := range tp.phases {
		ph := &tp.phases[p]
		var acc float64
		for t := 0; t < tpTaps; t++ {
			acc += ph[t] * win[t]
		}
		if a := math.Abs(acc); a > peak {
			peak = a
		}
	}
	return peak
}

func (tp *truePeak) drain() float64 {
	var peak float64
	for c := range tp.hist {
		for i := 0; i < tpTaps; i++ {
			if p := tp.push(c, 0); p > peak {
				peak = p
			}
		}
	}
	return peak
}
