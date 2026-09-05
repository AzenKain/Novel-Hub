package opus

import (
	"encoding/binary"
	"fmt"
	"math"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/waxerr"
)

const (
	opusFrameLM       = celtMaxLM
	opusFrameSize     = celtShortMDCTSize << opusFrameLM
	opusCELTBands     = 21
	delayCompensation = SampleRate / 250
	EncoderDelay      = SampleRate/400 + delayCompensation
	DefaultBitrate    = 96000
	// maxFrameBytes is RFC 6716's cap on one frame's compressed data (1275 bytes); the packet clamp keeps TOC+payload within it, one byte conservative, matching the reference's packet_size_cap.
	maxFrameBytes = 1275
	encoderBuffer = SampleRate / 100
)

const modeNone = -1

// Signal is a content-type hint steering the encoder's speech/music mode and bandwidth decisions (OPUS_SET_SIGNAL): voice/music pin the voice estimate the decisions interpolate on, overriding the analyser's probability.
type Signal int

const (
	SignalAuto Signal = iota
	SignalVoice
	SignalMusic
)

const (
	signalAuto  = SignalAuto
	signalVoice = SignalVoice
	signalMusic = SignalMusic
)

const (
	bandwidthNarrow    = 1101
	bandwidthMedium    = 1102
	bandwidthWide      = 1103
	bandwidthSuperwide = 1104
	bandwidthFull      = 1105
)

// EncoderVersion identifies the encoder bitstream/algorithm revision for the cache key.
const EncoderVersion = "opus-enc-5"

// EncoderOptions configures the Opus encoder.
type EncoderOptions struct {
	Bitrate        int
	Complexity     int
	VBR            bool
	ConstrainedVBR bool
	Signal         Signal
	LSBDepth       int
}

// DefaultComplexity is the analysis depth used when none is requested.
const DefaultComplexity = 5

// Encoder is a full Opus encoder (SILK, hybrid, and CELT modes with analysis-driven selection) producing raw Opus packets.
type Encoder struct {
	fmt      audio.Format
	channels int
	bitrate  int
	vbr      bool
	useCVBR  bool

	celt     *celtEncoder
	silk     *silkEncoder
	silkMode silkEncControl
	analysis tonalityAnalysisState

	mode              int
	prevMode          int
	forcedMode        int
	signal            Signal
	lsbDepth          int
	bandwidth         int
	autoBandwidth     int
	detectedBandwidth int
	voiceRatio        int
	first             bool
	silkBwSwitch      bool

	hpMem                [4]float32
	delayBuffer          [][]float32
	prevHBGain           float32
	hybridStereoWidthQ14 int16
	variableHPSmth2Q15   int32
	widthMem             stereoWidthState
	rangeFinal           uint32

	complexity int

	buf       [][]float32
	inSamples int64
	outFrames int64
}

// NewEncoder returns an Opus encoder for the given input format, which must be 48 kHz float32, mono or stereo.
func NewEncoder(f audio.Format, opts *EncoderOptions) (*Encoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.Rate != SampleRate || f.Channels < 1 || f.Channels > 2 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("opus: input %v is not an Opus encode shape (48 kHz float32, 1-2 ch)", f))
	}
	bitrate := DefaultBitrate
	if opts != nil && opts.Bitrate != 0 {
		bitrate = opts.Bitrate
	}
	if bitrate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("opus: bit rate %d must be positive", bitrate))
	}
	if bitrate < 6000 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("opus: bit rate %d below the 6000 b/s floor", bitrate))
	}
	complexity := DefaultComplexity
	if opts != nil && opts.Complexity != 0 {
		complexity = opts.Complexity
	}
	if complexity == -1 {
		complexity = 0
	}
	if complexity < 0 || complexity > 10 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("opus: complexity %d outside -1..10", complexity))
	}
	signal := SignalAuto
	if opts != nil {
		signal = opts.Signal
	}
	if signal != SignalAuto && signal != SignalVoice && signal != SignalMusic {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("opus: signal hint %d is not auto, voice, or music", signal))
	}
	lsbDepth := 24
	if opts != nil && opts.LSBDepth != 0 {
		lsbDepth = opts.LSBDepth
	}
	if lsbDepth < 8 || lsbDepth > 24 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("opus: lsb depth %d outside 8..24", lsbDepth))
	}
	vbr := opts != nil && opts.VBR
	e := &Encoder{
		fmt:        f,
		channels:   f.Channels,
		bitrate:    bitrate,
		vbr:        vbr,
		useCVBR:    vbr && opts != nil && opts.ConstrainedVBR,
		celt:       newCELTEncoder(f.Channels),
		silk:       newSILKEncoder(f.Channels),
		complexity: complexity,
		signal:     signal,
		lsbDepth:   lsbDepth,
		voiceRatio: -1,
		first:      true,
		prevMode:   modeNone,
		forcedMode: modeNone,
		prevHBGain: 1.0,
		bandwidth:  bandwidthFull,
	}
	e.hybridStereoWidthQ14 = 1 << 14
	e.variableHPSmth2Q15 = silkLSHIFT(silkLin2Log(silkFixConst(variableHPMinCutoffHz, 16))-16<<7, 8)
	e.celt.bitrate = bitrate
	e.celt.complexity = complexity
	e.silkMode = silkEncControl{
		nChannelsAPI:              f.Channels,
		nChannelsInternal:         f.Channels,
		apiSampleRate:             SampleRate,
		maxInternalSampleRate:     16000,
		minInternalSampleRate:     8000,
		desiredInternalSampleRate: 16000,
		payloadSizeMS:             20,
		bitRate:                   int32(bitrate),
		complexity:                complexity,
	}
	e.delayBuffer = make([][]float32, f.Channels)
	e.buf = make([][]float32, f.Channels)
	for c := 0; c < f.Channels; c++ {
		e.delayBuffer[c] = make([]float32, encoderBuffer)
	}
	return e, nil
}

// InputFormat is the PCM format the encoder consumes.
func (e *Encoder) InputFormat() audio.Format { return e.fmt }

// FrameSize is the encoder-native chunk: one 20 ms frame.
func (e *Encoder) FrameSize() int { return opusFrameSize }

// Bitrate reports the bit rate in bits per second the stream can be relied on to hold: the exact rate in CBR, the reservoir-bounded long-term target in constrained VBR, and 0 for unconstrained VBR, whose rate is signal-dependent (size and rate hints are then honestly unknown).
func (e *Encoder) Bitrate() int {
	if e.vbr && !e.useCVBR {
		return 0
	}
	if e.vbr {
		return e.bitrate
	}
	return e.cbrBytes() * 8 * (SampleRate / opusFrameSize)
}

func (e *Encoder) cbrBytes() int {
	n := (e.bitrate*opusFrameSize + 4*SampleRate) / (8 * SampleRate)
	if n < 3 {
		n = 3
	}
	if n > maxFrameBytes {
		n = maxFrameBytes
	}
	return n
}

// FinalRange reports the range coder's final state after the most recently emitted packet, the integrity value libopus exposes as OPUS_GET_FINAL_RANGE: a conformant decoder's range state after decoding that packet must equal it.
func (e *Encoder) FinalRange() uint32 { return e.rangeFinal }

// CodecConfig returns the OpusHead identification header (RFC 7845).
func (e *Encoder) CodecConfig() []byte {
	head := make([]byte, 19)
	copy(head, "OpusHead")
	head[8] = 1
	head[9] = byte(e.channels)
	binary.LittleEndian.PutUint16(head[10:], uint16(EncoderDelay))
	binary.LittleEndian.PutUint32(head[12:], SampleRate)
	binary.LittleEndian.PutUint16(head[16:], 0)
	head[18] = 0
	return head
}

// Encode buffers src and emits every whole 20 ms frame that becomes available.
func (e *Encoder) Encode(src *audio.Buffer, emit func(codec.Packet) error) error {
	if src.Fmt != e.fmt {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("opus: encode input %v disagrees with %v", src.Fmt, e.fmt))
	}
	for c := 0; c < e.channels; c++ {
		e.buf[c] = append(e.buf[c], src.ChanF(c)[:src.N]...)
	}
	e.inSamples += int64(src.N)
	return e.drainFrames(emit)
}

func (e *Encoder) drainFrames(emit func(codec.Packet) error) error {
	for len(e.buf[0]) >= opusFrameSize {
		if err := e.emitFrame(emit); err != nil {
			return err
		}
		for c := 0; c < e.channels; c++ {
			e.buf[c] = append(e.buf[c][:0], e.buf[c][opusFrameSize:]...)
		}
	}
	return nil
}

func (e *Encoder) emitFrame(emit func(codec.Packet) error) error {
	pcm := make([][]float32, e.channels)
	for c := 0; c < e.channels; c++ {
		pcm[c] = e.buf[c][:opusFrameSize]
	}
	pkt := e.encodePacket(pcm)
	e.outFrames++
	return emit(codec.Packet{Data: pkt, PTS: (e.outFrames - 1) * opusFrameSize, Dur: opusFrameSize, Sync: true})
}

// Finish pads the tail to a whole frame, emits enough frames to cover the pre-skip priming, and reports the gapless trailer.
func (e *Encoder) Finish(emit func(codec.Packet) error) (codec.Trailer, error) {
	if n := len(e.buf[0]); n > 0 {
		for c := 0; c < e.channels; c++ {
			e.buf[c] = append(e.buf[c], make([]float32, opusFrameSize-n)...)
		}
		if err := e.emitFrame(emit); err != nil {
			return codec.Trailer{}, err
		}
		for c := 0; c < e.channels; c++ {
			e.buf[c] = e.buf[c][:0]
		}
	}
	for e.outFrames*opusFrameSize < e.inSamples+EncoderDelay {
		for c := 0; c < e.channels; c++ {
			e.buf[c] = append(e.buf[c][:0], make([]float32, opusFrameSize)...)
		}
		if err := e.emitFrame(emit); err != nil {
			return codec.Trailer{}, err
		}
		for c := 0; c < e.channels; c++ {
			e.buf[c] = e.buf[c][:0]
		}
	}
	delay := int64(EncoderDelay)
	padding := e.outFrames*opusFrameSize - e.inSamples - delay
	if padding < 0 {
		padding = 0
	}
	return codec.Trailer{Samples: e.inSamples, Delay: delay, Padding: padding}, nil
}

var opusModeThresholds = [2][2]int32{
	{64000, 10000},
	{44000, 10000},
}

var monoVoiceBandwidthThresholds = [8]int32{9000, 700, 9000, 700, 13500, 1000, 14000, 2000}
var monoMusicBandwidthThresholds = [8]int32{9000, 700, 9000, 700, 11000, 1000, 12000, 2000}
var stereoVoiceBandwidthThresholds = [8]int32{9000, 700, 9000, 700, 13500, 1000, 14000, 2000}
var stereoMusicBandwidthThresholds = [8]int32{9000, 700, 9000, 700, 11000, 1000, 12000, 2000}

func computeEquivRate(bitrate int32, channels, frameRate int, vbr bool, mode, complexity, loss int) int32 {
	equiv := bitrate
	if frameRate > 50 {
		equiv -= int32((40*channels + 20) * (frameRate - 50))
	}
	if !vbr {
		equiv -= equiv / 12
	}
	equiv = equiv * int32(90+complexity) / 100
	switch {
	case mode == modeSILK || mode == modeHybrid:
		if complexity < 2 {
			equiv = equiv * 4 / 5
		}
		equiv -= equiv * int32(loss) / int32(6*loss+10)
	case mode == modeCELT:
		if complexity < 5 {
			equiv = equiv * 9 / 10
		}
	default:
		equiv -= equiv * int32(loss) / int32(12*loss+20)
	}
	return equiv
}

func computeSILKRateForHybrid(rate int, bandwidth int, frame20ms, vbr, fec bool, channels int) int {
	rateTable := [7][5]int{
		{0, 0, 0, 0, 0},
		{12000, 10000, 10000, 11000, 11000},
		{16000, 13500, 13500, 15000, 15000},
		{20000, 16000, 16000, 18000, 18000},
		{24000, 18000, 18000, 21000, 21000},
		{32000, 22000, 22000, 28000, 28000},
		{64000, 38000, 38000, 50000, 50000},
	}
	rate /= channels
	entry := 1 + b2i(frame20ms) + 2*b2i(fec)
	N := len(rateTable)
	i := 1
	for ; i < N; i++ {
		if rateTable[i][0] > rate {
			break
		}
	}
	var silkRate int
	if i == N {
		silkRate = rateTable[i-1][entry]
		silkRate += (rate - rateTable[i-1][0]) / 2
	} else {
		lo := rateTable[i-1][entry]
		hi := rateTable[i][entry]
		x0 := rateTable[i-1][0]
		x1 := rateTable[i][0]
		silkRate = (lo*(x1-rate) + hi*(rate-x0)) / (x1 - x0)
	}
	if !vbr {
		silkRate += 100
	}
	if bandwidth == bandwidthSuperwide {
		silkRate += 300
	}
	silkRate *= channels
	if channels == 2 && rate >= 12000 {
		silkRate -= 1000
	}
	return silkRate
}

func computeRedundancyBytes(maxDataBytes int, bitrateBps int32, frameRate, channels int) int {
	baseBits := 40*channels + 20
	redundancyRate := int(bitrateBps) + baseBits*(200-frameRate)
	redundancyRate = 3 * redundancyRate / 2
	redundancyBytes := redundancyRate / 1600

	availableBits := maxDataBytes*8 - 2*baseBits
	cap := (availableBits*240/(240+48000/frameRate) + baseBits) / 8
	if redundancyBytes > cap {
		redundancyBytes = cap
	}
	if redundancyBytes > 4+8*channels {
		if redundancyBytes > 257 {
			redundancyBytes = 257
		}
	} else {
		redundancyBytes = 0
	}
	return redundancyBytes
}

func genTOC(mode, frameRate, bandwidth, channels int) byte {
	period := 0
	for frameRate < 400 {
		frameRate <<= 1
		period++
	}
	var toc byte
	switch mode {
	case modeSILK:
		toc = byte(bandwidth-bandwidthNarrow) << 5
		toc |= byte(period-2) << 3
	case modeCELT:
		tmp := bandwidth - bandwidthMedium
		if tmp < 0 {
			tmp = 0
		}
		toc = 0x80
		toc |= byte(tmp) << 5
		toc |= byte(period) << 3
	default:
		toc = 0x60
		toc |= byte(bandwidth-bandwidthSuperwide) << 4
		toc |= byte(period-2) << 3
	}
	if channels == 2 {
		toc |= 1 << 2
	}
	return toc
}

func dcReject(in [][]float32, cutoffHz int, out [][]float32, hpMem []float32, length, channels int) {
	coef := 6.3 * float32(cutoffHz) / SampleRate
	coef2 := 1 - coef
	for c := 0; c < channels; c++ {
		m0 := hpMem[2*c]
		for i := 0; i < length; i++ {
			x := in[c][i]
			y := x - m0
			m0 = coef*x + 1e-30 + coef2*m0
			out[c][i] = y
		}
		hpMem[2*c] = m0
	}
}

func (e *Encoder) gainFade(buf [][]float32, g1, g2 float32, frameSize int) {
	overlap := e.celt.overlap
	for i := 0; i < overlap; i++ {
		w := float32(opusFadeWindow[i])
		w = w * w
		g := w*g2 + (1-w)*g1
		for c := 0; c < e.channels; c++ {
			buf[c][i] *= g
		}
	}
	for c := 0; c < e.channels; c++ {
		for i := overlap; i < frameSize; i++ {
			buf[c][i] *= g2
		}
	}
}

func (e *Encoder) stereoFade(buf [][]float32, g1, g2 float32, frameSize int) {
	overlap := e.celt.overlap
	g1 = 1 - g1
	g2 = 1 - g2
	for i := 0; i < frameSize; i++ {
		var g float32
		if i < overlap {
			w := float32(opusFadeWindow[i])
			w = w * w
			g = w*g2 + (1-w)*g1
		} else {
			g = g2
		}
		diff := 0.5 * (buf[0][i] - buf[1][i]) * g
		buf[0][i] -= diff
		buf[1][i] += diff
	}
}

type stereoWidthState struct {
	XX, XY, YY    float32
	smoothedWidth float32
	maxFollower   float32
}

func (m *stereoWidthState) compute(pcm [][]float32, frameSize int) float32 {
	frameRate := SampleRate / frameSize
	shortAlpha := float32(25) / float32(maxIntA(50, frameRate))
	var xx, xy, yy float32
	for i := 0; i+3 < frameSize; i += 4 {
		var pxx, pxy, pyy float32
		for j := 0; j < 4; j++ {
			x := pcm[0][i+j]
			y := pcm[1][i+j]
			pxx += x * x
			pxy += x * y
			pyy += y * y
		}
		xx += pxx
		xy += pxy
		yy += pyy
	}
	if !(xx < 1e9) || math.IsNaN(float64(xx)) || !(yy < 1e9) || math.IsNaN(float64(yy)) {
		xx, xy, yy = 0, 0, 0
	}
	m.XX += shortAlpha * (xx - m.XX)
	m.XY = (1-shortAlpha)*m.XY + shortAlpha*xy
	m.YY += shortAlpha * (yy - m.YY)
	m.XX = maxA(0, m.XX)
	m.XY = maxA(0, m.XY)
	m.YY = maxA(0, m.YY)
	if maxA(m.XX, m.YY) > 8e-4 {
		sqrtXX := float32(math.Sqrt(float64(m.XX)))
		sqrtYY := float32(math.Sqrt(float64(m.YY)))
		qrrtXX := float32(math.Sqrt(float64(sqrtXX)))
		qrrtYY := float32(math.Sqrt(float64(sqrtYY)))
		m.XY = minA(m.XY, sqrtXX*sqrtYY)
		corr := m.XY / (1e-15 + sqrtXX*sqrtYY)
		ldiff := 1.0 * absA(qrrtXX-qrrtYY) / (1e-15 + qrrtXX + qrrtYY)
		width := float32(math.Sqrt(float64(1.0-corr*corr))) * ldiff
		m.smoothedWidth += (width - m.smoothedWidth) / float32(frameRate)
		m.maxFollower = maxA(m.maxFollower-0.02/float32(frameRate), m.smoothedWidth)
	}
	return minA(1.0, 20*m.maxFollower)
}

var opusFadeWindow = celtWindow(celtOverlap)

func (e *Encoder) encodePacket(pcm [][]float32) []byte {
	frameSize := opusFrameSize
	C := e.channels
	frameRate := SampleRate / frameSize

	var maxDataBytes int
	if e.vbr {
		maxDataBytes = maxFrameBytes + 1
	} else {
		maxDataBytes = e.cbrBytes()
	}

	var info analysisInfo
	e.analysis.runAnalysis(pcm, frameSize, frameSize, 0, -2, C, e.lsbDepth, &info)

	isSilence := true
	for c := 0; c < C && isSilence; c++ {
		for _, v := range pcm[c] {
			if absA(v) > 1.0/(1<<24) {
				isSilence = false
				break
			}
		}
	}

	e.voiceRatio = -1
	e.detectedBandwidth = 0
	if info.valid {
		var prob float32
		switch {
		case e.prevMode == modeNone:
			prob = info.musicProb
		case e.prevMode == modeCELT:
			prob = info.musicProbMax
		default:
			prob = info.musicProbMin
		}
		e.voiceRatio = int(math.Floor(0.5 + 100*float64(1-prob)))

		switch {
		case info.bandwidth <= 12:
			e.detectedBandwidth = bandwidthNarrow
		case info.bandwidth <= 14:
			e.detectedBandwidth = bandwidthMedium
		case info.bandwidth <= 16:
			e.detectedBandwidth = bandwidthWide
		case info.bandwidth <= 18:
			e.detectedBandwidth = bandwidthSuperwide
		default:
			e.detectedBandwidth = bandwidthFull
		}
	}

	var stereoWidth float32
	if C == 2 {
		stereoWidth = e.widthMem.compute(pcm, frameSize)
	}

	maxRate := int32(maxDataBytes * 8 * frameRate)

	voiceEst := 48
	switch {
	case e.signal == signalVoice:
		voiceEst = 127
	case e.signal == signalMusic:
		voiceEst = 0
	case e.voiceRatio >= 0:
		voiceEst = e.voiceRatio * 327 >> 8
		if voiceEst > 115 {
			voiceEst = 115
		}
	}

	equivRate := computeEquivRate(int32(e.bitrate), C, frameRate, e.vbr, modeNone, e.complexity, 0)

	redundancy := false
	celtToSilk := false
	toCELT := false
	prefill := 0
	if e.forcedMode != modeNone {
		e.mode = e.forcedMode
	} else {
		modeVoice := int32(float32(1-stereoWidth)*float32(opusModeThresholds[0][0]) +
			stereoWidth*float32(opusModeThresholds[1][0]))
		modeMusic := int32(float32(1-stereoWidth)*float32(opusModeThresholds[1][1]) +
			stereoWidth*float32(opusModeThresholds[1][1]))
		threshold := modeMusic + int32(voiceEst*voiceEst)*(modeVoice-modeMusic)>>14
		if e.prevMode == modeCELT {
			threshold -= 4000
		} else if e.prevMode != modeNone {
			threshold += 4000
		}
		if equivRate >= threshold {
			e.mode = modeCELT
		} else {
			e.mode = modeSILK
		}
		if maxDataBytes < 9000/(frameRate*8) {
			e.mode = modeCELT
		}
	}

	if e.prevMode != modeNone &&
		((e.mode != modeCELT && e.prevMode == modeCELT) ||
			(e.mode == modeCELT && e.prevMode != modeCELT)) {
		redundancy = true
		celtToSilk = e.mode != modeCELT
		if !celtToSilk {
			e.mode = e.prevMode
			toCELT = true
		}
	}

	equivRate = computeEquivRate(int32(e.bitrate), C, frameRate, e.vbr, e.mode, e.complexity, 0)

	if e.mode != modeCELT && e.prevMode == modeCELT {
		e.silk = newSILKEncoder(C)
		prefill = 1
	}

	if e.mode == modeCELT || e.first || e.silkMode.allowBandwidthSwitch {
		var voiceTh, musicTh *[8]int32
		if C == 2 {
			voiceTh, musicTh = &stereoVoiceBandwidthThresholds, &stereoMusicBandwidthThresholds
		} else {
			voiceTh, musicTh = &monoVoiceBandwidthThresholds, &monoMusicBandwidthThresholds
		}
		var thresholds [8]int32
		for i := 0; i < 8; i++ {
			thresholds[i] = musicTh[i] + int32(voiceEst*voiceEst)*(voiceTh[i]-musicTh[i])>>14
		}
		bandwidth := bandwidthFull
		for bandwidth > bandwidthNarrow {
			threshold := thresholds[2*(bandwidth-bandwidthMedium)]
			hysteresis := thresholds[2*(bandwidth-bandwidthMedium)+1]
			if !e.first {
				if e.autoBandwidth >= bandwidth {
					threshold -= hysteresis
				} else {
					threshold += hysteresis
				}
			}
			if equivRate >= threshold {
				break
			}
			bandwidth--
		}
		if bandwidth == bandwidthMedium {
			bandwidth = bandwidthWide
		}
		e.bandwidth = bandwidth
		e.autoBandwidth = bandwidth
		if !e.first && e.mode != modeCELT &&
			!e.silkMode.inWBmodeWithoutVariableLP && e.bandwidth > bandwidthWide {
			e.bandwidth = bandwidthWide
		}
	}

	if e.mode != modeCELT && maxRate < 15000 {
		if e.bandwidth > bandwidthWide {
			e.bandwidth = bandwidthWide
		}
	}

	if e.detectedBandwidth != 0 {
		var minDetected int
		switch {
		case equivRate <= int32(18000*C) && e.mode == modeCELT:
			minDetected = bandwidthNarrow
		case equivRate <= int32(24000*C) && e.mode == modeCELT:
			minDetected = bandwidthMedium
		case equivRate <= int32(30000*C):
			minDetected = bandwidthWide
		case equivRate <= int32(44000*C):
			minDetected = bandwidthSuperwide
		default:
			minDetected = bandwidthFull
		}
		if e.detectedBandwidth < minDetected {
			e.detectedBandwidth = minDetected
		}
		if e.bandwidth > e.detectedBandwidth {
			e.bandwidth = e.detectedBandwidth
		}
	}

	if e.mode == modeCELT && e.bandwidth == bandwidthMedium {
		e.bandwidth = bandwidthWide
	}

	curBandwidth := e.bandwidth
	if e.mode == modeSILK && curBandwidth > bandwidthWide {
		e.mode = modeHybrid
	}
	if e.mode == modeHybrid && curBandwidth <= bandwidthWide {
		e.mode = modeSILK
	}

	if isSilence {
		e.voiceRatio = -1
	}

	if e.silkBwSwitch {
		redundancy = true
		celtToSilk = true
		e.silkBwSwitch = false
		prefill = 2
	}
	if e.mode == modeCELT {
		redundancy = false
	}

	redundancyBytes := 0
	if redundancy {
		redundancyBytes = computeRedundancyBytes(minIntA(maxDataBytes, maxFrameBytes+1), int32(e.bitrate), frameRate, C)
		if redundancyBytes == 0 {
			redundancy = false
		}
	}

	bitsTarget := minIntA(8*(maxDataBytes-redundancyBytes), e.bitrate*frameSize/SampleRate) - 8

	payloadCap := minIntA(maxDataBytes, maxFrameBytes+1) - 1
	data := make([]byte, payloadCap)
	enc := newRangeEncoder(data)

	totalBuffer := delayCompensation
	pcmBuf := make([][]float32, C)
	for c := 0; c < C; c++ {
		pcmBuf[c] = make([]float32, totalBuffer+frameSize)
		copy(pcmBuf[c][:totalBuffer], e.delayBuffer[c][encoderBuffer-totalBuffer:])
	}

	var hpFreqSmth1 int32
	if e.mode == modeCELT {
		hpFreqSmth1 = silkLSHIFT(silkLin2Log(variableHPMinCutoffHz), 8)
	} else {
		hpFreqSmth1 = e.silk.channel[0].variableHPSmth1Q15
	}
	e.variableHPSmth2Q15 = silkSMLAWB(e.variableHPSmth2Q15,
		hpFreqSmth1-e.variableHPSmth2Q15, silkFixConst(variableHPSmthCoef2, 16))

	{
		out := make([][]float32, C)
		for c := 0; c < C; c++ {
			out[c] = pcmBuf[c][totalBuffer:]
		}
		dcReject(pcm, 3, out, e.hpMem[:], frameSize, C)
	}

	{
		var sum float64
		for c := 0; c < C; c++ {
			sum += silkEnergyFLP(pcmBuf[c][totalBuffer:], frameSize)
		}
		if !(sum < 1e9) || math.IsNaN(sum) {
			for c := 0; c < C; c++ {
				for i := range pcmBuf[c][totalBuffer:] {
					pcmBuf[c][totalBuffer+i] = 0
				}
			}
			e.hpMem = [4]float32{}
		}
	}

	hbGain := float32(1.0)
	if e.mode != modeCELT {
		totalRate := int32(bitsTarget) * int32(frameRate)
		if e.mode == modeHybrid {
			e.silkMode.bitRate = int32(computeSILKRateForHybrid(int(totalRate),
				curBandwidth, true, e.vbr, false, C))
			celtRate := totalRate - e.silkMode.bitRate
			hbGain = 1.0 - float32(math.Exp2(-float64(celtRate)*(1.0/1024)))
		} else {
			e.silkMode.bitRate = totalRate
		}

		e.silkMode.payloadSizeMS = 20
		e.silkMode.nChannelsAPI = C
		e.silkMode.nChannelsInternal = C
		switch curBandwidth {
		case bandwidthNarrow:
			e.silkMode.desiredInternalSampleRate = 8000
		case bandwidthMedium:
			e.silkMode.desiredInternalSampleRate = 12000
		default:
			e.silkMode.desiredInternalSampleRate = 16000
		}
		if e.mode == modeHybrid {
			e.silkMode.minInternalSampleRate = 16000
		} else {
			e.silkMode.minInternalSampleRate = 8000
		}
		e.silkMode.maxInternalSampleRate = 16000
		if e.mode == modeSILK {
			effectiveMaxRate := int32(maxDataBytes * 8 * frameRate)
			if effectiveMaxRate < 8000 {
				e.silkMode.maxInternalSampleRate = 12000
				e.silkMode.desiredInternalSampleRate = minIntA(12000, e.silkMode.desiredInternalSampleRate)
			}
			if effectiveMaxRate < 7000 {
				e.silkMode.maxInternalSampleRate = 8000
				e.silkMode.desiredInternalSampleRate = minIntA(8000, e.silkMode.desiredInternalSampleRate)
			}
		}

		e.silkMode.useCBR = !e.vbr
		e.silkMode.complexity = e.complexity
		e.silkMode.maxBits = (minIntA(maxDataBytes, maxFrameBytes+1) - 1) * 8
		if redundancy && redundancyBytes >= 2 {
			e.silkMode.maxBits -= redundancyBytes*8 + 1
			if e.mode == modeHybrid {
				e.silkMode.maxBits -= 20
			}
		}
		if e.silkMode.useCBR {
			if e.mode == modeHybrid {
				otherBits := maxIntA(0, e.silkMode.maxBits-int(e.silkMode.bitRate)*frameSize/SampleRate)
				e.silkMode.maxBits = maxIntA(0, e.silkMode.maxBits-otherBits*3/4)
				e.silkMode.useCBR = false
			}
		} else {
			if e.mode == modeHybrid {
				maxBitRate := computeSILKRateForHybrid(e.silkMode.maxBits*frameRate,
					curBandwidth, true, e.vbr, false, C)
				e.silkMode.maxBits = maxBitRate / frameRate
			}
		}

		if prefill != 0 {
			prefillOffset := encoderBuffer - delayCompensation - SampleRate/400
			for c := 0; c < C; c++ {
				ramp := e.delayBuffer[c][prefillOffset:]
				for i := 0; i < e.celt.overlap && i < SampleRate/400; i++ {
					w := float32(opusFadeWindow[i])
					ramp[i] *= w * w
				}
				for i := 0; i < prefillOffset; i++ {
					e.delayBuffer[c][i] = 0
				}
			}
			pcm16 := interleaveToInt16(e.delayBuffer, C, encoderBuffer)
			zero := 0
			e.silk.encode(&e.silkMode, pcm16, encoderBuffer, nil, &zero, prefill, -1)
			e.silkMode.opusCanSwitch = false
		}

		pcm16 := interleaveToInt16planarView(pcmBuf, C, totalBuffer, frameSize)
		nBytes := 0
		e.silk.encode(&e.silkMode, pcm16, frameSize, enc, &nBytes, 0, -1)

		if e.mode == modeSILK {
			switch e.silkMode.internalSampleRate {
			case 8000:
				curBandwidth = bandwidthNarrow
			case 12000:
				curBandwidth = bandwidthMedium
			case 16000:
				curBandwidth = bandwidthWide
			}
		}

		e.silkMode.opusCanSwitch = e.silkMode.switchReady
		if e.silkMode.opusCanSwitch {
			redundancyBytes = computeRedundancyBytes(minIntA(maxDataBytes, maxFrameBytes+1), int32(e.bitrate), frameRate, C)
			redundancy = redundancyBytes != 0
			celtToSilk = false
			e.silkBwSwitch = true
		}
	}

	endband := 21
	switch curBandwidth {
	case bandwidthNarrow:
		endband = 13
	case bandwidthMedium, bandwidthWide:
		endband = 17
	case bandwidthSuperwide:
		endband = 19
	}
	e.celt.disablePF = e.mode != modeCELT
	e.celt.forceIntra = false

	var tmpPrefill [][]float32
	if e.mode != modeSILK && e.mode != e.prevMode && e.prevMode != modeNone {
		tmpPrefill = make([][]float32, C)
		for c := 0; c < C; c++ {
			tmpPrefill[c] = make([]float32, SampleRate/400)
			copy(tmpPrefill[c], e.delayBuffer[c][encoderBuffer-totalBuffer-SampleRate/400:])
		}
	}

	for c := 0; c < C; c++ {
		if encoderBuffer-(frameSize+totalBuffer) > 0 {
			copy(e.delayBuffer[c], e.delayBuffer[c][frameSize:])
			copy(e.delayBuffer[c][encoderBuffer-frameSize-totalBuffer:], pcmBuf[c])
		} else {
			copy(e.delayBuffer[c], pcmBuf[c][frameSize+totalBuffer-encoderBuffer:])
		}
	}

	if e.prevHBGain < 1.0 || hbGain < 1.0 {
		e.gainFade(pcmBuf, e.prevHBGain, hbGain, totalBuffer+frameSize)
	}
	e.prevHBGain = hbGain

	if e.mode != modeHybrid || C == 1 {
		var w int32
		switch {
		case equivRate > 32000:
			w = 16384
		case equivRate < 16000:
			w = 0
		default:
			w = 16384 - 2048*(32000-equivRate)/(equivRate-14000)
		}
		e.silkMode.stereoWidthQ14 = int16(w)
	}
	if C == 2 {
		if e.hybridStereoWidthQ14 < 1<<14 || e.silkMode.stereoWidthQ14 < 1<<14 {
			g1 := float32(e.hybridStereoWidthQ14) / 16384
			g2 := float32(e.silkMode.stereoWidthQ14) / 16384
			e.stereoFade(pcmBuf, g1, g2, totalBuffer+frameSize)
			e.hybridStereoWidthQ14 = e.silkMode.stereoWidthQ14
		}
	}

	nbCompressedBytes := 0
	var redundantRng uint32
	if e.mode != modeCELT && enc.tell()+17+20*b2i(e.mode == modeHybrid) <= 8*(payloadCap) {
		if e.mode == modeHybrid {
			enc.encodeBitLogp(b2i(redundancy), 12)
		}
		if redundancy {
			enc.encodeBitLogp(b2i(celtToSilk), 1)
			var maxRedundancy int
			if e.mode == modeHybrid {
				maxRedundancy = payloadCap - (enc.tell()+8+3+7)>>3
			} else {
				maxRedundancy = payloadCap - (enc.tell()+7)>>3
			}
			redundancyBytes = minIntA(maxRedundancy, redundancyBytes)
			redundancyBytes = minIntA(257, maxIntA(2, redundancyBytes))
			if e.mode == modeHybrid {
				enc.encodeUint(uint32(redundancyBytes-2), 256)
			}
		}
	} else {
		redundancy = false
	}
	if !redundancy {
		e.silkBwSwitch = false
		redundancyBytes = 0
	}
	startBand := 0
	if e.mode != modeCELT {
		startBand = 17
	}

	if e.mode == modeSILK {
		nbCompressedBytes = (enc.tell() + 7) >> 3
		enc.done()
	} else {
		nbCompressedBytes = payloadCap - redundancyBytes
		enc.shrink(nbCompressedBytes)
	}

	if redundancy || e.mode != modeSILK {
		e.celt.analysis = info
	}
	if e.mode == modeHybrid {
		e.celt.silkInfoSignalType = int(e.silkMode.signalType)
		e.celt.silkInfoOffset = int(e.silkMode.offset)
	} else {
		e.celt.silkInfoSignalType = 0
		e.celt.silkInfoOffset = 0
	}

	if redundancy && celtToSilk {
		e.celt.vbr = false
		prevBitrate := e.celt.bitrate
		e.celt.bitrate = 0
		sub := subFrames(pcmBuf, 0, SampleRate/200)
		redPayload := e.celt.celtEncode(sub, SampleRate/200, 1, C, 0, endband, redundancyBytes)
		copy(data[nbCompressedBytes:], redPayload)
		redundantRng = e.celt.rng
		e.celt.bitrate = prevBitrate
		e.celt.Reset()
	}

	if e.mode != modeSILK {
		e.celt.vbr = e.vbr
		e.celt.constrainedVBR = false
		if e.mode == modeHybrid {
			e.celt.bitrate = e.bitrate - int(e.silkMode.bitRate)
		} else {
			e.celt.bitrate = e.bitrate
			e.celt.constrainedVBR = e.useCVBR
		}
		if !e.vbr {
			e.celt.bitrate = 0
		}

		if e.mode != e.prevMode && e.prevMode != modeNone {
			e.celt.Reset()
			e.celt.disablePF = true
			e.celt.forceIntra = true
			prevVBR := e.celt.vbr
			e.celt.vbr = false
			bitratePrev := e.celt.bitrate
			e.celt.bitrate = 0
			e.celt.celtEncode(tmpPrefill, SampleRate/400, 0, C, 0, endband, 2)
			e.celt.bitrate = bitratePrev
			e.celt.vbr = prevVBR
			e.celt.forceIntra = false
			e.celt.disablePF = true
		}

		if enc.tell() <= 8*nbCompressedBytes {
			frame := subFrames(pcmBuf, 0, frameSize)
			ret := e.celt.celtEncodeWithEC(frame, frameSize, opusFrameLM, C, startBand, endband, enc, nbCompressedBytes)
			if redundancy && celtToSilk && e.mode == modeHybrid && nbCompressedBytes != ret {
				copy(data[ret:ret+redundancyBytes], data[nbCompressedBytes:nbCompressedBytes+redundancyBytes])
			}
			nbCompressedBytes = ret
		}
		e.rangeFinal = e.celt.rng
	} else {
		e.rangeFinal = enc.rng
	}

	if redundancy && !celtToSilk {
		e.celt.Reset()
		e.celt.disablePF = true
		e.celt.forceIntra = true
		prevVBR := e.celt.vbr
		e.celt.vbr = false
		prevBitrate := e.celt.bitrate
		e.celt.bitrate = 0

		N2 := SampleRate / 200
		N4 := SampleRate / 400
		pre := subFrames(pcmBuf, frameSize-N2-N4, N4)
		e.celt.celtEncode(pre, N4, 0, C, 0, endband, 2)

		red := subFrames(pcmBuf, frameSize-N2, N2)
		redPayload := e.celt.celtEncode(red, N2, 1, C, 0, endband, redundancyBytes)
		copy(data[nbCompressedBytes:], redPayload)
		redundantRng = e.celt.rng
		e.celt.bitrate = prevBitrate
		e.celt.vbr = prevVBR
		e.celt.forceIntra = false
	}

	e.rangeFinal ^= redundantRng

	if toCELT {
		e.prevMode = modeCELT
	} else {
		e.prevMode = e.mode
	}
	e.first = false

	if enc.tell() > payloadCap*8 {
		nbCompressedBytes = 1
		redundancyBytes = 0
		data[0] = 0
	} else if e.mode == modeSILK && !redundancy {
		for nbCompressedBytes > 2 && data[nbCompressedBytes-1] == 0 {
			nbCompressedBytes--
		}
	}

	total := nbCompressedBytes + redundancyBytes
	pkt := make([]byte, 1+total)
	pkt[0] = genTOC(e.mode, frameRate, curBandwidth, C)
	copy(pkt[1:], data[:total])

	// CBR packets keep their fixed size via RFC 6716 code-3 padding.
	if !e.vbr && len(pkt) < maxDataBytes {
		pkt = opusPacketPad(pkt, maxDataBytes)
	}
	return pkt
}

func opusPacketPad(pkt []byte, targetLen int) []byte {
	if len(pkt) >= targetLen {
		return pkt
	}
	frame := pkt[1:]
	out := make([]byte, targetLen)
	out[0] = pkt[0] | 0x3
	rem := targetLen - 2 - len(frame)
	if rem == 0 {
		out[1] = 0x01
		copy(out[2:], frame)
		return out
	}
	out[1] = 0x41
	pos := 2
	for rem > 255 {
		out[pos] = 255
		pos++
		rem -= 255
	}
	out[pos] = byte(rem - 1)
	pos++
	copy(out[pos:], frame)
	return out
}

func subFrames(buf [][]float32, off, n int) [][]float32 {
	out := make([][]float32, len(buf))
	for c := range buf {
		out[c] = buf[c][off : off+n]
	}
	return out
}

func interleaveToInt16(buf [][]float32, C, n int) []int16 {
	out := make([]int16, C*n)
	for c := 0; c < C; c++ {
		for i := 0; i < n; i++ {
			out[i*C+c] = int16(silkSAT16(silkFloat2Int(buf[c][i] * 32768)))
		}
	}
	return out
}

func interleaveToInt16planarView(buf [][]float32, C, off, n int) []int16 {
	out := make([]int16, C*n)
	for c := 0; c < C; c++ {
		for i := 0; i < n; i++ {
			out[i*C+c] = int16(silkSAT16(silkFloat2Int(buf[c][off+i] * 32768)))
		}
	}
	return out
}
