package opus

// CELT encode main loop (RFC 6716 section 4.3, encode direction).

type celtEncoder struct {
	channels int
	overlap  int

	complexity     int
	vbr            bool
	constrainedVBR bool
	bitrate        int

	preemphMem [2]float32
	preHistory [][]float32

	prefilterMem    [][]float32
	prefilterPeriod int
	prefilterGain   float32
	prefilterTapset int

	oldBandE    []float32
	oldLogE     []float32
	oldLogE2    []float32
	energyError []float32

	disablePF          bool
	forceIntra         bool
	silkInfoSignalType int
	silkInfoOffset     int
	analysis           analysisInfo

	delayedIntra    float32
	consecTransient int
	spreadDecision  int
	tonalAverage    int
	hfAverage       int
	tapsetDecision  int
	intensity       int
	stereoSaving    float32
	specAvg         float32
	lastCodedBands  int
	rng             uint32

	vbrReservoir int
	vbrDrift     int
	vbrOffset    int
	vbrCount     int

	window  []float64
	mdctScr *mdctScratch
	mdctPlanCache
	iy []int
	u  []uint32
}

func newCELTEncoder(channels int) *celtEncoder {
	e := &celtEncoder{
		channels:       channels,
		overlap:        celtOverlap,
		complexity:     5,
		oldBandE:       make([]float32, channels*celtNBands),
		oldLogE:        make([]float32, channels*celtNBands),
		oldLogE2:       make([]float32, channels*celtNBands),
		energyError:    make([]float32, channels*celtNBands),
		spreadDecision: spreadNormal,
		tonalAverage:   256,
		window:         celtWindow(celtOverlap),
		mdctScr:        newMDCTScratch(480),
		iy:             make([]int, (celtShortMDCTSize<<celtMaxLM)+3),
		u:              make([]uint32, celtMaxPulses+2),
	}
	e.preHistory = make([][]float32, channels)
	e.prefilterMem = make([][]float32, channels)
	for c := range e.preHistory {
		e.preHistory[c] = make([]float32, celtOverlap)
		e.prefilterMem[c] = make([]float32, combMaxPeriod)
	}
	for i := range e.oldLogE {
		e.oldLogE[i] = -28
		e.oldLogE2[i] = -28
	}
	return e
}

// Reset clears the inter-frame state (matching the decoder's OPUS_RESET_STATE parity: energies to -28, everything else zeroed).
func (e *celtEncoder) Reset() {
	e.preemphMem = [2]float32{}
	for c := range e.preHistory {
		clear(e.preHistory[c])
		clear(e.prefilterMem[c])
	}
	e.prefilterPeriod = 0
	e.prefilterGain = 0
	e.prefilterTapset = 0
	clear(e.oldBandE)
	clear(e.energyError)
	for i := range e.oldLogE {
		e.oldLogE[i] = -28
		e.oldLogE2[i] = -28
	}
	e.analysis = analysisInfo{}
	e.delayedIntra = 0
	e.consecTransient = 0
	e.spreadDecision = spreadNormal
	e.tonalAverage = 256
	e.hfAverage = 0
	e.tapsetDecision = 0
	e.intensity = 0
	e.stereoSaving = 0
	e.specAvg = 0
	e.lastCodedBands = 0
	e.rng = 0
	e.vbrReservoir = 0
	e.vbrDrift = 0
	e.vbrOffset = 0
	e.vbrCount = 0
}

func (e *celtEncoder) celtPreemphasis(pcm, inp []float32, N int, mem *float32) {
	coef0 := float32(celtPreemph)
	m := *mem
	for i := 0; i < N; i++ {
		x := pcm[i] * celtSigScale
		inp[i] = x - m
		m = coef0 * x
	}
	*mem = m
}

func (e *celtEncoder) computeMDCTs(shortBlocks int, in [][]float32, freq []float32, C, LM int) {
	var B, N int
	if shortBlocks != 0 {
		B = shortBlocks
		N = celtShortMDCTSize
	} else {
		B = 1
		N = celtShortMDCTSize << LM
	}
	plan := e.planFor(2 * N)
	frameN := celtShortMDCTSize << LM
	for c := 0; c < C; c++ {
		for b := 0; b < B; b++ {
			plan.forward(in[c][b*N:], freq[c*frameN+b:], B, e.window, e.overlap, e.mdctScr)
		}
	}
}

func tfEncode(start, end, isTransient int, tfRes []int, LM, tfSelect int, e *rangeEncoder) {
	budget := e.storage * 8
	tell := e.tell()
	logp := 4
	if isTransient != 0 {
		logp = 2
	}
	tfSelectRsv := 0
	if LM > 0 && tell+logp+1 <= budget {
		tfSelectRsv = 1
	}
	budget -= tfSelectRsv
	curr, tfChanged := 0, 0
	for i := start; i < end; i++ {
		if tell+logp <= budget {
			e.encodeBitLogp(tfRes[i]^curr, uint(logp))
			tell = e.tell()
			curr = tfRes[i]
			tfChanged |= curr
		} else {
			tfRes[i] = curr
		}
		if isTransient != 0 {
			logp = 4
		} else {
			logp = 5
		}
	}
	if tfSelectRsv != 0 &&
		tfSelectTable[LM][4*isTransient+tfChanged] != tfSelectTable[LM][4*isTransient+2+tfChanged] {
		e.encodeBitLogp(tfSelect, 1)
	} else {
		tfSelect = 0
	}
	for i := start; i < end; i++ {
		tfRes[i] = int(tfSelectTable[LM][4*isTransient+2*tfSelect+tfRes[i]])
	}
}

func (e *celtEncoder) celtEncode(pcm [][]float32, N, LM, C, start, end, nbBytes int) []byte {
	buf := make([]byte, nbBytes)
	enc := newRangeEncoder(buf)
	n := e.celtEncodeWithEC(pcm, N, LM, C, start, end, enc, nbBytes)
	return enc.payload()[:n]
}

func (e *celtEncoder) celtEncodeWithEC(pcm [][]float32, N, LM, C, start, end int, enc *rangeEncoder, nbBytes int) int {
	M := 1 << LM
	overlap := e.overlap
	nb := celtNBands
	effEnd := min(end, nb)
	hybrid := start != 0

	tell0Frac := enc.tellFrac()
	tell := enc.tell()
	nbFilledBytes := 0
	if tell > 1 {
		nbFilledBytes = (tell + 4) >> 3
	} else {
		tell0Frac = 1
		tell = 1
	}

	nbCompressedBytes := nbBytes
	vbrRate := 0
	effectiveBytes := nbCompressedBytes - nbFilledBytes
	if e.vbr && e.bitrate > 0 {
		vbrRate = (e.bitrate / (48000 / N)) << bitRes
		effectiveBytes = vbrRate >> (3 + bitRes)
	}
	totalBits := nbCompressedBytes * 8

	equivRate := nbCompressedBytes*8*50<<(3-LM) - (40*C+20)*((400>>LM)-50)
	if e.bitrate > 0 {
		equivRate = min(equivRate, e.bitrate-(40*C+20)*((400>>LM)-50))
	}

	if vbrRate > 0 && e.constrainedVBR {
		vbrBound := vbrRate
		maxAllowed := min(max(2, (vbrRate+vbrBound-e.vbrReservoir)>>(bitRes+3)), nbCompressedBytes)
		if maxAllowed < nbCompressedBytes {
			nbCompressedBytes = maxAllowed
			totalBits = nbCompressedBytes * 8
			enc.shrink(nbCompressedBytes)
		}
	}

	in := make([][]float32, C)
	var sampleMax float32
	for c := 0; c < C; c++ {
		in[c] = make([]float32, N+overlap)
		copy(in[c][:overlap], e.prefilterMem[c][combMaxPeriod-overlap:])
		e.celtPreemphasis(pcm[c], in[c][overlap:], N, &e.preemphMem[c])
		for i := 0; i < N; i++ {
			if a := absf(pcm[c][i]); a > sampleMax {
				sampleMax = a
			}
		}
	}

	silence := 0
	if sampleMax <= 1.0/(1<<24) {
		silence = 1
	}
	if tell == 1 {
		enc.encodeBitLogp(silence, 15)
	} else {
		silence = 0
	}
	if silence != 0 {
		tellAll := nbCompressedBytes * 8
		enc.nbits += tellAll - enc.tell()
	}

	var toneishness float32
	toneFreq := toneDetect(in, C, N+overlap, &toneishness)

	isTransient := 0
	shortBlocks := 0
	weakTransient := 0
	var tfEstimate float32
	tfChan := 0
	if silence == 0 {
		allowWeak := hybrid && effectiveBytes < 15 && e.silkInfoSignalType != 2
		isTransient = transientAnalysis(in, N+overlap, C, &tfEstimate, &tfChan, allowWeak, &weakTransient, toneFreq, toneishness)
	}
	toneishness = min(toneishness, 1-tfEstimate)

	nbAvailableBytes := nbCompressedBytes - nbFilledBytes
	pitchChange := false
	{
		enabled := silence == 0 && nbAvailableBytes > 12*C && enc.tell()+16 <= totalBits &&
			!hybrid && !e.disablePF
		prefilterTapset := e.tapsetDecision
		pfOn, pitchIndex, gain1, qg := e.runPrefilter(in, C, N, prefilterTapset,
			enabled, tfEstimate, nbAvailableBytes, toneFreq, toneishness)
		if (gain1 > 0.4 || e.prefilterGain > 0.4) && (!e.analysis.valid || e.analysis.tonality > 0.3) &&
			(float64(pitchIndex) > 1.26*float64(e.prefilterPeriod) || float64(pitchIndex) < 0.79*float64(e.prefilterPeriod)) {
			pitchChange = true
		}
		if pfOn == 0 {
			if !hybrid && enc.tell()+16 <= totalBits {
				enc.encodeBitLogp(0, 1)
			}
		} else {
			enc.encodeBitLogp(1, 1)
			pitchIndex++
			octave := ilog(uint32(pitchIndex)) - 5
			enc.encodeUint(uint32(octave), 6)
			enc.encodeRawBits(uint32(pitchIndex-(16<<octave)), uint(4+octave))
			pitchIndex--
			enc.encodeRawBits(uint32(qg), 3)
			enc.encodeICDF(prefilterTapset, celtTapsetICDF, 2)
		}
		e.prefilterPeriod = pitchIndex
		e.prefilterGain = gain1
		e.prefilterTapset = prefilterTapset
	}

	if LM > 0 && enc.tell()+3 <= totalBits {
		if isTransient != 0 {
			shortBlocks = M
		}
	} else {
		isTransient = 0
	}

	freq := make([]float32, C*N)
	bandE := make([]float32, nb*C)
	bandLogE := make([]float32, nb*C)
	bandLogE2 := make([]float32, C*nb)
	secondMdct := shortBlocks != 0 && e.complexity >= 8
	if secondMdct {
		e.computeMDCTs(0, in, freq, C, LM)
		computeBandEnergies(freq, bandE, effEnd, C, LM)
		amp2Log2(effEnd, end, bandE, bandLogE2, C)
		for c := 0; c < C; c++ {
			for i := 0; i < end; i++ {
				bandLogE2[nb*c+i] += 0.5 * float32(LM)
			}
		}
	}
	e.computeMDCTs(shortBlocks, in, freq, C, LM)
	computeBandEnergies(freq, bandE, effEnd, C, LM)
	amp2Log2(effEnd, end, bandE, bandLogE, C)
	if !secondMdct {
		copy(bandLogE2, bandLogE)
	}

	var temporalVBR float32
	{
		follow := float32(-10)
		var frameAvg, offset float32
		if shortBlocks != 0 {
			offset = 0.5 * float32(LM)
		}
		for i := start; i < end; i++ {
			follow = max(follow-1, bandLogE[i]-offset)
			if C == 2 {
				follow = max(follow, bandLogE[i+nb]-offset)
			}
			frameAvg += follow
		}
		frameAvg /= float32(end - start)
		temporalVBR = min(3.0, max(-1.5, frameAvg-e.specAvg))
		e.specAvg += 0.02 * temporalVBR
	}

	if LM > 0 && enc.tell()+3 <= totalBits && isTransient == 0 && e.complexity >= 5 {
		if patchTransientDecision(bandLogE, e.oldBandE, start, end, C) {
			isTransient = 1
			shortBlocks = M
			e.computeMDCTs(shortBlocks, in, freq, C, LM)
			computeBandEnergies(freq, bandE, effEnd, C, LM)
			amp2Log2(effEnd, end, bandE, bandLogE, C)
			for c := 0; c < C; c++ {
				for i := 0; i < end; i++ {
					bandLogE2[nb*c+i] += 0.5 * float32(LM)
				}
			}
			tfEstimate = 0.2
		}
	}

	if LM > 0 && enc.tell()+3 <= totalBits {
		enc.encodeBitLogp(isTransient, 3)
	}

	X := make([]float32, C*N)
	normaliseBands(freq, X, bandE, effEnd, C, M)

	offsets := make([]int, nb)
	importance := make([]int, nb)
	spreadWeight := make([]int, nb)
	totBoost := 0
	maxDepth := dynallocOffsets(bandLogE, bandLogE2, e.oldBandE, start, end, C, LM, effectiveBytes, isTransient,
		e.vbr, e.constrainedVBR, offsets, importance, spreadWeight, &totBoost, toneFreq, toneishness, &e.analysis)

	tfRes := make([]int, nb)
	tfSelect := 0
	switch {
	case effectiveBytes >= 15*C && !hybrid && e.complexity >= 2 && toneishness < 0.98:
		lambda := max(80, 20480/effectiveBytes+2)
		tfSelect = tfAnalysis(effEnd, isTransient, tfRes, lambda, X, N, LM, tfEstimate, tfChan, importance)
		for i := effEnd; i < end; i++ {
			tfRes[i] = tfRes[effEnd-1]
		}
	case hybrid && weakTransient != 0:
		for i := 0; i < end; i++ {
			tfRes[i] = 1
		}
	case hybrid && effectiveBytes < 15 && e.silkInfoSignalType != 2:
		for i := 0; i < end; i++ {
			tfRes[i] = 0
		}
		tfSelect = isTransient
	default:
		for i := 0; i < end; i++ {
			tfRes[i] = isTransient
		}
	}

	for c := 0; c < C; c++ {
		for i := start; i < end; i++ {
			if absf(bandLogE[i+c*nb]-e.oldBandE[i+c*nb]) < 2 {
				bandLogE[i+c*nb] -= 0.25 * e.energyError[i+c*nb]
			}
		}
	}

	errorArr := make([]float32, C*nb)
	quantCoarseEnergy(start, end, effEnd, bandLogE, e.oldBandE, totalBits, errorArr, enc,
		C, LM, nbAvailableBytes, e.forceIntra, &e.delayedIntra, e.complexity >= 4)

	tfEncode(start, end, isTransient, tfRes, LM, tfSelect, enc)

	spread := spreadNormal
	if enc.tell()+4 <= totalBits {
		if hybrid {
			if e.complexity == 0 {
				spread = spreadNone
			} else if isTransient == 0 {
				spread = spreadAggr
			}
		} else if shortBlocks != 0 || e.complexity < 3 || nbCompressedBytes < 10*C {
			if e.complexity == 0 {
				spread = spreadNone
			}
		} else {
			spread = spreadingDecision(X, &e.tonalAverage, e.spreadDecision,
				&e.hfAverage, &e.tapsetDecision, false, effEnd, C, M, spreadWeight)
		}
		e.spreadDecision = spread
		enc.encodeICDF(spread, celtSpreadICDF, 5)
	} else {
		e.spreadDecision = spreadNormal
	}

	caps := make([]int, nb)
	initCaps(caps, LM, C)
	dynallocLogp := 6
	totalBitsFrac := totalBits << bitRes
	totalBoost := 0
	tellF := enc.tellFrac()
	for i := start; i < end; i++ {
		width := C * (int(celtEBands[i+1]) - int(celtEBands[i])) << LM
		quanta := min(width<<bitRes, max(6<<bitRes, width))
		loopLogp := dynallocLogp
		boost := 0
		j := 0
		for tellF+(loopLogp<<bitRes) < totalBitsFrac-totalBoost && boost < caps[i] {
			flag := 0
			if j < offsets[i] {
				flag = 1
			}
			enc.encodeBitLogp(flag, uint(loopLogp))
			tellF = enc.tellFrac()
			if flag == 0 {
				break
			}
			boost += quanta
			totalBoost += quanta
			loopLogp = 1
			j++
		}
		if j != 0 {
			dynallocLogp = max(2, dynallocLogp-1)
		}
		offsets[i] = boost
	}

	dualStereo := 0
	if C == 2 {
		if LM != 0 {
			dualStereo = stereoAnalysis(X, LM, N)
		}
		e.intensity = hysteresisDecision(float32(equivRate/1000),
			celtIntensityThresholds, celtIntensityHysteresis, 21, e.intensity)
		e.intensity = min(end, max(start, e.intensity))
	}

	allocTrim := 5
	if tellF+(6<<bitRes) <= totalBitsFrac-totalBoost {
		allocTrim = allocTrimAnalysis(X, bandLogE, end, LM, C, N, tfEstimate, e.intensity, equivRate, &e.stereoSaving, &e.analysis)
		enc.encodeICDF(allocTrim, celtTrimICDF, 7)
		tellF = enc.tellFrac()
	}

	if vbrRate > 0 {
		lmDiff := celtMaxLM - LM
		minAllowed := ((tellF + totalBoost + (1 << (bitRes + 3)) - 1) >> (bitRes + 3)) + 2
		if hybrid {
			minAllowed = max(minAllowed, (tell0Frac+(37<<bitRes)+totalBoost+(1<<(bitRes+3))-1)>>(bitRes+3))
		}
		nbCompressedBytes = min(nbCompressedBytes, maxFrameBytes>>(3-LM))
		var baseTarget int
		if !hybrid {
			baseTarget = vbrRate - ((40*C + 20) << bitRes)
		} else {
			baseTarget = max(0, vbrRate-((9*C+4)<<bitRes))
		}
		if e.constrainedVBR {
			baseTarget += e.vbrOffset >> lmDiff
		}
		var target int
		if !hybrid {
			target = computeVBR(baseTarget, LM, e.bitrate, e.lastCodedBands, C, e.intensity,
				e.constrainedVBR, e.stereoSaving, totBoost, tfEstimate, maxDepth, temporalVBR,
				&e.analysis, pitchChange)
		} else {
			target = baseTarget
			if e.silkInfoOffset < 100 {
				target += 12 << bitRes >> (3 - LM)
			}
			if e.silkInfoOffset > 100 {
				target -= 18 << bitRes >> (3 - LM)
			}
			target += int((tfEstimate - 0.25) * float32(50<<bitRes))
			if tfEstimate > 0.7 {
				target = max(target, 50<<bitRes)
			}
		}
		target += tellF
		nbAvailableBytes := (target + (1 << (bitRes + 2))) >> (bitRes + 3)
		nbAvailableBytes = max(minAllowed, nbAvailableBytes)
		nbAvailableBytes = min(nbCompressedBytes, nbAvailableBytes)
		delta := target - vbrRate
		target = nbAvailableBytes << (bitRes + 3)
		if silence != 0 {
			nbAvailableBytes = 2
			target = 2 * 8 << bitRes
			delta = 0
		}
		var alpha float32
		if e.vbrCount < 970 {
			e.vbrCount++
			alpha = 1 / float32(e.vbrCount+20)
		} else {
			alpha = 0.001
		}
		if e.constrainedVBR {
			e.vbrReservoir += target - vbrRate
			e.vbrDrift += int(alpha * float32((delta<<lmDiff)-e.vbrOffset-e.vbrDrift))
			e.vbrOffset = -e.vbrDrift
			if e.vbrReservoir < 0 {
				adjust := -e.vbrReservoir / (8 << bitRes)
				if silence == 0 {
					nbAvailableBytes += adjust
				}
				e.vbrReservoir = 0
			}
		}
		nbCompressedBytes = min(nbCompressedBytes, nbAvailableBytes)
		enc.shrink(nbCompressedBytes)
	}
	totalBits = nbCompressedBytes * 8

	bits := (nbCompressedBytes*8)<<bitRes - enc.tellFrac() - 1
	antiCollapseRsv := 0
	if isTransient != 0 && LM >= 2 && bits >= (LM+2)<<bitRes {
		antiCollapseRsv = 1 << bitRes
	}
	bits -= antiCollapseRsv

	pulses := make([]int, nb)
	fineQuant := make([]int, nb)
	finePriority := make([]int, nb)
	balance := 0
	signalBandwidth := end - 1
	if e.analysis.valid {
		var minBandwidth int
		switch {
		case equivRate < 32000*C:
			minBandwidth = 13
		case equivRate < 48000*C:
			minBandwidth = 16
		case equivRate < 60000*C:
			minBandwidth = 18
		case equivRate < 80000*C:
			minBandwidth = 19
		default:
			minBandwidth = 20
		}
		signalBandwidth = max(e.analysis.bandwidth, minBandwidth)
	}
	codedBands := cltComputeAllocation(start, end, offsets, caps, allocTrim, &e.intensity, &dualStereo,
		bits, &balance, pulses, fineQuant, finePriority, C, LM, enc, nil, true, e.lastCodedBands, signalBandwidth)
	if e.lastCodedBands != 0 {
		e.lastCodedBands = min(e.lastCodedBands+1, max(e.lastCodedBands-1, codedBands))
	} else {
		e.lastCodedBands = codedBands
	}

	quantFineEnergy(start, end, e.oldBandE, errorArr, fineQuant, enc, C)
	clear(e.energyError)

	collapseMasks := make([]byte, C*nb)
	var Y []float32
	if C == 2 {
		Y = X[N:]
	}
	quantAllBands(start, end, X, Y, collapseMasks, bandE, pulses, shortBlocks, spread,
		dualStereo, e.intensity, tfRes, nbCompressedBytes*(8<<bitRes)-antiCollapseRsv, balance, enc, nil,
		LM, codedBands, &e.rng, e.complexity, b2i(C == 1), e.iy, e.u)

	antiCollapseOn := 0
	if antiCollapseRsv > 0 {
		if e.consecTransient < 2 {
			antiCollapseOn = 1
		}
		enc.encodeRawBits(uint32(antiCollapseOn), 1)
	}
	quantEnergyFinalise(start, end, e.oldBandE, errorArr, fineQuant, finePriority,
		nbCompressedBytes*8-enc.tell(), enc, C)

	for c := 0; c < C; c++ {
		for i := start; i < end; i++ {
			e.energyError[i+c*nb] = max(-0.5, min(0.5, errorArr[i+c*nb]))
		}
	}
	if silence != 0 {
		for i := 0; i < C*nb; i++ {
			e.oldBandE[i] = -28
		}
	}

	e.updateEnergyState(C, isTransient, start, end)
	e.rng = enc.rng

	for c := 0; c < C; c++ {
		copy(e.preHistory[c], in[c][N:N+overlap])
	}

	enc.done()
	return nbCompressedBytes
}

func (e *celtEncoder) updateEnergyState(C, isTransient, start, end int) {
	nb := celtNBands
	if isTransient == 0 {
		copy(e.oldLogE2, e.oldLogE[:C*nb])
		copy(e.oldLogE, e.oldBandE[:C*nb])
	} else {
		for i := 0; i < C*nb; i++ {
			e.oldLogE[i] = min(e.oldLogE[i], e.oldBandE[i])
		}
	}
	for c := 0; c < C; c++ {
		for i := 0; i < start; i++ {
			e.oldBandE[c*nb+i] = 0
			e.oldLogE[c*nb+i] = -28
			e.oldLogE2[c*nb+i] = -28
		}
		for i := end; i < nb; i++ {
			e.oldBandE[c*nb+i] = 0
			e.oldLogE[c*nb+i] = -28
			e.oldLogE2[c*nb+i] = -28
		}
	}
	if isTransient != 0 {
		e.consecTransient++
	} else {
		e.consecTransient = 0
	}
}
