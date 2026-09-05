package vorbis

import (
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
)

var (
	_ codec.Decoder  = (*Decoder)(nil)
	_ codec.Releaser = (*Decoder)(nil)
)

// Decoder decodes a Vorbis I stream into planar float32 buffers.
type Decoder struct {
	cfg     Config
	fmt     audio.Format
	chOrder []int

	floors      []floorState
	spec        [][]float32
	prev        [][]float64
	floorUnused []bool
	doNotDecode []bool
	prevN       int
	havePrev    bool

	modeBits            int
	planShort, planLong *imdctPlan
	residueVec          []float32
	resVecs             [][]float32
	resSkip             []bool

	timeBuf []float64
	cr, ci  []float64
	out     *audio.Buffer
	maxN    int
}

func (d *Decoder) planFor(n int) *imdctPlan {
	if n == d.cfg.blockSizes[1] {
		return d.planLong
	}
	return d.planShort
}

// NewDecoder returns a decoder for a parsed Config.
func NewDecoder(cfg Config, f audio.Format) (*Decoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.Channels != cfg.Channels || f.Rate != cfg.Rate {
		return nil, malformed("track format %v does not match stream %dHz %dch", f, cfg.Rate, cfg.Channels)
	}
	n := cfg.blockSizes[1]
	d := &Decoder{
		cfg:         cfg,
		fmt:         f,
		chOrder:     waveFromVorbis(cfg.Channels),
		floors:      make([]floorState, cfg.Channels),
		spec:        make([][]float32, cfg.Channels),
		prev:        make([][]float64, cfg.Channels),
		floorUnused: make([]bool, cfg.Channels),
		doNotDecode: make([]bool, cfg.Channels),
		timeBuf:     make([]float64, n),
		cr:          make([]float64, n),
		ci:          make([]float64, n),
		maxN:        n / 2,
	}
	for c := 0; c < cfg.Channels; c++ {
		d.spec[c] = make([]float32, n/2)
		d.prev[c] = make([]float64, n)
	}
	d.planShort = getPlan(cfg.blockSizes[0])
	d.planLong = getPlan(cfg.blockSizes[1])
	d.modeBits = ModeBits(cfg)
	d.residueVec = make([]float32, maxDim(cfg.codebooks))
	return d, nil
}

// Decode decodes one Vorbis audio packet, emitting one buffer of overlap-added samples (or none for the priming packet after a reset).
func (d *Decoder) Decode(pkt []byte, emit func(*audio.Buffer) error) error {
	if len(pkt) == 0 {
		return nil
	}
	r := newBitReader(pkt)
	if r.bit() != 0 {
		return malformed("packet is not an audio packet")
	}
	modeNum := int(r.read(d.modeBits))
	if modeNum >= len(d.cfg.modes) {
		return malformed("mode %d of %d", modeNum, len(d.cfg.modes))
	}
	m := d.cfg.modes[modeNum]
	long := m.blockflag
	n := d.cfg.blockSizes[0]
	if long {
		n = d.cfg.blockSizes[1]
	}
	n2 := n / 2

	prevFlag, nextFlag := false, false
	if long {
		prevFlag = r.bit() == 1
		nextFlag = r.bit() == 1
	}

	mp := &d.cfg.mappings[m.mapping]

	for ch := 0; ch < d.cfg.Channels; ch++ {
		clear(d.spec[ch][:n2])
		fl := d.cfg.floors[mp.submaps[mp.mux[ch]].floor]
		unused, err := fl.decode(r, d.cfg.codebooks, &d.floors[ch])
		if err != nil {
			return err
		}
		d.floorUnused[ch] = unused
		d.doNotDecode[ch] = unused
	}

	for i := range mp.couplingMag {
		mag, ang := mp.couplingMag[i], mp.couplingAng[i]
		if !d.doNotDecode[mag] || !d.doNotDecode[ang] {
			d.doNotDecode[mag] = false
			d.doNotDecode[ang] = false
		}
	}

	for s := range mp.submaps {
		vecs := d.resVecs[:0]
		skip := d.resSkip[:0]
		for ch := 0; ch < d.cfg.Channels; ch++ {
			if mp.mux[ch] == s {
				vecs = append(vecs, d.spec[ch])
				skip = append(skip, d.doNotDecode[ch])
			}
		}
		d.resVecs, d.resSkip = vecs, skip
		if len(vecs) == 0 {
			continue
		}
		res := &d.cfg.residues[mp.submaps[s].residue]
		if err := res.decode(r, d.cfg.codebooks, vecs, skip, n2, d.residueVec); err != nil {
			return err
		}
	}

	for i := len(mp.couplingMag) - 1; i >= 0; i-- {
		mag := d.spec[mp.couplingMag[i]]
		ang := d.spec[mp.couplingAng[i]]
		for j := 0; j < n2; j++ {
			mv, av := mag[j], ang[j]
			var newM, newA float32
			if mv > 0 {
				if av > 0 {
					newM, newA = mv, mv-av
				} else {
					newA, newM = mv, mv+av
				}
			} else {
				if av > 0 {
					newM, newA = mv, mv+av
				} else {
					newA, newM = mv, mv-av
				}
			}
			mag[j], ang[j] = newM, newA
		}
	}

	for ch := 0; ch < d.cfg.Channels; ch++ {
		if d.floorUnused[ch] {
			clear(d.spec[ch][:n2])
			continue
		}
		fl := d.cfg.floors[mp.submaps[mp.mux[ch]].floor]
		fl.apply(&d.floors[ch], d.spec[ch], n2)
	}

	ln, rn := n, n
	if long {
		if !prevFlag {
			ln = d.cfg.blockSizes[0]
		}
		if !nextFlag {
			rn = d.cfg.blockSizes[0]
		}
	}
	plan := d.planFor(n)
	leftWin := d.planFor(ln).window
	rightWin := d.planFor(rn).window

	if !d.havePrev {
		for ch := 0; ch < d.cfg.Channels; ch++ {
			plan.imdct(d.spec[ch], d.timeBuf, d.cr, d.ci)
			applyWindow(d.timeBuf, n, ln, rn, leftWin, rightWin)
			copy(d.prev[ch], d.timeBuf[:n])
		}
		d.prevN = n
		d.havePrev = true
		return nil
	}

	l := (d.prevN + n) / 4
	if l <= 0 {
		d.prevN = n
		return nil
	}
	if d.out == nil || d.out.Cap() < l || d.out.Fmt != d.fmt {
		audio.Put(d.out)
		d.out = audio.Get(d.fmt, max(l, d.maxN))
	}
	d.out.N = l
	shift := (d.prevN - n) / 4
	for wc := 0; wc < d.cfg.Channels; wc++ {
		vc := d.chOrder[wc]
		plan.imdct(d.spec[vc], d.timeBuf, d.cr, d.ci)
		applyWindow(d.timeBuf, n, ln, rn, leftWin, rightWin)
		dst := d.out.ChanF(wc)
		prev := d.prev[vc]
		for j := 0; j < l; j++ {
			var v float64
			if pi := j + d.prevN/2; pi < d.prevN {
				v += prev[pi]
			}
			if ci := j - shift; ci >= 0 && ci < n {
				v += d.timeBuf[ci]
			}
			dst[j] = float32(v)
		}
		copy(prev[:n], d.timeBuf[:n])
	}
	d.prevN = n
	return emit(d.out)
}

// Drain flushes decoder latency.
func (d *Decoder) Drain(func(*audio.Buffer) error) error { return nil }

// Reset discards overlap state after a seek so the next packet primes.
func (d *Decoder) Reset() {
	d.havePrev = false
	d.prevN = 0
}

// Release returns the pooled output buffer.
func (d *Decoder) Release() {
	audio.Put(d.out)
	d.out = nil
}
