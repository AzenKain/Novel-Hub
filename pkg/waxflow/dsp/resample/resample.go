// Package resample converts PCM sample rates with a streaming Kaiser windowed-sinc polyphase filter.
package resample

import (
	"fmt"

	"novelhub/pkg/waxflow/waxerr"
)

// Profile selects the quality/cost trade of the anti-alias filter.
type Profile string

const (
	HQ   Profile = "hq"
	Fast Profile = "fast"
)

// Version returns the profile's algorithm revision for cache keys.
func (p Profile) Version() string {
	return "resample-" + string(p) + "-1"
}

// ParseProfile resolves a profile name; the empty string means HQ.
func ParseProfile(name string) (Profile, error) {
	switch p := Profile(name); {
	case name == "":
		return HQ, nil
	case p.valid():
		return p, nil
	default:
		return "", waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("resample: unknown profile %q (%s, %s)", name, HQ, Fast))
	}
}

func (p Profile) valid() bool { return p == HQ || p == Fast }

const maxBankFloats = 4 << 20

// Resampler is a streaming rational resampler for one PCM stream of one or more channels, all converted in lockstep.
type Resampler struct {
	bank     *bank
	channels int

	win      [][]float32
	winStart int64
	have     int

	inCount  int64
	outCount int64
	off      int
	draining bool
	padded   int64
}

// New returns a Resampler converting inRate to outRate for the given channel count.
func New(inRate, outRate, channels int, profile Profile) (*Resampler, error) {
	if inRate <= 0 || outRate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("resample: rates must be positive, got %d -> %d", inRate, outRate))
	}
	if inRate == outRate {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("resample: input and output rate are both %d", inRate))
	}
	if channels < 1 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("resample: channel count %d must be positive", channels))
	}
	if !profile.valid() {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("resample: unknown profile %q", profile))
	}
	b, err := bankFor(inRate, outRate, profile)
	if err != nil {
		return nil, err
	}
	r := &Resampler{bank: b, channels: channels}
	r.win = make([][]float32, channels)
	winLen := b.tpp - 1 + max(4096, b.tpp)
	for c := range r.win {
		r.win[c] = make([]float32, winLen)
	}
	r.Reset(0)
	return r, nil
}

// Ratio returns the reduced conversion ratio L/M (outRate/inRate).
func (r *Resampler) Ratio() (l, m int) { return r.bank.l, r.bank.m }

// GroupDelay reports the anti-alias filter's group delay as the rational num/den in input samples.
func (r *Resampler) GroupDelay() (num, den int) { return r.bank.delay, r.bank.l }

// Reset clears all filter state for a new stream segment (a seek or splice).
func (r *Resampler) Reset(phaseOff int) {
	if phaseOff < 0 || phaseOff >= r.bank.m {
		panic(fmt.Sprintf("resample: phase offset %d outside [0, %d)", phaseOff, r.bank.m))
	}
	hist := r.bank.tpp - 1
	for c := range r.win {
		clear(r.win[c][:hist])
	}
	r.winStart = -int64(hist)
	r.have = hist
	r.inCount = 0
	r.outCount = 0
	r.off = phaseOff
	r.draining = false
	r.padded = 0
}

// OffsetFor returns the anchor pair for a stream segment that starts at absolute input sample pos: the absolute output index of the segment's first output sample, and the phase offset to pass to Reset so that output sample lands exactly on the output timeline grid.
func (r *Resampler) OffsetFor(pos int64) (outPos int64, phaseOff int) {
	l, m := int64(r.bank.l), int64(r.bank.m)
	outPos = ceilDiv(pos*l, m)
	phaseOff = int(outPos*m - pos*l)
	return outPos, phaseOff
}

// OutputLen returns the exact number of output frames a full stream of n input frames yields: ceil(n*outRate/inRate) in reduced terms.
func OutputLen(n int64, inRate, outRate int) int64 {
	if n < 0 {
		return -1
	}
	g := gcd(inRate, outRate)
	return ceilDiv(n*int64(outRate/g), int64(inRate/g))
}

// Process consumes frames from src and produces converted frames into dst, per channel, in lockstep.
func (r *Resampler) Process(dst, src [][]float32) (produced, consumed int) {
	if r.draining {
		panic("resample: Process after Drain")
	}
	return r.run(dst, src)
}

// Drain finalizes the stream: the input seen so far is the whole segment, and the filter tail is flushed by feeding zeros until exactly OutputLen-many frames (adjusted for the anchor phase) have been produced.
func (r *Resampler) Drain(dst [][]float32) (produced int) {
	r.draining = true
	produced, _ = r.run(dst, nil)
	return produced
}

func (r *Resampler) outTarget() int64 {
	l, m := int64(r.bank.l), int64(r.bank.m)
	return ceilDiv(r.inCount*l-int64(r.off), m)
}

func (r *Resampler) run(dst, src [][]float32) (produced, consumed int) {
	if len(dst) != r.channels || (src != nil && len(src) != r.channels) {
		panic("resample: channel count mismatch")
	}
	b := r.bank
	tpp, l, m := b.tpp, int64(b.l), int64(b.m)
	space := len(dst[0])
	limit := int64(-1)
	if src == nil {
		limit = r.outTarget()
	}
	for {
		for produced < space {
			if limit >= 0 && r.outCount >= limit {
				return produced, consumed
			}
			u := r.outCount*m + int64(b.delay) + int64(r.off)
			k := u / l
			if k >= r.winStart+int64(r.have) {
				break
			}
			p := int(u % l)
			base := int(k-r.winStart) - (tpp - 1)
			coef := b.coef[p*tpp : (p+1)*tpp]
			if r.channels == 2 {
				y0, y1 := dot2(coef, r.win[0][base:base+tpp], r.win[1][base:base+tpp])
				dst[0][produced], dst[1][produced] = y0, y1
			} else {
				for c := 0; c < r.channels; c++ {
					dst[c][produced] = dot(coef, r.win[c][base:base+tpp])
				}
			}
			produced++
			r.outCount++
		}
		if produced >= space {
			return produced, consumed
		}

		u := r.outCount*m + int64(b.delay) + int64(r.off)
		if drop := int(u/l - int64(tpp-1) - r.winStart); drop > 0 {
			drop = min(drop, r.have)
			for c := range r.win {
				copy(r.win[c], r.win[c][drop:r.have])
			}
			r.winStart += int64(drop)
			r.have -= drop
		}

		free := len(r.win[0]) - r.have
		if free == 0 {
			panic("resample: window deadlock")
		}
		take := 0
		if src != nil {
			take = min(free, len(src[0])-consumed)
			if take > 0 {
				for c := range r.win {
					copy(r.win[c][r.have:], src[c][consumed:consumed+take])
				}
				consumed += take
				r.inCount += int64(take)
			}
		} else {
			take = free
			for c := range r.win {
				clear(r.win[c][r.have : r.have+take])
			}
			r.padded += int64(take)
		}
		if take == 0 {
			return produced, consumed
		}
		r.have += take
	}
}

func dot(c, x []float32) float32 {
	x = x[:len(c)]
	var s0, s1, s2, s3 float32
	n := len(c) &^ 3
	for j := 0; j < n; j += 4 {
		s0 += c[j] * x[j]
		s1 += c[j+1] * x[j+1]
		s2 += c[j+2] * x[j+2]
		s3 += c[j+3] * x[j+3]
	}
	for j := n; j < len(c); j++ {
		s0 += c[j] * x[j]
	}
	return (s0 + s1) + (s2 + s3)
}

func dot2(c, x0, x1 []float32) (float32, float32) {
	x0 = x0[:len(c)]
	x1 = x1[:len(c)]
	var a0, a1, b0, b1 float32
	n := len(c) &^ 1
	for j := 0; j < n; j += 2 {
		c0, c1 := c[j], c[j+1]
		a0 += c0 * x0[j]
		a1 += c1 * x0[j+1]
		b0 += c0 * x1[j]
		b1 += c1 * x1[j+1]
	}
	if n < len(c) {
		a0 += c[n] * x0[n]
		b0 += c[n] * x1[n]
	}
	return a0 + a1, b0 + b1
}

func ceilDiv(a, b int64) int64 {
	q, r := a/b, a%b
	if r > 0 {
		q++
	}
	return q
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
