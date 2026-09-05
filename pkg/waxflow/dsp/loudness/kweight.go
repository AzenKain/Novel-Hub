package loudness

import "math"

const (
	shelfHz   = 1681.9744509742096
	shelfGain = 3.999843853973347
	shelfQ    = 0.7071752369554193
	vbExp     = 0.4996667741545416

	highpassHz = 38.13547087613982
	highpassQ  = 0.5003270373238773
)

type biquad struct {
	b0, b1, b2, a1, a2 float64
}

type kState struct {
	s1a, s1b, s2a, s2b float64
}

func kWeighting(rate int) (shelf, highpass biquad) {
	fs := float64(rate)

	k := math.Tan(math.Pi * shelfHz / fs)
	vh := math.Pow(10, shelfGain/20)
	vb := math.Pow(vh, vbExp)
	a0 := 1 + k/shelfQ + k*k
	shelf = biquad{
		b0: (vh + vb*k/shelfQ + k*k) / a0,
		b1: 2 * (k*k - vh) / a0,
		b2: (vh - vb*k/shelfQ + k*k) / a0,
		a1: 2 * (k*k - 1) / a0,
		a2: (1 - k/shelfQ + k*k) / a0,
	}

	k = math.Tan(math.Pi * highpassHz / fs)
	a0 = 1 + k/highpassQ + k*k
	highpass = biquad{
		b0: 1, b1: -2, b2: 1,
		a1: 2 * (k*k - 1) / a0,
		a2: (1 - k/highpassQ + k*k) / a0,
	}
	return shelf, highpass
}
