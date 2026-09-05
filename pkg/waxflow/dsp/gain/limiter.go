package gain

import (
	"fmt"
	"math"
	"time"

	"novelhub/pkg/waxflow/dsp/internal/firwin"
	"novelhub/pkg/waxflow/waxerr"
)

// LimiterVersion is the limiter node's revision for cache keys: bump on any change to detection, smoothing constants, or the clamp (plan section 10).
const LimiterVersion = "limiter-3"

const limiterRelease = 50 * time.Millisecond

// DefaultCeilingDB is the default limiter ceiling in dBTP.
const DefaultCeilingDB = -1.0

// Limiter is a look-ahead true-peak limiter: peaks are detected on a 4x oversampled estimate (BS.1770-4 style), the gain falls before each peak arrives over a 5 ms look-ahead window and releases over 50 ms after it passes, all channels sharing one gain so the stereo image cannot shift.
type Limiter struct {
	channels int
	ceil     float64
	look     int
	aRel     float64

	interp [3][interpTaps]float32

	buf   [][]float32
	peaks []float32
	start int64
	base  int64
	have  int
	pk    int64

	deque    []maxEntry
	dqHead   int
	pushed   int64
	gRel     float64
	box1     boxAvg
	box2     boxAvg
	primed   bool
	draining bool

	clamped int
}

type boxAvg struct {
	ring []int64
	sum  int64
	idx  int
}

const boxScale = 1 << 40

const maxLimiterRate = 1 << 20

func quantizeGain(v float64) int64 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return boxScale
	}
	return int64(v * boxScale)
}

func (b *boxAvg) fill(v float64) {
	q := quantizeGain(v)
	for i := range b.ring {
		b.ring[i] = q
	}
	b.sum = q * int64(len(b.ring))
	b.idx = 0
}

func (b *boxAvg) step(v float64) float64 {
	q := quantizeGain(v)
	b.sum += q - b.ring[b.idx]
	b.ring[b.idx] = q
	if b.idx++; b.idx == len(b.ring) {
		b.idx = 0
	}
	return float64(b.sum) / float64(int64(len(b.ring))*boxScale)
}

type maxEntry struct {
	idx int64
	v   float32
}

const interpTaps = 16

// NewLimiter returns a limiter for one stream.
func NewLimiter(rate, channels int, ceilingDB float64) (*Limiter, error) {
	if rate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: limiter rate %d must be positive", rate))
	}
	if channels < 1 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: limiter channel count %d must be positive", channels))
	}
	if rate > maxLimiterRate {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: limiter rate %d above the %d ceiling", rate, maxLimiterRate))
	}
	if ceilingDB > 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: limiter ceiling %+.2f dB is above full scale", ceilingDB))
	}
	l := &Limiter{
		channels: channels,
		ceil:     FromDB(ceilingDB),
		look:     max(rate/200, 32),
	}
	l.aRel = 1 - math.Exp(-1/(limiterRelease.Seconds()*float64(rate)))
	box := l.look/2 + 1
	l.box1.ring = make([]int64, box)
	l.box2.ring = make([]int64, box)
	l.designInterp()

	capFrames := l.look + interpTaps + 4096
	l.buf = make([][]float32, channels)
	for c := range l.buf {
		l.buf[c] = make([]float32, capFrames)
	}
	l.peaks = make([]float32, capFrames)
	l.Reset()
	return l, nil
}

// Horizon reports the pre-roll a restarted run needs before its output rejoins a continuous run's bit-exactly (dsp.Settler): 2 s, from the 50 ms release.
func (l *Limiter) Horizon() time.Duration {
	return time.Duration(settleTimeConstants * float64(limiterRelease))
}

// Reset clears all state for a new stream segment.
func (l *Limiter) Reset() {
	l.start, l.base, l.have, l.pk, l.pushed = 0, 0, 0, 0, 0
	l.deque = l.deque[:0]
	l.dqHead = 0
	l.gRel = 1
	l.primed = false
	l.clamped = 0
	l.draining = false
}

// Process consumes frames from src and produces limited frames into dst, per channel, in lockstep, returning the counts written and consumed.
func (l *Limiter) Process(dst, src [][]float32) (produced, consumed int) {
	if l.draining {
		panic("gain: limiter Process after Drain")
	}
	if len(dst) != l.channels || len(src) != l.channels {
		panic("gain: limiter channel count mismatch")
	}
	checkFrames("limiter source", src)
	space := checkFrames("limiter destination", dst)
	for {
		l.compact()
		take := min(len(l.buf[0])-l.have, len(src[0])-consumed)
		for c := range l.buf {
			copy(l.buf[c][l.have:], src[c][consumed:consumed+take])
		}
		l.have += take
		consumed += take

		l.detect(false)
		produced += l.emit(dst, produced, space, false)
		if produced >= space || consumed >= len(src[0]) {
			return produced, consumed
		}
	}
}

// Drain flushes the delayed tail after the final Process call.
func (l *Limiter) Drain(dst [][]float32) (produced int) {
	if len(dst) != l.channels {
		panic("gain: limiter channel count mismatch")
	}
	checkFrames("limiter drain destination", dst)
	l.draining = true
	l.detect(true)
	return l.emit(dst, 0, len(dst[0]), true)
}

func (l *Limiter) compact() {
	drop := int(l.base - l.start)
	if drop <= 0 {
		return
	}
	for c := range l.buf {
		copy(l.buf[c], l.buf[c][drop:l.have])
	}
	copy(l.peaks, l.peaks[drop:l.have])
	l.have -= drop
	l.start = l.base
}

func (l *Limiter) detect(draining bool) {
	const half = interpTaps / 2
	end := l.have - half
	if draining {
		end = l.have
	}
	for j := int(l.pk - l.start); j < end; j++ {
		var peak float32
		for c := range l.buf {
			buf := l.buf[c]
			if v := abs32(buf[j]); v > peak {
				peak = v
			}
			if lo := j - half + 1; lo >= 0 && j+half < l.have {
				win := (*[interpTaps]float32)(buf[lo : lo+interpTaps])
				for p := range l.interp {
					phase := &l.interp[p]
					var acc float32
					for t := 0; t < interpTaps; t++ {
						acc += phase[t] * win[t]
					}
					if a := abs32(acc); a > peak {
						peak = a
					}
				}
			} else {
				peak = l.detectEdge(buf, j, peak)
			}
		}
		l.peaks[j] = peak
		l.pk++
	}
}

func (l *Limiter) detectEdge(buf []float32, j int, peak float32) float32 {
	const half = interpTaps / 2
	for p := range l.interp {
		phase := &l.interp[p]
		var acc float32
		for t := 0; t < interpTaps; t++ {
			if i := j - half + 1 + t; i >= 0 && i < l.have {
				acc += phase[t] * buf[i]
			}
		}
		if a := abs32(acc); a > peak {
			peak = a
		}
	}
	return peak
}

func (l *Limiter) emit(dst [][]float32, off, space int, draining bool) (produced int) {
	ceil := float32(l.ceil)
	for off+produced < space {
		n := l.base
		winEnd := n + int64(l.look)
		if !draining && l.pk <= winEnd {
			return produced
		}
		if draining && n >= l.pk {
			return produced
		}

		for ; l.pushed < min(winEnd+1, l.pk); l.pushed++ {
			v := l.peaks[l.pushed-l.start]
			for len(l.deque) > l.dqHead && l.deque[len(l.deque)-1].v <= v {
				l.deque = l.deque[:len(l.deque)-1]
			}
			l.deque = append(l.deque, maxEntry{l.pushed, v})
		}
		for l.dqHead < len(l.deque) && l.deque[l.dqHead].idx < n {
			l.dqHead++
		}
		if l.dqHead > 0 && l.dqHead*2 > len(l.deque) {
			n := copy(l.deque, l.deque[l.dqHead:])
			l.deque = l.deque[:n]
			l.dqHead = 0
		}

		m := 1.0
		if env := float64(l.deque[l.dqHead].v); env > l.ceil {
			m = l.ceil / env
		}
		if rel := l.gRel + (1-l.gRel)*l.aRel; rel < m {
			l.gRel = rel
		} else {
			l.gRel = m
		}

		if !l.primed {
			l.box1.fill(l.gRel)
			l.box2.fill(l.gRel)
			l.primed = true
		}

		i := int(n - l.start)
		g := float32(l.box2.step(l.box1.step(l.gRel)))
		for c := range l.buf {
			v := l.buf[c][i] * g
			if v > ceil {
				v, l.clamped = ceil, l.clamped+1
			} else if v < -ceil {
				v, l.clamped = -ceil, l.clamped+1
			}
			dst[c][off+produced] = v
		}
		produced++
		l.base++
	}
	return produced
}

func (l *Limiter) designInterp() {
	const beta = 3.67
	half := interpTaps / 2
	i0 := firwin.BesselI0(beta)
	for p := 1; p <= 3; p++ {
		c := &l.interp[p-1]
		frac := float64(p) / 4
		var sum float64
		for t := 0; t < interpTaps; t++ {
			x := frac - float64(t-half+1)
			w := firwin.BesselI0(beta*math.Sqrt(1-(x/float64(half))*(x/float64(half)))) / i0
			c[t] = float32(firwin.Sinc(x) * w)
			sum += float64(c[t])
		}
		for t := range c {
			c[t] = float32(float64(c[t]) / sum)
		}
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
