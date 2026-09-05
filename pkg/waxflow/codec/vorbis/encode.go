package vorbis

import (
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/dsp/psy"
	"novelhub/pkg/waxflow/waxerr"
)

var _ codec.Encoder = (*Encoder)(nil)

// EncoderVersion identifies the encode algorithm/bitstream revision for the ADR-0004 cache key.
const EncoderVersion = "vorbis-enc-9+" + psy.Version

const encVendor = "WaxFlow"

// EncoderOptions configures NewEncoder.
type EncoderOptions struct {
	Quality float64
	Bitrate int
}

// DefaultQuality is the VBR quality used when EncoderOptions.Quality is unset.
const DefaultQuality = 3.0

type plannedBlock struct {
	size   int
	center int64
}

// Encoder is a Vorbis I encoder producing raw Vorbis audio packets; a container muxer (Ogg, Matroska) frames them and carries CodecConfig's three headers.
type Encoder struct {
	fmt      audio.Format
	channels int
	cfg      *encConfig
	config   []byte

	quality  float64
	offsetDB float64

	fwd    [2]*mdctForward
	psy    [2][]*psy.Model
	attack *psy.AttackDetector
	long   int
	short  int

	buf       [][]float32
	bufBase   int64
	inSamples int64

	pending      []plannedBlock
	lastCenter   int64
	lastN        int
	prevN        int
	firstBlock   bool
	plannedFirst bool
	shortRun     int
	scanPos      int64
	attackPos    []int64
	decoded      int64

	wbuf    []float32
	spec    [][]float32
	curve   [][]float32
	resid   [][]float32
	vals    [][]int
	targets []int
	classes [][]int
	thrLine []float64
	fFinal  []int
	fStep2  []bool
	monobuf []float32
	psyIn   []float32
	w       bitWriter
}

// NewEncoder returns a Vorbis encoder for f, which must be float32 (Vorbis is inherently float) with 1..MaxChannels channels at a positive rate.
func NewEncoder(f audio.Format, opts *EncoderOptions) (*Encoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.BitDepth != 32 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "vorbis: encoder input must be float32")
	}
	if f.Channels < 1 || f.Channels > maxChannels {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("vorbis: %d channels outside 1..%d", f.Channels, maxChannels))
	}
	quality := DefaultQuality
	if opts != nil && opts.Quality != 0 {
		quality = opts.Quality
	}
	if quality < -1 || quality > 10 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("vorbis: quality %g outside -1..10", quality))
	}
	if opts != nil && opts.Bitrate != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			"vorbis: ABR (Bitrate target) is not implemented; leave Bitrate 0 for quality-driven VBR")
	}

	cfg := newEncConfig(f.Channels, f.Rate)
	long := cfg.blockSizes[slotLong]
	short := cfg.blockSizes[slotShort]
	e := &Encoder{
		fmt:      f,
		channels: f.Channels,
		cfg:      cfg,
		quality:  quality,
		offsetDB: qualityToOffsetDB(quality),
		long:     long,
		short:    short,
		attack:   psy.NewAttackDetector(attackRatio),
	}
	e.fwd[slotLong] = newMDCTForward(long)
	e.fwd[slotShort] = newMDCTForward(short)
	e.config = cfg.codecConfig(encVendor, nil)

	maxN2 := long / 2
	maxPosts := len(cfg.floors[slotLong].xs)
	e.wbuf = make([]float32, long)
	e.monobuf = make([]float32, short)
	e.buf = make([][]float32, f.Channels)
	e.spec = make([][]float32, f.Channels)
	e.curve = make([][]float32, f.Channels)
	e.resid = make([][]float32, f.Channels)
	e.vals = make([][]int, f.Channels)
	e.classes = make([][]int, f.Channels)
	for s := 0; s < 2; s++ {
		e.psy[s] = make([]*psy.Model, f.Channels)
	}
	e.bufBase = -int64(long / 2)
	for c := 0; c < f.Channels; c++ {
		e.buf[c] = make([]float32, long/2, long/2+4*long)
		e.spec[c] = make([]float32, maxN2)
		e.curve[c] = make([]float32, maxN2)
		e.resid[c] = make([]float32, maxN2)
		e.vals[c] = make([]int, maxPosts)
		e.classes[c] = make([]int, maxN2/resPartSize)
		for s := 0; s < 2; s++ {
			n2 := cfg.blockSizes[s] / 2
			m, err := newPsyModel(f.Rate, n2, e.offsetDB)
			if err != nil {
				return nil, err
			}
			e.psy[s][c] = m
		}
	}
	e.thrLine = make([]float64, maxN2)
	e.targets = make([]int, maxPosts)
	e.fFinal = make([]int, maxPosts)
	e.fStep2 = make([]bool, maxPosts)

	e.lastCenter = -int64(long / 2)
	e.lastN = long
	e.prevN = long
	e.firstBlock = true
	e.scanPos = e.bufBase
	return e, nil
}

// InputFormat implements codec.Encoder: Vorbis consumes the source float format unchanged (native rate and channel count).
func (e *Encoder) InputFormat() audio.Format { return e.fmt }

// FrameSize implements codec.Encoder.
func (e *Encoder) FrameSize() int { return 0 }

// Bitrate reports the rate the stream can be relied on to hold.
func (e *Encoder) Bitrate() int { return 0 }

// Delay reports the encoder priming to trim from the front of the decoded output.
func (e *Encoder) Delay() int { return 0 }

// CodecConfig returns the three Vorbis headers packed with Xiph lacing.
func (e *Encoder) CodecConfig() []byte { return e.config }

func (e *Encoder) bufEnd() int64 { return e.bufBase + int64(len(e.buf[0])) }

// Encode buffers src and emits every whole block that becomes available.
func (e *Encoder) Encode(src *audio.Buffer, emit func(codec.Packet) error) error {
	if src.Fmt != e.fmt {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("vorbis: encode input %v disagrees with %v", src.Fmt, e.fmt))
	}
	for c := 0; c < e.channels; c++ {
		e.buf[c] = append(e.buf[c], src.ChanF(c)[:src.N]...)
	}
	e.inSamples += int64(src.N)
	return e.drain(emit)
}

func (e *Encoder) drain(emit func(codec.Packet) error) error {
	for {
		e.scanTransients(false)
		for len(e.pending) < 2 && e.planNext(false) {
		}
		if len(e.pending) < 2 {
			return nil
		}
		blk := e.pending[0]
		if e.bufEnd() < blk.center+int64(blk.size/2) {
			return nil
		}
		if err := e.emitPending(emit, e.pending[1].size, false); err != nil {
			return err
		}
	}
}

func (e *Encoder) planNext(flush bool) bool {
	detectTo := e.lastCenter + int64(e.long)
	if !flush && e.scanPos < detectTo {
		return false
	}
	size := e.long
	if e.plannedFirst {
		size = e.decideSize(e.lastCenter)
	} else {
		e.plannedFirst = true
	}
	center := e.lastCenter + int64((e.lastN+size)/4)
	e.pending = append(e.pending, plannedBlock{size: size, center: center})
	e.lastCenter = center
	e.lastN = size
	return true
}

func (e *Encoder) decideSize(from int64) int {
	if e.attackBetween(from, from+int64(e.long)) {
		e.shortRun = shortRunBlocks
	}
	if e.shortRun > 0 {
		e.shortRun--
		return e.short
	}
	return e.long
}

func (e *Encoder) emitPending(emit func(codec.Packet) error, nextN int, last bool) error {
	blk := e.pending[0]
	if err := e.emitBlock(emit, blk.size, e.prevN, nextN); err != nil {
		return err
	}
	e.prevN = blk.size
	e.pending = e.pending[1:]
	if !last {
		e.trimBuffer()
	}
	return nil
}

func (e *Encoder) trimBuffer() {
	if len(e.pending) == 0 {
		return
	}
	keepFrom := e.pending[0].center - int64(e.long)
	drop := int(keepFrom - e.bufBase)
	if drop <= 0 {
		return
	}
	if drop > len(e.buf[0]) {
		drop = len(e.buf[0])
	}
	for c := 0; c < e.channels; c++ {
		e.buf[c] = append(e.buf[c][:0], e.buf[c][drop:]...)
	}
	e.bufBase += int64(drop)
	k := 0
	for k < len(e.attackPos) && e.attackPos[k] < e.bufBase {
		k++
	}
	if k > 0 {
		e.attackPos = append(e.attackPos[:0], e.attackPos[k:]...)
	}
}

func (e *Encoder) emitBlock(emit func(codec.Packet) error, n, prevN, nextN int) error {
	slot := e.cfg.slotFor(n)
	n2 := n / 2
	fl := e.cfg.floors[slot]
	res := e.cfg.residues[slot]
	postCount := len(fl.xs)
	ln, rn := neighbourWin(n, prevN), neighbourWin(n, nextN)
	center := e.pending[0].center
	start := center - int64(n/2)

	capNoise := e.blockTemporallySteady(start, n)

	for c := 0; c < e.channels; c++ {
		e.windowBlock(c, start, n, ln, rn)
		e.fwd[slot].forward(e.wbuf[:n], e.spec[c][:n2])
		floor1Fit(fl, e.spec[c][:n2], e.targets[:postCount], n2)
		floor1EncodeVals(fl, e.targets[:postCount], e.vals[c][:postCount], e.fFinal[:postCount])
		fl.curve(e.vals[c][:postCount], e.curve[c][:n2], e.fFinal[:postCount], e.fStep2[:postCount], n2)
		normalizeResidue(e.spec[c][:n2], e.curve[c][:n2], e.resid[c][:n2], n2)

		block := e.blockSamples(c, start, n)
		psyRes, err := e.psy[slot][c].Analyze(block)
		if err != nil {
			return err
		}
		lineThresholds(psyRes, e.spec[c][:n2], e.thrLine[:n2], n2)
		classifyPartitions(e.spec[c][:n2], e.curve[c][:n2], e.thrLine[:n2], e.classes[c][:n2/resPartSize], resPartSize, n2, capNoise)
		maskResidue(e.spec[c][:n2], e.resid[c][:n2], e.thrLine[:n2], n2)
	}

	magChannel := -1
	if len(e.cfg.mappings[slot].couplingMag) > 0 {
		coupleResidues(e.resid, n2)
		deriveCoupledClasses(e.classes[0][:n2/resPartSize], e.classes[1][:n2/resPartSize], e.resid[1][:n2], n2)
		magChannel = e.cfg.mappings[slot].couplingMag[0]
	}

	e.w.reset()
	e.w.writeBit(0)
	e.w.writeBits(uint(e.cfg.modeBits), uint32(modeForSlot(slot)))
	if slot == slotLong {
		e.w.writeBit(boolBit(prevN == e.long))
		e.w.writeBit(boolBit(nextN == e.long))
	}
	for c := 0; c < e.channels; c++ {
		writeFloorData(&e.w, fl, e.vals[c][:postCount], e.cfg.books[bookFloorPosts])
	}
	encodeResidueType1(&e.w, res, e.cfg.books, e.resid, e.classes, n2, magChannel)

	l := int64((prevN + n) / 4)
	out := l
	pts := e.decoded
	if e.firstBlock {
		out = 0
		e.firstBlock = false
	}
	e.decoded += out
	return emit(codec.Packet{Data: e.w.bytes(), PTS: pts, Dur: out, Sync: true})
}

func neighbourWin(n, other int) int {
	if other < n {
		return other
	}
	return n
}

func (e *Encoder) windowBlock(c int, start int64, n, ln, rn int) {
	leftWin := getPlan(ln).window
	rightWin := getPlan(rn).window
	leftBegin := n/4 - ln/4
	leftEnd := leftBegin + ln/2
	rightBegin := 3*n/4 - rn/4
	rightEnd := rightBegin + rn/2
	off := int(start - e.bufBase)
	src := e.buf[c]
	for i := 0; i < leftBegin; i++ {
		e.wbuf[i] = 0
	}
	for i := leftBegin; i < leftEnd; i++ {
		e.wbuf[i] = e.at(src, off+i) * leftWin[i-leftBegin]
	}
	for i := leftEnd; i < rightBegin; i++ {
		e.wbuf[i] = e.at(src, off+i)
	}
	for i := rightBegin; i < rightEnd; i++ {
		e.wbuf[i] = e.at(src, off+i) * rightWin[rightEnd-1-i]
	}
	for i := rightEnd; i < n; i++ {
		e.wbuf[i] = 0
	}
}

func (e *Encoder) blockSamples(c int, start int64, n int) []float32 {
	off := int(start - e.bufBase)
	src := e.buf[c]
	if cap(e.psyIn) < n {
		e.psyIn = make([]float32, n)
	}
	e.psyIn = e.psyIn[:n]
	for i := 0; i < n; i++ {
		e.psyIn[i] = e.at(src, off+i)
	}
	return e.psyIn
}

func (e *Encoder) blockTemporallySteady(start int64, n int) bool {
	const subs = 16
	sw := n / subs
	if sw == 0 {
		return true
	}
	var maxE, sumE float64
	for s := 0; s < subs; s++ {
		off := int(start-e.bufBase) + s*sw
		var eSub float64
		for i := 0; i < sw; i++ {
			var v float32
			for c := 0; c < e.channels; c++ {
				v += e.at(e.buf[c], off+i)
			}
			eSub += float64(v) * float64(v)
		}
		sumE += eSub
		if eSub > maxE {
			maxE = eSub
		}
	}
	if sumE <= 0 {
		return true
	}
	return maxE/(sumE/subs) < steadyPeakRatio
}

const steadyPeakRatio = 4.0

func (e *Encoder) at(src []float32, i int) float32 {
	if i < 0 || i >= len(src) {
		return 0
	}
	return src[i]
}

func (e *Encoder) scanTransients(flush bool) {
	for e.scanPos+int64(e.short) <= e.bufEnd() {
		start := e.scanPos
		off := int(start - e.bufBase)
		for i := 0; i < e.short; i++ {
			var v float32
			for c := 0; c < e.channels; c++ {
				v += e.at(e.buf[c], off+i)
			}
			e.monobuf[i] = v / float32(e.channels)
		}
		if attack, pos := e.attack.Scan(e.monobuf, attackSubWindows); attack {
			e.attackPos = append(e.attackPos, start+int64(pos*(e.short/attackSubWindows)))
		}
		e.scanPos += int64(e.short)
	}
	if flush {
		e.scanPos = e.bufEnd()
	}
}

func (e *Encoder) attackBetween(lo, hi int64) bool {
	for _, p := range e.attackPos {
		if p >= lo && p < hi {
			return true
		}
	}
	return false
}

// Finish pads and emits enough trailing blocks that the decoder outputs every real sample, then reports the gapless trailer.
func (e *Encoder) Finish(emit func(codec.Packet) error) (codec.Trailer, error) {
	if e.inSamples == 0 {
		return codec.Trailer{}, nil
	}
	if err := e.drain(emit); err != nil {
		return codec.Trailer{}, err
	}
	e.scanTransients(true)
	for e.decoded < e.inSamples {
		for len(e.pending) < 2 {
			e.planNext(true)
		}
		e.ensureBuffered(e.pending[0].center + int64(e.long))
		if err := e.emitPending(emit, e.pending[1].size, false); err != nil {
			return codec.Trailer{}, err
		}
	}
	padding := e.decoded - e.inSamples
	if padding < 0 {
		padding = 0
	}
	return codec.Trailer{Samples: e.inSamples, Delay: 0, Padding: padding}, nil
}

func (e *Encoder) ensureBuffered(end int64) {
	if have := e.bufEnd(); have < end {
		pad := int(end - have)
		for c := 0; c < e.channels; c++ {
			e.buf[c] = append(e.buf[c], make([]float32, pad)...)
		}
	}
}

const (
	attackRatio      = 4.0
	attackSubWindows = 4
	shortRunBlocks   = 12
)
