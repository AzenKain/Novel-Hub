package aac

import "math"

const (
	onlyLong   = 0
	longStart  = 1
	eightShort = 2
	longStop   = 3
)

const (
	shapeSine = 0
	shapeKBD  = 1
)

const (
	zeroHCB       = 0
	escHCB        = 11
	reservedHCB   = 12
	noiseHCB      = 13
	intensityHCB2 = 14
	intensityHCB  = 15
)

var (
	hcbDim      = [11]int{4, 4, 4, 4, 2, 2, 2, 2, 2, 2, 2}
	hcbMod      = [11]int{3, 3, 3, 3, 9, 9, 8, 8, 13, 13, 17}
	hcbOff      = [11]int{1, 1, 0, 0, 4, 4, 0, 0, 0, 0, 0}
	hcbUnsigned = [11]bool{false, false, true, true, false, false, true, true, true, true, true}
)

func samplingIndex(rate int) int {
	for i, r := range sampleRates {
		if r == rate {
			return i
		}
	}
	return -1
}

func swbCountLong(rateIdx int) int  { return len(swbOffsetLong[rateIdx]) - 1 }
func swbCountShort(rateIdx int) int { return len(swbOffsetShort[rateIdx]) - 1 }

var (
	tnsMaxBandsLong  = [13]int{31, 31, 34, 40, 42, 51, 46, 46, 42, 42, 42, 39, 39}
	tnsMaxBandsShort = [13]int{9, 9, 10, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14}
)

var (
	longWindow  [2][2048]float64
	shortWindow [2][256]float64
)

func init() {
	buildWindow(longWindow[shapeSine][:], sineHalf(1024))
	buildWindow(shortWindow[shapeSine][:], sineHalf(128))
	buildWindow(longWindow[shapeKBD][:], kbdHalf(1024, 4.0))
	buildWindow(shortWindow[shapeKBD][:], kbdHalf(128, 6.0))
}

func buildWindow(w []float64, half []float64) {
	n := len(half)
	for i := 0; i < n; i++ {
		w[i] = half[i]
		w[2*n-1-i] = half[i]
	}
}

func sineHalf(n int) []float64 {
	half := make([]float64, n)
	for i := range half {
		half[i] = math.Sin(math.Pi / float64(2*n) * (float64(i) + 0.5))
	}
	return half
}

func kbdHalf(n int, alpha float64) []float64 {
	w := make([]float64, n+1)
	denom := besselI0(math.Pi * alpha)
	for i := 0; i <= n; i++ {
		x := (2*float64(i)/float64(n) - 1)
		w[i] = besselI0(math.Pi*alpha*math.Sqrt(1-x*x)) / denom
	}
	var total float64
	for i := 0; i <= n; i++ {
		total += w[i]
	}
	half := make([]float64, n)
	var running float64
	for i := 0; i < n; i++ {
		running += w[i]
		half[i] = math.Sqrt(running / total)
	}
	return half
}

func besselI0(x float64) float64 {
	var sum, term float64 = 1, 1
	xx := x * x / 4
	for k := 1; k < 64; k++ {
		term *= xx / float64(k*k)
		sum += term
		if term < 1e-18*sum {
			break
		}
	}
	return sum
}
