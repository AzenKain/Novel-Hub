// Package gain scales PCM level: a plain scalar gain kernel, plus a look-ahead true-peak limiter the chain inserts whenever the level path can clip (net positive gain, or a downmix whose worst-case matrix gain exceeds unity).
package gain

import (
	"fmt"
	"math"
)

// Version is the scalar gain node's revision for cache keys (plan section 10).
const Version = "gain-1"

// FromDB converts decibels to a linear amplitude factor.
func FromDB(db float64) float64 {
	return math.Pow(10, db/20)
}

// Apply scales one channel in place.
func Apply(x []float32, g float32) {
	if g == 1 {
		return
	}
	for i := range x {
		x[i] *= g
	}
}

func checkFrames(what string, slices [][]float32) int {
	n := len(slices[0])
	for _, s := range slices[1:] {
		if len(s) != n {
			panic(fmt.Sprintf("gain: %s channel slices differ in length (%d vs %d); "+
				"every channel must cover the same frames", what, len(s), n))
		}
	}
	return n
}
