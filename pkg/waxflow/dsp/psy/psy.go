// Package psy is the shared psychoacoustic model behind the lossy encoders.
package psy

import (
	"fmt"
	"math"

	"novelhub/pkg/waxflow/waxerr"
)

// Version is the model's algorithm revision.
const Version = "psy-1"

const (
	tmnDB   = 29.0
	nmtDB   = 6.0
	rpelev  = 2.0
	athSPL  = 96.0
	maxPart = 1.0 / 3.0
)

// Config declares one analysis geometry.
type Config struct {
	Rate        int
	Lines       int
	FFTSize     int
	BandOffsets []int
	NoPredict   bool
	FixedC      float64
	OffsetDB    float64
	ATHOffsetDB float64
}

// Result is one block's analysis.
type Result struct {
	Thr    []float64
	Energy []float64
	PE     float64
}

type partition struct {
	lo, hi int
	bval   float64
	minDB  float64
	qthr   float64
}

// Model is one channel's analysis state for one geometry.
type Model struct {
	cfg                 Config
	win                 []float64
	fft                 *fftPlan
	parts               []partition
	spread              []float64
	bandParts           [][]bandPart
	bandLines           [][2]int
	xw, re, im          []float64
	e, c                []float64
	pe, pc              []float64
	thrP                []float64
	prevThr             []float64
	thr, energy         []float64
	rPrev, cPrev, sPrev [2][]float64
	frames              int
}

type bandPart struct {
	p    int
	frac float64
}

// New builds a Model for the geometry.
func New(cfg Config) (*Model, error) {
	if cfg.Rate <= 0 || cfg.Lines <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "psy: rate and lines must be positive")
	}
	if cfg.FFTSize < 64 || cfg.FFTSize&(cfg.FFTSize-1) != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("psy: FFT size %d is not a power of two >= 64", cfg.FFTSize))
	}
	if n := len(cfg.BandOffsets); n < 2 || cfg.BandOffsets[0] != 0 || cfg.BandOffsets[n-1] != cfg.Lines {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			"psy: band offsets must run 0..Lines")
	}
	for i := 1; i < len(cfg.BandOffsets); i++ {
		if cfg.BandOffsets[i] <= cfg.BandOffsets[i-1] {
			return nil, waxerr.New(waxerr.CodeInvalidRequest,
				"psy: band offsets must be strictly increasing")
		}
	}
	if cfg.NoPredict && !(cfg.FixedC >= 0 && cfg.FixedC <= 1) {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "psy: FixedC outside [0,1]")
	}

	m := &Model{cfg: cfg, fft: newFFTPlan(cfg.FFTSize)}
	n := cfg.FFTSize
	m.win = make([]float64, n)
	for i := range m.win {
		m.win[i] = 0.5 - 0.5*math.Cos(2*math.Pi*(float64(i)+0.5)/float64(n))
	}

	m.buildPartitions()
	m.buildSpreading()
	m.buildBandMap()

	half := n / 2
	m.xw = make([]float64, n)
	m.re = make([]float64, n)
	m.im = make([]float64, n)
	m.e = make([]float64, half)
	m.c = make([]float64, half)
	np := len(m.parts)
	m.pe = make([]float64, np)
	m.pc = make([]float64, np)
	m.thrP = make([]float64, np)
	m.prevThr = make([]float64, np)
	nb := len(cfg.BandOffsets) - 1
	m.thr = make([]float64, nb)
	m.energy = make([]float64, nb)
	if !cfg.NoPredict {
		for h := range m.rPrev {
			m.rPrev[h] = make([]float64, half)
			m.cPrev[h] = make([]float64, half)
			m.sPrev[h] = make([]float64, half)
		}
	}
	return m, nil
}

// Bands returns the band count the geometry produces.
func (m *Model) Bands() int { return len(m.cfg.BandOffsets) - 1 }

// Reset clears prediction history and pre-echo memory, for use after seeks or stream splices.
func (m *Model) Reset() {
	m.frames = 0
	for h := range m.rPrev {
		if m.rPrev[h] != nil {
			clear(m.rPrev[h])
			clear(m.cPrev[h])
			clear(m.sPrev[h])
		}
	}
	clear(m.prevThr)
}

func bark(f float64) float64 {
	return 13*math.Atan(0.00076*f) + 3.5*math.Atan((f/7500)*(f/7500))
}

func athDB(f float64) float64 {
	if f < 20 {
		return 120
	}
	k := f / 1000
	v := 3.64*math.Pow(k, -0.8) - 6.5*math.Exp(-0.6*(k-3.3)*(k-3.3)) + 1e-3*k*k*k*k
	return math.Min(math.Max(v, -12), 120)
}

func (m *Model) buildPartitions() {
	half := m.cfg.FFTSize / 2
	df := float64(m.cfg.Rate) / float64(m.cfg.FFTSize)
	fsLine := float64(m.cfg.FFTSize) / 4
	fsEnergy := fsLine * fsLine

	athScale := math.Pow(10, -m.cfg.ATHOffsetDB/10)
	lo := 0
	for lo < half {
		hi := lo + 1
		bLo := bark(float64(lo) * df)
		for hi < half && bark(float64(hi+1)*df)-bLo <= maxPart {
			hi++
		}
		center := (float64(lo) + float64(hi)) / 2 * df
		bval := bark(center)
		qthr := 0.0
		for w := lo; w < hi; w++ {
			qthr += math.Pow(10, (athDB(float64(w)*df)-athSPL)/10) * fsEnergy * athScale
		}
		m.parts = append(m.parts, partition{
			lo: lo, hi: hi, bval: bval,
			minDB: minvalDB(bval),
			qthr:  qthr,
		})
		lo = hi
	}
}

func minvalDB(bval float64) float64 {
	if bval <= 12 {
		return 24.5
	}
	return math.Max(0, 24.5-(bval-12)*(24.5/6))
}

func spreadDB(dz float64) float64 {
	x := 1.05 * dz
	d := x - 0.5
	extra := 8 * math.Min(d*d-2*d, 0)
	y := 15.811389 + 7.5*(x+0.474) - 17.5*math.Sqrt(1+(x+0.474)*(x+0.474))
	if y < -100 {
		return -1000
	}
	return extra + y
}

func (m *Model) buildSpreading() {
	np := len(m.parts)
	m.spread = make([]float64, np*np)
	for i := 0; i < np; i++ {
		row := m.spread[i*np : (i+1)*np]
		sum := 0.0
		for j := 0; j < np; j++ {
			s := math.Pow(10, spreadDB(m.parts[i].bval-m.parts[j].bval)/10)
			row[j] = s
			sum += s
		}
		for j := range row {
			row[j] /= sum
		}
	}
}

func (m *Model) buildBandMap() {
	nb := len(m.cfg.BandOffsets) - 1
	m.bandParts = make([][]bandPart, nb)
	m.bandLines = make([][2]int, nb)
	scale := float64(m.cfg.FFTSize) / (2 * float64(m.cfg.Lines))
	half := float64(m.cfg.FFTSize / 2)
	for b := 0; b < nb; b++ {
		loF := math.Min(float64(m.cfg.BandOffsets[b])*scale, half)
		hiF := math.Min(float64(m.cfg.BandOffsets[b+1])*scale, half)
		li := int(loF)
		hi := int(math.Ceil(hiF))
		m.bandLines[b] = [2]int{li, hi}
		for p := range m.parts {
			pLo, pHi := float64(m.parts[p].lo), float64(m.parts[p].hi)
			ov := math.Min(hiF, pHi) - math.Max(loF, pLo)
			if ov <= 0 {
				continue
			}
			m.bandParts[b] = append(m.bandParts[b], bandPart{p: p, frac: ov / (pHi - pLo)})
		}
	}
}

// Analyze runs the model over one block.
func (m *Model) Analyze(x []float32) (Result, error) {
	n := m.cfg.FFTSize
	if len(x) != n {
		return Result{}, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("psy: block is %d samples, geometry wants %d", len(x), n))
	}
	for i := 0; i < n; i++ {
		m.xw[i] = float64(x[i]) * m.win[i]
		m.im[i] = 0
	}
	copy(m.re, m.xw)
	m.fft.transform(m.re, m.im)

	half := n / 2
	cur, old := m.frames%2, (m.frames+1)%2
	for w := 0; w < half; w++ {
		re, im := m.re[w], m.im[w]
		e := re*re + im*im
		r := math.Sqrt(e)
		m.e[w] = e
		if m.cfg.NoPredict {
			m.c[w] = m.cfg.FixedC
			continue
		}
		cf, sf := 1.0, 0.0
		if r > 0 {
			cf, sf = re/r, im/r
		}
		if m.frames < 2 {
			m.c[w] = 0.4
		} else {
			r1, c1, s1 := m.rPrev[cur][w], m.cPrev[cur][w], m.sPrev[cur][w]
			c2, s2 := m.cPrev[old][w], m.sPrev[old][w]
			rp := 2*r1 - m.rPrev[old][w]
			c11 := 2*c1*c1 - 1
			s11 := 2 * s1 * c1
			cp := c11*c2 + s11*s2
			sp := s11*c2 - c11*s2
			dx := r*cf - rp*cp
			dy := r*sf - rp*sp
			den := r + math.Abs(rp)
			if den > 0 {
				m.c[w] = math.Min(math.Sqrt(dx*dx+dy*dy)/den, 1)
			} else {
				m.c[w] = 0.4
			}
		}
		m.rPrev[old][w] = r
		m.cPrev[old][w] = cf
		m.sPrev[old][w] = sf
	}
	m.frames++

	for p := range m.parts {
		e, c := 0.0, 0.0
		for w := m.parts[p].lo; w < m.parts[p].hi; w++ {
			e += m.e[w]
			c += m.c[w] * m.e[w]
		}
		m.pe[p] = e
		m.pc[p] = c
	}

	np := len(m.parts)
	pe := 0.0
	for i := 0; i < np; i++ {
		row := m.spread[i*np : (i+1)*np]
		ecb, ctb := 0.0, 0.0
		for j := 0; j < np; j++ {
			ecb += row[j] * m.pe[j]
			ctb += row[j] * m.pc[j]
		}
		cb := 1.0
		if ecb > 0 {
			cb = math.Min(math.Max(ctb/ecb, 1e-10), 1)
		}
		tb := math.Min(math.Max(-0.299-0.43*math.Log(cb), 0), 1)
		snr := math.Max(m.parts[i].minDB, tmnDB*tb+nmtDB*(1-tb)) + m.cfg.OffsetDB
		nb := ecb * math.Pow(10, -snr/10)
		if m.frames > 1 {
			nb = math.Min(nb, rpelev*m.prevThr[i])
		}
		thr := math.Max(m.parts[i].qthr, nb)
		m.thrP[i] = thr
		m.prevThr[i] = thr
		if m.pe[i] > thr {
			w := float64(m.parts[i].hi - m.parts[i].lo)
			pe += w * math.Log2(m.pe[i]/thr)
		}
	}

	for b := range m.thr {
		t := 0.0
		for _, bp := range m.bandParts[b] {
			t += m.thrP[bp.p] * bp.frac
		}
		m.thr[b] = t
		e := 0.0
		for w := m.bandLines[b][0]; w < m.bandLines[b][1]; w++ {
			e += m.e[w]
		}
		m.energy[b] = e
	}
	return Result{Thr: m.thr, Energy: m.energy, PE: pe}, nil
}
