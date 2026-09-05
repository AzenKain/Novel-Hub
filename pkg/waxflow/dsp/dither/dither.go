// Package dither requantizes float PCM to integer bit depths.
package dither

import (
	"fmt"
	"math"

	"novelhub/pkg/waxflow/waxerr"
)

// Version is this node's algorithm revision for cache keys: bump on any change to the dither, shaping coefficients, or rounding (plan section 10).
const Version = "dither-2"

// DefaultSeed seeds deterministic-mode quantizers.
const DefaultSeed uint64 = 1

// Shaping selects the quantizer's noise strategy.
type Shaping uint8

const (
	TPDF Shaping = iota
	Shaped
	None
)

func (s Shaping) String() string {
	switch s {
	case TPDF:
		return "tpdf"
	case Shaped:
		return "shaped"
	case None:
		return "none"
	default:
		return fmt.Sprintf("Shaping(%d)", uint8(s))
	}
}

var fWeighted = [5]float64{2.033, -2.165, 1.959, -1.590, 0.6149}

// SupportsShaping reports whether Shaped's noise shaping filter suits an output rate.
func SupportsShaping(rate int) bool {
	return rate == 44100 || rate == 48000
}

// Quantizer converts float32 samples in [-1, 1) to integers of a target bit depth, per channel.
type Quantizer struct {
	bits    int
	shaping Shaping
	seed    uint64
	scale   float64
	lo, hi  float64
	errs    [][5]float64
}

func splitmix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

func (q *Quantizer) tpdf(ch int, pos int64) float64 {
	base := q.seed ^ (uint64(ch)+1)*0x9E3779B97F4A7C15
	a := float64(splitmix64(base+2*uint64(pos))>>11) * 0x1p-53
	b := float64(splitmix64(base+2*uint64(pos)+1)>>11) * 0x1p-53
	return a - b
}

// NewQuantizer returns a quantizer producing bits-deep integers (right-justified in int32, the audio.Buffer convention) for the given channel count.
func NewQuantizer(bits, channels int, shaping Shaping, seed uint64) (*Quantizer, error) {
	if bits < 2 || bits > 32 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dither: target depth %d outside 2..32", bits))
	}
	if channels < 1 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dither: channel count %d must be positive", channels))
	}
	if shaping != TPDF && shaping != Shaped && shaping != None {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dither: unknown shaping %d", uint8(shaping)))
	}
	q := &Quantizer{
		bits:    bits,
		shaping: shaping,
		seed:    seed,
		scale:   math.Ldexp(1, bits-1),
		errs:    make([][5]float64, channels),
	}
	q.lo = -q.scale
	q.hi = q.scale - 1
	q.Reset()
	return q, nil
}

// Reset clears the shaping history for a new stream segment.
func (q *Quantizer) Reset() {
	for c := range q.errs {
		q.errs[c] = [5]float64{}
	}
}

// Bits returns the target depth.
func (q *Quantizer) Bits() int { return q.bits }

// Quantize converts channel ch of src into dst, sample for sample.
func (q *Quantizer) Quantize(dst []int32, src []float32, ch int, pos int64) {
	if len(dst) != len(src) {
		panic("dither: dst and src length mismatch")
	}
	errs := &q.errs[ch]
	for i, s := range src {
		v := float64(s) * q.scale
		if math.IsNaN(v) {
			dst[i] = 0
			if q.shaping == Shaped {
				errs[4], errs[3], errs[2], errs[1], errs[0] = errs[3], errs[2], errs[1], errs[0], 0
			}
			continue
		}
		if v > 2*q.scale {
			v = 2 * q.scale
		} else if v < -2*q.scale {
			v = -2 * q.scale
		}
		w := v
		if q.shaping == Shaped {
			for k, h := range fWeighted {
				w -= h * errs[k]
			}
		}
		d := 0.0
		if q.shaping != None {
			d = q.tpdf(ch, pos+int64(i))
		}
		out := math.Floor(w + d + 0.5)
		if q.shaping == Shaped {
			e := out - w
			errs[4], errs[3], errs[2], errs[1], errs[0] = errs[3], errs[2], errs[1], errs[0], e
		}
		if out < q.lo {
			out = q.lo
		} else if out > q.hi {
			out = q.hi
		}
		dst[i] = int32(out)
	}
}
