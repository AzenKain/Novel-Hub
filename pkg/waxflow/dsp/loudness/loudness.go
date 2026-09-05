// Package loudness implements the ITU-R BS.1770-4 / EBU R128 loudness meter behind the engine's analysis jobs: gated integrated loudness, loudness range per EBU Tech 3342, and oversampled true peak.
package loudness

import (
	"fmt"
	"math"
	"slices"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

// Version identifies the meter algorithm revision (ADR-0004 style).
const Version = "r128-1"

const (
	momSub = 4
	stSub  = 30
)

const loudnessOffset = -0.691

const absGate = -70.0

var absGatePower = math.Pow(10, (absGate-loudnessOffset)/10)

// Meter measures one stream.
type Meter struct {
	rate     int
	channels int
	weights  []float64

	shelf, hp biquad
	state     []kState

	subLen  int
	subFill int
	subAcc  []float64
	ring    [][]float64
	ringPos int
	ringCnt int64

	blocks []float64
	st     []float64

	tp    *truePeak
	maxSP float64
	maxTP float64

	flushed bool
}

// NewMeter returns a meter for the given sample rate, channel count, and channel layout.
func NewMeter(rate, channels int, layout audio.ChannelMask) (*Meter, error) {
	if rate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("loudness: meter rate %d must be positive", rate))
	}
	if channels <= 0 || channels > 8 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("loudness: meter channel count %d outside 1..8", channels))
	}
	if layout == 0 {
		layout = audio.DefaultLayout(channels)
	}
	m := &Meter{
		rate:     rate,
		channels: channels,
		weights:  channelWeights(channels, layout),
		state:    make([]kState, channels),
		subLen:   max(rate/10, 1),
		subAcc:   make([]float64, channels),
		ring:     make([][]float64, channels),
		tp:       newTruePeak(rate, channels),
	}
	m.shelf, m.hp = kWeighting(rate)
	for c := range m.ring {
		m.ring[c] = make([]float64, stSub)
	}
	return m, nil
}

func channelWeights(channels int, layout audio.ChannelMask) []float64 {
	w := make([]float64, channels)
	for i := range w {
		w[i] = 1
	}
	c := 0
	for bit := 0; bit < 32 && c < channels; bit++ {
		mask := audio.ChannelMask(1) << bit
		if layout&mask == 0 {
			continue
		}
		switch mask {
		case audio.LowFrequency:
			w[c] = 0
		case audio.BackLeft, audio.BackRight, audio.BackCenter,
			audio.SideLeft, audio.SideRight:
			w[c] = 1.41
		}
		c++
	}
	return w
}

// Process consumes one chunk of planar float32 PCM: chans[c][i] is sample i of channel c.
func (m *Meter) Process(chans [][]float32) error {
	if m.flushed {
		return waxerr.New(waxerr.CodeInvalidRequest, "loudness: Process after Flush")
	}
	if len(chans) != m.channels {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("loudness: chunk has %d channels, meter expects %d", len(chans), m.channels))
	}
	n := len(chans[0])
	for _, ch := range chans[1:] {
		if len(ch) != n {
			return waxerr.New(waxerr.CodeInvalidRequest, "loudness: channel slices differ in length")
		}
	}
	for off := 0; off < n; {
		take := n - off
		if rem := m.subLen - m.subFill; take > rem {
			take = rem
		}
		for c := range chans {
			m.consume(c, chans[c][off:off+take])
		}
		m.subFill += take
		off += take
		if m.subFill == m.subLen {
			m.finishSubBlock()
		}
	}
	return nil
}

func (m *Meter) consume(c int, seg []float32) {
	st := &m.state[c]
	acc := m.subAcc[c]
	maxSP, maxTP := m.maxSP, m.maxTP
	for _, s := range seg {
		x := float64(s)
		a := math.Abs(x)
		if a > maxSP {
			maxSP = a
		}
		if m.tp != nil {
			if p := m.tp.push(c, x); p > maxTP {
				maxTP = p
			}
		} else if a > maxTP {
			maxTP = a
		}
		y := m.shelf.b0*x + st.s1a
		st.s1a = m.shelf.b1*x - m.shelf.a1*y + st.s1b
		st.s1b = m.shelf.b2*x - m.shelf.a2*y
		z := m.hp.b0*y + st.s2a
		st.s2a = m.hp.b1*y - m.hp.a1*z + st.s2b
		st.s2b = m.hp.b2*y - m.hp.a2*z
		acc += z * z
	}
	m.subAcc[c] = acc
	m.maxSP, m.maxTP = maxSP, maxTP
}

func (m *Meter) finishSubBlock() {
	for c := range m.subAcc {
		m.ring[c][m.ringPos] = m.subAcc[c]
		m.subAcc[c] = 0
	}
	m.ringPos = (m.ringPos + 1) % stSub
	m.ringCnt++
	m.subFill = 0
	if m.ringCnt >= momSub {
		if p := m.windowPower(momSub); p > absGatePower {
			m.blocks = append(m.blocks, p)
		}
	}
	if m.ringCnt >= stSub {
		if p := m.windowPower(stSub); p > absGatePower {
			m.st = append(m.st, p)
		}
	}
}

func (m *Meter) windowPower(n int) float64 {
	var sum float64
	for c, w := range m.weights {
		if w == 0 {
			continue
		}
		ring := m.ring[c]
		var s float64
		for k := 1; k <= n; k++ {
			s += ring[(m.ringPos-k+stSub)%stSub]
		}
		sum += w * s
	}
	return sum / float64(n*m.subLen)
}

// Flush finalizes measurement.
func (m *Meter) Flush() {
	if m.flushed {
		return
	}
	m.flushed = true
	if m.tp != nil {
		if p := m.tp.drain(); p > m.maxTP {
			m.maxTP = p
		}
	}
}

// Integrated returns the gated integrated loudness in LUFS per BS.1770-4.
func (m *Meter) Integrated() float64 {
	if len(m.blocks) == 0 {
		return math.Inf(-1)
	}
	var sum float64
	for _, p := range m.blocks {
		sum += p
	}
	thresh := sum / float64(len(m.blocks)) / 10
	var gated float64
	var n int
	for _, p := range m.blocks {
		if p > thresh {
			gated += p
			n++
		}
	}
	if n == 0 {
		return math.Inf(-1)
	}
	return loudnessOffset + 10*math.Log10(gated/float64(n))
}

// Range returns the loudness range (LRA) in LU per EBU Tech 3342, verified against the document's four test cases (see TestEBUTech3342Vectors).
func (m *Meter) Range() float64 {
	if len(m.st) == 0 {
		return 0
	}
	var sum float64
	for _, p := range m.st {
		sum += p
	}
	thresh := sum / float64(len(m.st)) / 100
	gated := make([]float64, 0, len(m.st))
	for _, p := range m.st {
		if p >= thresh {
			gated = append(gated, p)
		}
	}
	if len(gated) < 2 {
		return 0
	}
	slices.Sort(gated)
	lo := gated[percentileIndex(len(gated), 0.10)]
	hi := gated[percentileIndex(len(gated), 0.95)]
	return 10 * math.Log10(hi/lo)
}

func percentileIndex(n int, f float64) int {
	return int(f*float64(n-1) + 0.5)
}

// TruePeak returns the maximum true-peak level in dBTP (oversampled per BS.1770-4 Annex 2; rates above 192 kHz are dense enough that the sample grid is used directly).
func (m *Meter) TruePeak() float64 {
	return dbOrNegInf(m.maxTP)
}

// SamplePeak returns the maximum absolute sample level in dBFS, or math.Inf(-1) for silence.
func (m *Meter) SamplePeak() float64 {
	return dbOrNegInf(m.maxSP)
}

func dbOrNegInf(v float64) float64 {
	if v <= 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(v)
}
