package vorbis

import "math"

var floor1InverseDB = func() [256]float32 {
	const first = 1.0649863e-07
	var t [256]float32
	ratio := 1.0 / first
	for i := range t {
		t[i] = float32(first * math.Pow(ratio, float64(i)/255))
	}
	t[255] = 1.0
	return t
}()

var floor1Ranges = [4]int{256, 128, 86, 64}
