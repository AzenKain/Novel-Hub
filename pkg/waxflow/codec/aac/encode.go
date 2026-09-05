package aac

import (
	"fmt"
	"math"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/dsp/psy"
	"novelhub/pkg/waxflow/waxerr"
)

var _ codec.Encoder = (*Encoder)(nil)

// EncoderVersion identifies the encode algorithm revision for cache keys (ADR-0004).
const EncoderVersion = "aac-enc-1+" + psy.Version

// EncoderDelay is the codec priming in output samples: one frame of zeros ahead of the first real sample, so frame 0's MDCT window (which reaches one frame into the past) sees defined history.
const EncoderDelay = 1024

const frameLen = 1024

// DefaultBitrate is used when EncoderOptions.Bitrate is zero.
const DefaultBitrate = 128000

const thrCalib = (8.0 / 3.0) * 32768 * 32768

const psyOffsetDB = 0.0

// EncoderOptions configures NewEncoder.
type EncoderOptions struct {
	Bitrate int
}

// Encoder is an AAC-LC encoder producing raw access units (one packet per 1024-sample frame).
type Encoder struct {
	fmt      audio.Format
	channels int
	rate     int
	rateIdx  int
	bitrate  int
	asc      [2]byte

	swbLong     []uint16
	swbShort    []uint16
	numSwbLong  int
	numSwbShort int
	maxSfbLong  int
	maxSfbShort int

	pending   [2][]float32
	hist      [2][3 * frameLen]float32
	inSamples int64
	outFrames int64

	det        [2]*psy.AttackDetector
	attackPrev [2]attackInfo
	attackCur  [2]attackInfo
	prevSeq    int

	psyLong  [2]*psy.Model
	psyShort [2]*psy.Model

	meanBits  float64
	reservoir float64
	avgPE     float64

	spec  [2][1024]float64
	cq    [2]chanQuant
	tns   [2]tnsEnc
	thr   [2][maxWindowGroups][maxSFBCount]float64
	msUse [maxWindowGroups][maxSFBCount]bool
	w     bitWriter
}

type attackInfo struct {
	attack bool
	pos    int
}

// NewEncoder returns an Encoder for f, which must be float32 with 1 or 2 channels at one of the 13 AAC sampling rates.
func NewEncoder(f audio.Format, opts *EncoderOptions) (*Encoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.BitDepth != 32 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "aac: encoder input must be float32")
	}
	if f.Channels < 1 || f.Channels > 2 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("aac: %d channels unsupported (mono or stereo)", f.Channels))
	}
	rateIdx := samplingIndex(f.Rate)
	if rateIdx < 0 || rateIdx >= len(swbOffsetLong) {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("aac: sample rate %d is not an AAC rate", f.Rate))
	}

	var o EncoderOptions
	if opts != nil {
		o = *opts
	}
	if o.Bitrate == 0 {
		o.Bitrate = DefaultBitrate
	}
	minRate := 8000 * f.Channels
	maxRate := 6 * f.Rate * f.Channels
	bitrate := min(max(o.Bitrate, minRate), maxRate)

	e := &Encoder{
		fmt:      f,
		channels: f.Channels,
		rate:     f.Rate,
		rateIdx:  rateIdx,
		bitrate:  bitrate,
		prevSeq:  onlyLong,
	}
	e.asc[0] = byte(aotAACLC<<3 | rateIdx>>1)
	e.asc[1] = byte(rateIdx<<7 | f.Channels<<3)

	e.swbLong = swbOffsetLong[rateIdx]
	e.swbShort = swbOffsetShort[rateIdx]
	e.numSwbLong = swbCountLong(rateIdx)
	e.numSwbShort = swbCountShort(rateIdx)

	cutoff := 3000.0 + float64(bitrate)/float64(f.Channels)/5
	cutoff = math.Min(cutoff, 0.94*float64(f.Rate)/2)
	e.maxSfbLong = coveringSfb(e.swbLong, e.numSwbLong, cutoff, f.Rate, 2048)
	e.maxSfbShort = coveringSfb(e.swbShort, e.numSwbShort, cutoff, f.Rate, 256)

	longBands := make([]int, e.numSwbLong+1)
	for i := range longBands {
		longBands[i] = int(e.swbLong[i])
	}
	shortBands := make([]int, e.numSwbShort+1)
	for i := range shortBands {
		shortBands[i] = int(e.swbShort[i])
	}
	for c := 0; c < f.Channels; c++ {
		var err error
		e.psyLong[c], err = psy.New(psy.Config{
			Rate: f.Rate, Lines: 1024, FFTSize: 2048,
			BandOffsets: longBands, OffsetDB: psyOffsetDB,
		})
		if err != nil {
			return nil, err
		}
		e.psyShort[c], err = psy.New(psy.Config{
			Rate: f.Rate, Lines: 128, FFTSize: 256,
			BandOffsets: shortBands, NoPredict: true, FixedC: 0.4,
			OffsetDB: psyOffsetDB,
		})
		if err != nil {
			return nil, err
		}
		e.det[c] = psy.NewAttackDetector(0)
	}

	e.meanBits = float64(bitrate) * frameLen / float64(f.Rate)
	e.avgPE = e.meanBits * 0.4
	return e, nil
}

func coveringSfb(swb []uint16, numSwb int, cutoff float64, rate, n int) int {
	lineHz := float64(rate) / float64(n)
	for sfb := 1; sfb <= numSwb; sfb++ {
		if float64(swb[sfb])*lineHz >= cutoff {
			return sfb
		}
	}
	return numSwb
}

// InputFormat implements codec.Encoder.
func (e *Encoder) InputFormat() audio.Format { return e.fmt }

// FrameSize implements codec.Encoder: 1024 samples per frame.
func (e *Encoder) FrameSize() int { return frameLen }

// Bitrate reports the clamped target bit rate the plan advertises.
func (e *Encoder) Bitrate() int { return e.bitrate }

// Delay reports the encoder priming in output samples.
func (e *Encoder) Delay() int { return EncoderDelay }

// CodecConfig returns the two-byte AudioSpecificConfig (AAC-LC, this stream's rate index and channel configuration).
func (e *Encoder) CodecConfig() []byte { return e.asc[:] }

const maxSample = 8.0

const auCeilingSlack = 256

// Encode buffers src and emits an access unit for every whole source block that becomes available.
func (e *Encoder) Encode(src *audio.Buffer, emit func(codec.Packet) error) error {
	if src.Fmt != e.fmt {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("aac: encode input %v disagrees with %v", src.Fmt, e.fmt))
	}
	for c := 0; c < e.channels; c++ {
		e.pending[c] = appendSanitized(e.pending[c], src.ChanF(c)[:src.N])
	}
	e.inSamples += int64(src.N)
	for len(e.pending[0]) >= frameLen {
		if err := e.pushBlock(emit); err != nil {
			return err
		}
	}
	return nil
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

func (e *Encoder) pushBlock(emit func(codec.Packet) error) error {
	for c := 0; c < e.channels; c++ {
		h := &e.hist[c]
		copy(h[:2*frameLen], h[frameLen:])
		copy(h[2*frameLen:], e.pending[c][:frameLen])
		e.pending[c] = append(e.pending[c][:0], e.pending[c][frameLen:]...)
		e.attackPrev[c] = e.attackCur[c]
		a, pos := e.det[c].Scan(h[2*frameLen:], 8)
		e.attackCur[c] = attackInfo{attack: a, pos: pos}
	}
	return e.encodeFrame(emit)
}

func (e *Encoder) windowSeq(shortNow, shortNext bool) int {
	switch {
	case shortNow:
		return eightShort
	case e.prevSeq == eightShort && shortNext:
		return eightShort
	case e.prevSeq == eightShort:
		return longStop
	case shortNext:
		return longStart
	default:
		return onlyLong
	}
}

func grouping(pos int) []int {
	win := pos - 3
	if win < 0 {
		win = 0
	}
	if win > 7 {
		win = 7
	}
	switch {
	case win == 0:
		return []int{1, 7}
	case win == 7:
		return []int{7, 1}
	default:
		return []int{win, 1, 7 - win}
	}
}

var longGroup = []int{1}

func (e *Encoder) encodeFrame(emit func(codec.Packet) error) error {
	shortNow := false
	shortNext := false
	attackPos := 0
	for c := 0; c < e.channels; c++ {
		if e.attackPrev[c].attack {
			if !shortNow {
				attackPos = e.attackPrev[c].pos
			}
			shortNow = true
		}
		shortNext = shortNext || e.attackCur[c].attack
	}
	seq := e.windowSeq(shortNow, shortNext)

	groupLen := longGroup
	swb := e.swbLong
	maxSfb := e.maxSfbLong
	if seq == eightShort {
		groupLen = grouping(attackPos)
		swb = e.swbShort
		maxSfb = e.maxSfbShort
	}

	pe := 0.0
	for c := 0; c < e.channels; c++ {
		rl, err := e.psyLong[c].Analyze(e.hist[c][frameLen : frameLen+2048])
		if err != nil {
			return err
		}
		pe = math.Max(pe, rl.PE)
		if seq != eightShort {
			for sfb := 0; sfb < e.numSwbLong; sfb++ {
				e.thr[c][0][sfb] = rl.Thr[sfb] * thrCalib
			}
			continue
		}
		var wThr [8][maxSFBCount]float64
		for i := 0; i < 8; i++ {
			off := frameLen + 448 + i*128
			rs, err := e.psyShort[c].Analyze(e.hist[c][off : off+256])
			if err != nil {
				return err
			}
			for sfb := 0; sfb < e.numSwbShort; sfb++ {
				wThr[i][sfb] = rs.Thr[sfb] * thrCalib
			}
		}
		win := 0
		for g, L := range groupLen {
			for sfb := 0; sfb < e.numSwbShort; sfb++ {
				t := 0.0
				for w := 0; w < L; w++ {
					t += wThr[win+w][sfb]
				}
				e.thr[c][g][sfb] = t
			}
			win += L
		}
	}

	for c := 0; c < e.channels; c++ {
		var tblk [2048]float64
		for i := range tblk {
			tblk[i] = float64(e.hist[c][frameLen+i]) * 32768
		}
		mdctFrame(&tblk, seq, &e.spec[c])
	}

	for c := 0; c < e.channels; c++ {
		e.tns[c] = tnsEnc{}
		if seq != eightShort {
			e.tns[c] = analyzeTNS(&e.spec[c], e.swbLong, e.numSwbLong, maxSfb, e.rateIdx, e.rate)
		}
	}

	msMask := 0
	if e.channels == 2 {
		msMask = e.decideMS(groupLen, swb, maxSfb)
	}

	for c := 0; c < e.channels; c++ {
		ch := c
		e.cq[c].buildBands(&e.spec[c], groupLen, swb, maxSfb,
			func(g, sfb int) float64 { return e.thr[ch][g][sfb] }, seq == eightShort)
	}

	e.avgPE = 0.95*e.avgPE + 0.05*pe
	difficulty := 1.0
	if e.avgPE > 0 {
		difficulty = min(max(pe/e.avgPE, 0.65), 1.7)
	}
	target := e.meanBits * difficulty
	target = math.Min(target, e.meanBits+math.Max(e.reservoir, 0)*0.5)
	target = math.Max(target, e.meanBits*0.3)
	target = math.Min(target, float64(6144*e.channels)*0.93)

	overhead := e.overheadBits(seq, maxSfb, msMask, len(groupLen))
	spectral := int(target) - overhead
	if spectral < 0 {
		spectral = 0
	}
	hard := 6144*e.channels - overhead - auCeilingSlack
	if hard < 0 {
		hard = 0
	}
	spectral = min(spectral, hard)

	frac := 0.5
	if e.channels == 2 {
		if dl, dr := e.cq[0].demand, e.cq[1].demand; dl+dr > 0 {
			frac = min(max(dl/(dl+dr), 0.2), 0.8)
		}
	}
	build := func(spectral, hard int) {
		if e.channels == 2 {
			lSpectral, lHard := int(float64(spectral)*frac), int(float64(hard)*frac)
			e.cq[0].quantizeChannel(lSpectral, lHard)
			e.cq[1].quantizeChannel(spectral-lSpectral, hard-lHard)
			e.w.reset()
			e.writeCPE(seq, groupLen, maxSfb, msMask)
		} else {
			e.cq[0].quantizeChannel(spectral, hard)
			e.w.reset()
			e.writeSCE(seq, groupLen, maxSfb)
		}
		e.w.writeBits(3, elEND)
		e.w.align()
	}
	build(spectral, hard)

	ceiling := 6144 * e.channels
	for pass := 1; e.w.bitLen() > ceiling && pass <= 2; pass++ {
		if pass == 1 {
			spectral = max(0, spectral-(e.w.bitLen()-ceiling)-auCeilingSlack)
		} else {
			spectral = 0
		}
		build(spectral, spectral)
	}

	e.reservoir += e.meanBits - float64(e.w.bitLen())
	if e.reservoir > float64(6144*e.channels) {
		e.reservoir = float64(6144 * e.channels)
	}
	if e.reservoir < -2*e.meanBits {
		e.reservoir = -2 * e.meanBits
	}

	e.prevSeq = seq
	e.outFrames++
	return emit(codec.Packet{Data: e.w.buf, PTS: (e.outFrames - 1) * frameLen, Dur: frameLen, Sync: true})
}

func (e *Encoder) decideMS(groupLen []int, swb []uint16, maxSfb int) int {
	all, none := true, true
	winBase := 0
	for g, L := range groupLen {
		for sfb := 0; sfb < maxSfb; sfb++ {
			lo, hi := int(swb[sfb]), int(swb[sfb+1])
			var eL, eR, eM, eS float64
			for w := 0; w < L; w++ {
				base := (winBase + w) * 128
				for k := lo; k < hi; k++ {
					l, r := e.spec[0][base+k], e.spec[1][base+k]
					eL += l * l
					eR += r * r
					m, s := (l+r)/2, (l-r)/2
					eM += m * m
					eS += s * s
				}
			}
			thrL, thrR := e.thr[0][g][sfb], e.thr[1][g][sfb]
			thrMS := math.Min(thrL, thrR) / 2
			w := float64(hi - lo)
			costLR := demandOf(eL, thrL, w) + demandOf(eR, thrR, w)
			costMS := demandOf(eM, thrMS, w) + demandOf(eS, thrMS, w)
			if costMS < costLR {
				e.msUse[g][sfb] = true
				none = false
			} else {
				e.msUse[g][sfb] = false
				all = false
			}
		}
		winBase += L
	}
	if none {
		return 0
	}
	winBase = 0
	for g, L := range groupLen {
		for sfb := 0; sfb < maxSfb; sfb++ {
			if !e.msUse[g][sfb] {
				continue
			}
			thrMS := math.Min(e.thr[0][g][sfb], e.thr[1][g][sfb]) / 2
			e.thr[0][g][sfb] = thrMS
			e.thr[1][g][sfb] = thrMS
			lo, hi := int(swb[sfb]), int(swb[sfb+1])
			for w := 0; w < L; w++ {
				base := (winBase + w) * 128
				for k := lo; k < hi; k++ {
					l, r := e.spec[0][base+k], e.spec[1][base+k]
					e.spec[0][base+k] = (l + r) / 2
					e.spec[1][base+k] = (l - r) / 2
				}
			}
		}
		winBase += L
	}
	if all {
		return 2
	}
	return 1
}

func demandOf(energy, thr, width float64) float64 {
	if thr <= 0 || energy <= thr {
		return 0
	}
	return width * math.Log2(energy/thr)
}

func (e *Encoder) overheadBits(seq, maxSfb, msMask, groups int) int {
	ics := 1 + 2 + 1
	if seq == eightShort {
		ics += 4 + 7
	} else {
		ics += 6 + 1
	}
	perChan := 8 + 1 + 1 + 1
	total := 3 + 4
	if e.channels == 2 {
		total += 1 + ics + 2
		if msMask == 1 {
			total += groups * maxSfb
		}
		total += 2 * perChan
	} else {
		total += ics + perChan
	}
	for c := 0; c < e.channels; c++ {
		total += e.tns[c].sideBits()
	}
	total += 3 + 7
	return total
}

func (e *Encoder) writeICSBody(c int, info func()) {
	cq := &e.cq[c]
	w := &e.w
	w.writeBits(8, uint64(cq.globalGain))
	if info != nil {
		info()
	}
	lenEsc := uint64(1)<<uint(cq.lenBits) - 1
	for g := 0; g < cq.nGroups; g++ {
		k := 0
		for k < cq.maxSfb {
			cb := cq.bands[g*cq.maxSfb+k].cb
			run := 1
			for k+run < cq.maxSfb && cq.bands[g*cq.maxSfb+k+run].cb == cb {
				run++
			}
			w.writeBits(4, uint64(cb))
			l := run
			for l >= int(lenEsc) {
				w.writeBits(uint(cq.lenBits), lenEsc)
				l -= int(lenEsc)
			}
			w.writeBits(uint(cq.lenBits), uint64(l))
			k += run
		}
	}
	prev := cq.globalGain
	for g := 0; g < cq.nGroups; g++ {
		for k := 0; k < cq.maxSfb; k++ {
			b := &cq.bands[g*cq.maxSfb+k]
			if b.cb == 0 {
				continue
			}
			e.w.writeSFDelta(b.sf - prev)
			prev = b.sf
		}
	}
	w.writeBits(1, 0)
	if e.tns[c].present {
		w.writeBits(1, 1)
		e.tns[c].write(w)
	} else {
		w.writeBits(1, 0)
	}
	w.writeBits(1, 0)
	var vbuf [1024]int
	for g := 0; g < cq.nGroups; g++ {
		for k := 0; k < cq.maxSfb; {
			b := &cq.bands[g*cq.maxSfb+k]
			run := 1
			for k+run < cq.maxSfb && cq.bands[g*cq.maxSfb+k+run].cb == b.cb {
				run++
			}
			if b.cb != 0 {
				n := 0
				for j := 0; j < run; j++ {
					bb := &cq.bands[g*cq.maxSfb+k+j]
					for i := 0; i < bb.n; i++ {
						vbuf[n] = cq.q[bb.off+i]
						n++
					}
				}
				w.writeSpecRun(b.cb, vbuf[:n])
			}
			k += run
		}
	}
}

func (e *Encoder) writeICSInfo(seq, maxSfb int, groupLen []int) {
	w := &e.w
	w.writeBits(1, 0)
	w.writeBits(2, uint64(seq))
	w.writeBits(1, shapeSine)
	if seq == eightShort {
		w.writeBits(4, uint64(maxSfb))
		bits := uint64(0)
		win := 0
		for _, L := range groupLen {
			for j := 1; j < L; j++ {
				bits |= 1 << uint(6-(win+j-1))
			}
			win += L
		}
		w.writeBits(7, bits)
	} else {
		w.writeBits(6, uint64(maxSfb))
		w.writeBits(1, 0)
	}
}

func (e *Encoder) writeSCE(seq int, groupLen []int, maxSfb int) {
	e.w.writeBits(3, elSCE)
	e.w.writeBits(4, 0)
	e.writeICSBody(0, func() { e.writeICSInfo(seq, maxSfb, groupLen) })
}

func (e *Encoder) writeCPE(seq int, groupLen []int, maxSfb, msMask int) {
	e.w.writeBits(3, elCPE)
	e.w.writeBits(4, 0)
	e.w.writeBits(1, 1)
	e.writeICSInfo(seq, maxSfb, groupLen)
	e.w.writeBits(2, uint64(msMask))
	if msMask == 1 {
		for g := range groupLen {
			for sfb := 0; sfb < maxSfb; sfb++ {
				v := uint64(0)
				if e.msUse[g][sfb] {
					v = 1
				}
				e.w.writeBits(1, v)
			}
		}
	}
	e.writeICSBody(0, nil)
	e.writeICSBody(1, nil)
}

// Finish pads the tail to a whole block, encodes it, then encodes one final block so every real sample is covered by two overlapping windows, and reports the gapless trailer.
func (e *Encoder) Finish(emit func(codec.Packet) error) (codec.Trailer, error) {
	if n := len(e.pending[0]); n > 0 {
		for c := 0; c < e.channels; c++ {
			e.pending[c] = append(e.pending[c], make([]float32, frameLen-n)...)
		}
		if err := e.pushBlock(emit); err != nil {
			return codec.Trailer{}, err
		}
	}
	for c := 0; c < e.channels; c++ {
		e.pending[c] = append(e.pending[c][:0], make([]float32, frameLen)...)
	}
	if err := e.pushBlock(emit); err != nil {
		return codec.Trailer{}, err
	}
	for c := 0; c < e.channels; c++ {
		e.pending[c] = e.pending[c][:0]
	}
	delay := int64(EncoderDelay)
	padding := e.outFrames*frameLen - e.inSamples - delay
	if padding < 0 {
		padding = 0
	}
	return codec.Trailer{Samples: e.inSamples, Delay: delay, Padding: padding}, nil
}
