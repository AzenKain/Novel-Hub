package mp3

import (
	"fmt"
	"math"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/dsp/psy"
	"novelhub/pkg/waxflow/waxerr"
)

var _ codec.Encoder = (*Encoder)(nil)

// EncoderVersion is the encoder's cache-key version constant (ADR-0004): bump on any change that alters the encoded bitstream.
const EncoderVersion = "mp3-enc-2+" + psy.Version

const thrCalib = 1.0e-5

const psyOffsetDB = 0.0

// EncoderOptions configures a Layer III encoder.
type EncoderOptions struct {
	Bitrate int
	VBR     bool
}

// DefaultBitrate is the CBR bit rate used when EncoderOptions leaves it zero.
const DefaultBitrate = 128000

const psyHistLen = 1024 - 576

// Encoder is an MPEG-1/2/2.5 Layer III encoder (CBR or VBR).
type Encoder struct {
	fmt      audio.Format
	version  MPEGVersion
	rateIdx  int
	bitrate  int
	vbr      bool
	channels int
	row      int
	granules int
	siLen    int
	resCap   int
	cutLine  int

	ana [2]analyzer
	buf [2][]float32
	xr  [2][2][576]float32

	psy     [2]*psy.Model
	psyHist [2][psyHistLen]float32
	psyBuf  []float32
	thr     [2][2][nSfBands]float64
	bandE   [2][2][nSfBands]float64
	avgPE   float64

	inSamples  int64
	outSamples int64

	padAcc, padStep, padThresh int

	frames   []pendingFrame
	physEnd  int
	writePos int

	sw bitWriter
	mw bitWriter
}

type pendingFrame struct {
	hdr   [4]byte
	si    []byte
	main  []byte
	start int
	spf   int
}

func legalRate(rate int) (MPEGVersion, int, bool) {
	for _, r := range []struct {
		v  MPEGVersion
		hz [3]int
	}{
		{MPEG1, [3]int{44100, 48000, 32000}},
		{MPEG2, [3]int{22050, 24000, 16000}},
		{MPEG25, [3]int{11025, 12000, 8000}},
	} {
		for idx, hz := range r.hz {
			if hz == rate {
				return r.v, idx, true
			}
		}
	}
	return 0, 0, false
}

func clampBitrate(v MPEGVersion, kbps int) int {
	lsf := 0
	if v != MPEG1 {
		lsf = 1
	}
	best := 0
	for i := 1; i < 15; i++ {
		if r := bitrateKbps[lsf][i]; r <= kbps && r > best {
			best = r
		}
	}
	if best == 0 {
		best = bitrateKbps[lsf][1]
	}
	return best
}

// NewEncoder returns an encoder for the given input format.
func NewEncoder(f audio.Format, opts *EncoderOptions) (*Encoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.Channels < 1 || f.Channels > 2 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp3: input %v is not a Layer III encode shape (float32, 1-2 ch)", f))
	}
	ver, rateIdx, ok := legalRate(f.Rate)
	if !ok {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp3: sample rate %d Hz is not an MPEG Layer III rate", f.Rate))
	}
	bitrate := DefaultBitrate
	vbr := false
	if opts != nil {
		if opts.Bitrate != 0 {
			bitrate = opts.Bitrate
		}
		vbr = opts.VBR
	}
	if bitrate <= 0 || bitrate%1000 != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("mp3: bit rate %d must be a positive whole number of kbit/s", bitrate))
	}
	bitrate = clampBitrate(ver, bitrate/1000) * 1000

	e := &Encoder{
		fmt:      f,
		version:  ver,
		rateIdx:  rateIdx,
		bitrate:  bitrate,
		vbr:      vbr,
		channels: f.Channels,
	}
	e.granules = 1
	e.resCap = 255
	if ver == MPEG1 {
		e.granules = 2
		e.resCap = 511
	}
	h := e.header(false, false)
	e.row = h.rateRow()
	e.siLen = h.SideInfoLen()
	e.padStep = ((h.SamplesPerFrame() / 8) * e.bitrate) % h.Rate
	e.padThresh = h.Rate

	cutoff := 3000.0 + float64(bitrate)/float64(f.Channels)/5
	cutoff = math.Min(cutoff, 0.94*float64(h.Rate)/2)
	lineHz := float64(h.Rate) / (2 * 576)
	e.cutLine = min(int(cutoff/lineHz), 576)

	offsetDB := psyOffsetDB
	if vbr {
		offsetDB = math.Min(math.Max(10*math.Log2(float64(bitrate)/128000), -18), 12)
	}
	offs := make([]int, len(sfbEdgesLong[e.row]))
	for i, v := range sfbEdgesLong[e.row] {
		offs[i] = v
	}
	for c := 0; c < f.Channels; c++ {
		m, err := psy.New(psy.Config{
			Rate: h.Rate, Lines: 576, FFTSize: 1024,
			BandOffsets: offs, OffsetDB: offsetDB,
		})
		if err != nil {
			return nil, err
		}
		e.psy[c] = m
	}
	e.psyBuf = make([]float32, psyHistLen+576*e.granules)
	return e, nil
}

func (e *Encoder) header(pad, ms bool) Header {
	rate := rateHz[e.rateIdx]
	if e.version != MPEG1 {
		rate >>= 1
	}
	if e.version == MPEG25 {
		rate >>= 1
	}
	mode := ModeStereo
	modeExt := 0
	if e.channels == 1 {
		mode = ModeMono
	} else if ms {
		mode = ModeJoint
		modeExt = 2
	}
	return Header{
		rateIdx:  e.rateIdx,
		Version:  e.version,
		Rate:     rate,
		Channels: e.channels,
		Mode:     mode,
		ModeExt:  modeExt,
		Bitrate:  e.bitrate,
		Padding:  pad,
	}
}

// InputFormat is the PCM format the encoder consumes.
func (e *Encoder) InputFormat() audio.Format { return e.fmt }

// Bitrate is the actual constant bit rate in bits per second, after the requested rate is clamped to one the layer supports.
func (e *Encoder) Bitrate() int {
	if e.vbr {
		return 0
	}
	return e.bitrate
}

// FrameSize is the encoder-native chunk in frames: one whole MP3 frame.
func (e *Encoder) FrameSize() int { return 576 * e.granules }

// CodecConfig is nil: MP3 is self-framing and carries no out-of-band setup.
func (e *Encoder) CodecConfig() []byte { return nil }

const maxSample = 8.0

// Encode buffers src and emits every whole frame that becomes available.
func (e *Encoder) Encode(src *audio.Buffer, emit func(codec.Packet) error) error {
	if src.Fmt != e.fmt {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp3: encode input %v disagrees with %v", src.Fmt, e.fmt))
	}
	for ch := 0; ch < e.channels; ch++ {
		e.buf[ch] = appendSanitized(e.buf[ch], src.ChanF(ch)[:src.N])
	}
	e.inSamples += int64(src.N)
	return e.drainFrames(emit)
}

func appendSanitized(dst, src []float32) []float32 {
	for _, v := range src {
		switch {
		case math.IsNaN(float64(v)) || math.IsInf(float64(v), 0):
			v = 0
		case v > maxSample:
			v = maxSample
		case v < -maxSample:
			v = -maxSample
		}
		dst = append(dst, v)
	}
	return dst
}

func (e *Encoder) drainFrames(emit func(codec.Packet) error) error {
	fs := e.FrameSize()
	for len(e.buf[0]) >= fs {
		if err := e.encodeFrame(fs, emit); err != nil {
			return err
		}
		for ch := 0; ch < e.channels; ch++ {
			e.buf[ch] = append(e.buf[ch][:0], e.buf[ch][fs:]...)
		}
	}
	return nil
}

// Finish pads the tail to a frame, flushes the filterbank latency with silent frames so every real sample reaches the output, drains the reservoir queue, and reports the gapless trailer.
func (e *Encoder) Finish(emit func(codec.Packet) error) (codec.Trailer, error) {
	fs := e.FrameSize()
	if n := len(e.buf[0]); n > 0 {
		for ch := 0; ch < e.channels; ch++ {
			e.buf[ch] = append(e.buf[ch], make([]float32, fs-n)...)
		}
		if err := e.encodeFrame(fs, emit); err != nil {
			return codec.Trailer{}, err
		}
		for ch := 0; ch < e.channels; ch++ {
			e.buf[ch] = e.buf[ch][:0]
		}
	}
	for i := 0; i < flushFrames; i++ {
		for ch := 0; ch < e.channels; ch++ {
			e.buf[ch] = append(e.buf[ch][:0], make([]float32, fs)...)
		}
		if err := e.encodeFrame(fs, emit); err != nil {
			return codec.Trailer{}, err
		}
	}
	for ch := 0; ch < e.channels; ch++ {
		e.buf[ch] = e.buf[ch][:0]
	}
	if err := e.flushQueue(emit); err != nil {
		return codec.Trailer{}, err
	}

	delay := int64(EncoderDelay)
	padding := e.outSamples - e.inSamples - delay
	if padding < 0 {
		padding = 0
	}
	return codec.Trailer{Samples: e.inSamples, Delay: delay, Padding: padding}, nil
}

// EncoderDelay is the encoder's intrinsic priming in samples: the leading output samples that precede the first real input sample, carried in the LAME tag so decoders trim them.
const EncoderDelay = 1057 - 529

const flushFrames = 2

// Delay reports the encoder's gapless delay (EncoderDelay) for the muxer's LAME tag.
func (e *Encoder) Delay() int { return EncoderDelay }

// FramesFor returns the number of MP3 frames the encoder emits for n input samples at the given sample rate: the whole and padded-tail frames plus the flush.
func FramesFor(n int64, rate int) int {
	fs := int64(1152)
	if v, _, ok := legalRate(rate); ok && v != MPEG1 {
		fs = 576
	}
	return int((n+fs-1)/fs) + flushFrames
}

func (e *Encoder) encodeFrame(fs int, emit func(codec.Packet) error) error {
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			e.ana[ch].granuleMDCT(e.buf[ch][gr*576:gr*576+576], &e.xr[gr][ch])
			for i := e.cutLine; i < 576; i++ {
				e.xr[gr][ch][i] = 0
			}
		}
	}

	pe := 0.0
	for ch := 0; ch < e.channels; ch++ {
		copy(e.psyBuf[:psyHistLen], e.psyHist[ch][:])
		copy(e.psyBuf[psyHistLen:], e.buf[ch][:fs])
		copy(e.psyHist[ch][:], e.buf[ch][fs-psyHistLen:fs])
		chPE := 0.0
		for gr := 0; gr < e.granules; gr++ {
			res, err := e.psy[ch].Analyze(e.psyBuf[gr*576 : gr*576+1024])
			if err != nil {
				return err
			}
			for b := 0; b < nSfBands; b++ {
				e.thr[gr][ch][b] = res.Thr[b] * thrCalib
			}
			chPE += res.PE
		}
		pe = math.Max(pe, chPE)
	}

	edges := &sfbEdgesLong[e.row]
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			for b := 0; b < nSfBands; b++ {
				s := 0.0
				for i := edges[b]; i < edges[b+1]; i++ {
					v := float64(e.xr[gr][ch][i])
					s += v * v
				}
				e.bandE[gr][ch][b] = s
			}
		}
	}

	ms := false
	if e.channels == 2 {
		ms = e.decideMS()
	}

	res := e.physEnd - e.writePos
	if res > e.resCap {
		e.writePos += res - e.resCap
		res = e.resCap
	}
	mdb := res

	var q [2][2]gcQuant
	var h Header
	var slots int
	if e.vbr {
		e.quantizeVBR(&q, res)
	} else {
		pad := e.nextPadding()
		h = e.header(pad, ms)
		slots = h.Size() - HeaderLen - e.siLen
		e.quantizeCBR(&q, slots, res, pe)
	}

	e.mw.reset()
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			e.writeGranuleData(&q[gr][ch])
		}
	}
	e.mw.align()
	main := e.mw.buf

	if e.vbr {
		h, slots = e.vbrFrameFor(len(main)-res, ms)
	}

	si := e.writeSideInfo(mdb, &q)

	f := pendingFrame{hdr: headerBytes(h), si: si, main: make([]byte, slots), start: e.physEnd, spf: h.SamplesPerFrame()}
	e.frames = append(e.frames, f)
	e.physEnd += slots
	e.outSamples += int64(h.SamplesPerFrame())

	e.writeLogical(main)

	return e.emitReady(emit)
}

func (e *Encoder) quantizeCBR(q *[2][2]gcQuant, slots, res int, pe float64) {
	availBits := slots*8 + res*8
	if e.avgPE == 0 {
		e.avgPE = pe
	}
	e.avgPE = 0.95*e.avgPE + 0.05*pe
	difficulty := 1.0
	if e.avgPE > 0 {
		difficulty = math.Min(math.Max(pe/e.avgPE, 0.7), 1.4)
	}
	target := int(float64(slots*8) * difficulty)
	minSpend := slots*8 + res*8 - e.resCap*8
	target = max(target, minSpend, 0)
	target = min(target, availBits)

	e.splitAndQuantize(q, target)
}

const (
	vbrTightness = 1.1
	vbrFloorBits = 150
)

func (e *Encoder) quantizeVBR(q *[2][2]gcQuant, res int) {
	_, maxSlots := e.vbrFrameFor(1<<30, false)
	capacity := (maxSlots+res)*8 - 8

	nGC := e.granules * e.channels
	var budgets [4]int
	total := 0
	i := 0
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			d := e.gcDemand(gr, ch)
			budgets[i] = min(int(d*vbrTightness)+vbrFloorBits, part23Max)
			total += budgets[i]
			i++
		}
	}
	if total > capacity {
		for i := range budgets[:nGC] {
			budgets[i] = budgets[i] * capacity / total
		}
	}
	i = 0
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			q[gr][ch] = quantizeGranule(quantIn{
				xr: &e.xr[gr][ch], row: e.row,
				thr: &e.thr[gr][ch], mpeg1: e.version == MPEG1,
			}, budgets[i])
			i++
		}
	}
}

func (e *Encoder) splitAndQuantize(q *[2][2]gcQuant, target int) {
	nGC := e.granules * e.channels
	var demand [4]float64
	total := 0.0
	i := 0
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			demand[i] = e.gcDemand(gr, ch)
			total += demand[i]
			i++
		}
	}
	floor := target / (nGC * 4)
	spent := 0
	i = 0
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			var budget int
			if i == nGC-1 {
				budget = target - spent
			} else if total > 0 {
				budget = int(float64(target) * demand[i] / total)
			} else {
				budget = target / nGC
			}
			budget = max(budget, floor)
			budget = min(budget, target-spent)
			budget = max(budget, 0)
			q[gr][ch] = quantizeGranule(quantIn{
				xr: &e.xr[gr][ch], row: e.row,
				thr: &e.thr[gr][ch], mpeg1: e.version == MPEG1,
			}, budget)
			spent += budget
			i++
		}
	}
}

func (e *Encoder) gcDemand(gr, ch int) float64 {
	edges := &sfbEdgesLong[e.row]
	d := 0.0
	for b := 0; b < nSfBands; b++ {
		en, thr := e.bandE[gr][ch][b], e.thr[gr][ch][b]
		if thr > 0 && en > thr {
			d += float64(edges[b+1]-edges[b]) * math.Log2(en/thr)
		}
	}
	return d
}

func (e *Encoder) vbrFrameFor(need int, ms bool) (Header, int) {
	lsf := 0
	if e.version != MPEG1 {
		lsf = 1
	}
	h := e.header(false, ms)
	var slots int
	for i := 1; i < 15; i++ {
		h.Bitrate = bitrateKbps[lsf][i] * 1000
		slots = h.Size() - HeaderLen - e.siLen
		if slots >= need {
			break
		}
	}
	return h, slots
}

func (e *Encoder) decideMS() bool {
	const invSqrt2 = 0.7071067811865476
	edges := &sfbEdgesLong[e.row]
	var eM, eS [2][nSfBands]float64
	demLR, demMS := 0.0, 0.0
	for gr := 0; gr < e.granules; gr++ {
		l, r := &e.xr[gr][0], &e.xr[gr][1]
		for b := 0; b < nSfBands; b++ {
			var em, es float64
			for i := edges[b]; i < edges[b+1]; i++ {
				lv, rv := float64(l[i]), float64(r[i])
				m := (lv + rv) * invSqrt2
				s := (lv - rv) * invSqrt2
				em += m * m
				es += s * s
			}
			eM[gr][b], eS[gr][b] = em, es
			w := float64(edges[b+1] - edges[b])
			thrL, thrR := e.thr[gr][0][b], e.thr[gr][1][b]
			thrMS := math.Min(thrL, thrR)
			demLR += demandOf(e.bandE[gr][0][b], thrL, w) + demandOf(e.bandE[gr][1][b], thrR, w)
			demMS += demandOf(em, thrMS, w) + demandOf(es, thrMS, w)
		}
	}
	if demMS >= demLR {
		return false
	}
	for gr := 0; gr < e.granules; gr++ {
		l, r := &e.xr[gr][0], &e.xr[gr][1]
		for i := 0; i < 576; i++ {
			lv, rv := float64(l[i]), float64(r[i])
			l[i] = float32((lv + rv) * invSqrt2)
			r[i] = float32((lv - rv) * invSqrt2)
		}
		for b := 0; b < nSfBands; b++ {
			thrMS := math.Min(e.thr[gr][0][b], e.thr[gr][1][b])
			e.thr[gr][0][b] = thrMS
			e.thr[gr][1][b] = thrMS
			e.bandE[gr][0][b] = eM[gr][b]
			e.bandE[gr][1][b] = eS[gr][b]
		}
	}
	return true
}

func demandOf(energy, thr, width float64) float64 {
	if thr <= 0 || energy <= thr {
		return 0
	}
	return width * math.Log2(energy/thr)
}

func (e *Encoder) writeGranuleData(q *gcQuant) {
	band := 0
	for p, cnt := range sfPartCount {
		bits := uint(q.slen[p])
		for i := 0; i < cnt; i++ {
			if bits > 0 {
				e.mw.writeBits(bits, uint32(q.sfTx[band]))
			}
			band++
		}
	}
	bigEnd := q.bigValues * 2
	for i := 0; i+1 < bigEnd; i += 2 {
		t := q.table[0]
		if i >= q.region1End {
			t = q.table[2]
		} else if i >= q.region0End {
			t = q.table[1]
		}
		e.mw.writePair(t, q.ix[i], q.ix[i+1])
	}
	for i := bigEnd; i+3 < bigEnd+q.count1*4 && i+3 < 576; i += 4 {
		e.mw.writeQuad(q.count1Table, q.ix[i], q.ix[i+1], q.ix[i+2], q.ix[i+3])
	}
}

func (e *Encoder) writeSideInfo(mdb int, q *[2][2]gcQuant) []byte {
	w := &e.sw
	w.reset()
	if e.version == MPEG1 {
		w.writeBits(9, uint32(mdb))
		if e.channels == 1 {
			w.writeBits(5, 0)
		} else {
			w.writeBits(3, 0)
		}
		for ch := 0; ch < e.channels; ch++ {
			w.writeBits(4, 0)
		}
	} else {
		w.writeBits(8, uint32(mdb))
		w.writeBits(uint(e.channels), 0)
	}
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			e.writeGranuleSide(w, &q[gr][ch])
		}
	}
	w.align()
	out := make([]byte, len(w.buf))
	copy(out, w.buf)
	return out
}

func (e *Encoder) writeGranuleSide(w *bitWriter, q *gcQuant) {
	w.writeBits(12, uint32(q.part23))
	w.writeBits(9, uint32(q.bigValues))
	w.writeBits(8, uint32(q.globalGain))
	if e.version == MPEG1 {
		w.writeBits(4, uint32(q.scfCompress))
	} else {
		w.writeBits(9, uint32(q.scfCompress))
	}
	w.writeBits(1, 0)
	w.writeBits(5, uint32(q.table[0]))
	w.writeBits(5, uint32(q.table[1]))
	w.writeBits(5, uint32(q.table[2]))
	w.writeBits(4, uint32(q.region0Count))
	w.writeBits(3, uint32(q.region1Count))
	if e.version == MPEG1 {
		pre := uint32(0)
		if q.preflag {
			pre = 1
		}
		w.writeBits(1, pre)
	}
	w.writeBits(1, uint32(q.scfScale))
	w.writeBits(1, uint32(q.count1Table))
}

func (e *Encoder) writeLogical(main []byte) {
	pos, src := e.writePos, 0
	for i := range e.frames {
		if src >= len(main) {
			break
		}
		f := &e.frames[i]
		if pos >= f.start+len(f.main) {
			continue
		}
		n := copy(f.main[pos-f.start:], main[src:])
		pos += n
		src += n
	}
	e.writePos = pos
}

func (e *Encoder) emitReady(emit func(codec.Packet) error) error {
	for len(e.frames) > 0 {
		f := &e.frames[0]
		if f.start+len(f.main) > e.writePos {
			break
		}
		if err := e.emit(f, emit); err != nil {
			return err
		}
		e.frames = e.frames[1:]
	}
	return nil
}

func (e *Encoder) flushQueue(emit func(codec.Packet) error) error {
	for i := range e.frames {
		if err := e.emit(&e.frames[i], emit); err != nil {
			return err
		}
	}
	e.frames = e.frames[:0]
	return nil
}

func (e *Encoder) emit(f *pendingFrame, emit func(codec.Packet) error) error {
	pkt := make([]byte, 0, HeaderLen+len(f.si)+len(f.main))
	pkt = append(pkt, f.hdr[:]...)
	pkt = append(pkt, f.si...)
	pkt = append(pkt, f.main...)
	return emit(codec.Packet{Data: pkt, Dur: int64(f.spf), Sync: true})
}

func (e *Encoder) nextPadding() bool {
	e.padAcc += e.padStep
	if e.padAcc >= e.padThresh {
		e.padAcc -= e.padThresh
		return true
	}
	return false
}

func headerBytes(h Header) [4]byte {
	var b [4]byte
	b[0] = 0xFF
	verBits := byte(3)
	switch h.Version {
	case MPEG2:
		verBits = 2
	case MPEG25:
		verBits = 0
	}
	b[1] = 0xE0 | verBits<<3 | 1<<1 | 1
	lsf := 0
	if h.Version != MPEG1 {
		lsf = 1
	}
	bi := 0
	for i := 1; i < 15; i++ {
		if bitrateKbps[lsf][i]*1000 == h.Bitrate {
			bi = i
			break
		}
	}
	pad := byte(0)
	if h.Padding {
		pad = 1
	}
	b[2] = byte(bi)<<4 | byte(h.rateIdx)<<2 | pad<<1
	b[3] = byte(h.Mode)<<6 | byte(h.ModeExt)<<4
	return b
}
