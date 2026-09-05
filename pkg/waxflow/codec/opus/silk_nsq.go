package opus

const quantLevelAdjustQ10 = 80

func nsqShortPrediction(buf []int32, pos int, coef []int16, order int) int32 {
	out := int32(order >> 1)
	for j := 0; j < order; j++ {
		out = silkSMLAWB(out, buf[pos-j], int32(coef[j]))
	}
	return out
}

func nsqNoiseShapeFeedbackLoop(data0 int32, data1 []int32, coef []int16, order int) int32 {
	tmp2 := data0
	tmp1 := data1[0]
	data1[0] = tmp2
	out := int32(order >> 1)
	out = silkSMLAWB(out, tmp2, int32(coef[0]))
	for j := 2; j < order; j += 2 {
		tmp2 = data1[j-1]
		data1[j-1] = tmp1
		out = silkSMLAWB(out, tmp1, int32(coef[j-1]))
		tmp1 = data1[j]
		data1[j] = tmp2
		out = silkSMLAWB(out, tmp2, int32(coef[j]))
	}
	data1[order-1] = tmp1
	out = silkSMLAWB(out, tmp1, int32(coef[order-1]))
	return silkLSHIFT32(out, 1)
}

func (ch *silkEncoderChannel) nsq(NSQ *silkNSQState, psIndices *silkIndices, x16 []int16, pulses []int8,
	predCoefQ12 *[2][silkMaxLPCOrder]int16, ltpCoefQ14 []int16, arQ13 []int16,
	harmShapeGainQ14, tiltQ14 []int32, lfShpQ14 []int32, gainsQ16 []int32, pitchL []int,
	lambdaQ10 int32, ltpScaleQ14 int32) {

	NSQ.randSeed = int32(psIndices.Seed)
	lag := NSQ.lagPrev

	offsetQ10 := int32(silk_Quantization_Offsets_Q10[psIndices.signalType>>1][psIndices.quantOffsetType])
	lsfInterpolationFlag := 0
	if psIndices.NLSFInterpCoefQ2 != 4 {
		lsfInterpolationFlag = 1
	}

	sLTPQ15 := make([]int32, ch.ltpMemLength+ch.frameLength)
	sLTP := make([]int16, ch.ltpMemLength+ch.frameLength)
	xScQ10 := make([]int32, ch.subfrLength)

	NSQ.sLTPShpBufIdx = ch.ltpMemLength
	NSQ.sLTPBufIdx = ch.ltpMemLength
	pxqOff := ch.ltpMemLength
	xOff, pulsesOff := 0, 0
	for k := 0; k < ch.nbSubfr; k++ {
		aQ12 := predCoefQ12[(k>>1)|(1-lsfInterpolationFlag)][:]
		bQ14 := ltpCoefQ14[k*silkLTPOrder:]
		arShpQ13 := arQ13[k*maxShapeLPCOrder:]

		harmShapeFIRPackedQ14 := silkRSHIFT(harmShapeGainQ14[k], 2)
		harmShapeFIRPackedQ14 |= silkLSHIFT(silkRSHIFT(harmShapeGainQ14[k], 1), 16)

		NSQ.rewhiteFlag = 0
		if psIndices.signalType == typeVoiced {
			lag = pitchL[k]
			if k&(3-(lsfInterpolationFlag<<1)) == 0 {
				startIdx := ch.ltpMemLength - lag - ch.predictLPCOrder - silkLTPOrder/2
				silkLPCAnalysisFilter(sLTP[startIdx:], NSQ.xq[startIdx+k*ch.subfrLength:],
					aQ12, ch.ltpMemLength-startIdx, ch.predictLPCOrder)
				NSQ.rewhiteFlag = 1
				NSQ.sLTPBufIdx = ch.ltpMemLength
			}
		}

		ch.nsqScaleStates(NSQ, x16[xOff:], xScQ10, sLTP, sLTPQ15, k, ltpScaleQ14, gainsQ16, pitchL, int(psIndices.signalType))
		noiseShapeQuantizer(NSQ, int(psIndices.signalType), xScQ10, pulses[pulsesOff:], NSQ.xq[pxqOff:], sLTPQ15,
			aQ12, bQ14, arShpQ13, lag, harmShapeFIRPackedQ14, tiltQ14[k], lfShpQ14[k], gainsQ16[k], lambdaQ10,
			offsetQ10, ch.subfrLength, ch.shapingLPCOrder, ch.predictLPCOrder)

		xOff += ch.subfrLength
		pulsesOff += ch.subfrLength
		pxqOff += ch.subfrLength
	}

	NSQ.lagPrev = pitchL[ch.nbSubfr-1]
	copy(NSQ.xq[:ch.ltpMemLength], NSQ.xq[ch.frameLength:ch.frameLength+ch.ltpMemLength])
	copy(NSQ.sLTPShpQ14[:ch.ltpMemLength], NSQ.sLTPShpQ14[ch.frameLength:ch.frameLength+ch.ltpMemLength])
}

func noiseShapeQuantizer(NSQ *silkNSQState, signalType int, xScQ10 []int32, pulses []int8, xq []int16,
	sLTPQ15 []int32, aQ12, bQ14, arShpQ13 []int16, lag int, harmShapeFIRPackedQ14 int32,
	tiltQ14, lfShpQ14, gainQ16, lambdaQ10, offsetQ10 int32, length, shapingLPCOrder, predictLPCOrder int) {

	shpLagPtr := NSQ.sLTPShpBufIdx - lag + harmShapeFIRTaps/2
	predLagPtr := NSQ.sLTPBufIdx - lag + silkLTPOrder/2
	gainQ10 := silkRSHIFT(gainQ16, 6)

	psLPC := nsqLPCBufLength - 1

	for i := 0; i < length; i++ {
		NSQ.randSeed = silkRAND(NSQ.randSeed)

		lpcPredQ10 := nsqShortPrediction(NSQ.sLPCQ14[:], psLPC, aQ12, predictLPCOrder)

		var ltpPredQ13 int32
		if signalType == typeVoiced {
			ltpPredQ13 = 2
			ltpPredQ13 = silkSMLAWB(ltpPredQ13, sLTPQ15[predLagPtr], int32(bQ14[0]))
			ltpPredQ13 = silkSMLAWB(ltpPredQ13, sLTPQ15[predLagPtr-1], int32(bQ14[1]))
			ltpPredQ13 = silkSMLAWB(ltpPredQ13, sLTPQ15[predLagPtr-2], int32(bQ14[2]))
			ltpPredQ13 = silkSMLAWB(ltpPredQ13, sLTPQ15[predLagPtr-3], int32(bQ14[3]))
			ltpPredQ13 = silkSMLAWB(ltpPredQ13, sLTPQ15[predLagPtr-4], int32(bQ14[4]))
			predLagPtr++
		}

		nARQ12 := nsqNoiseShapeFeedbackLoop(NSQ.sDiffShpQ14, NSQ.sAR2Q14[:], arShpQ13, shapingLPCOrder)
		nARQ12 = silkSMLAWB(nARQ12, NSQ.sLFARShpQ14, tiltQ14)

		nLFQ12 := silkSMULWB(NSQ.sLTPShpQ14[NSQ.sLTPShpBufIdx-1], lfShpQ14)
		nLFQ12 = silkSMLAWT(nLFQ12, NSQ.sLFARShpQ14, lfShpQ14)

		tmp1 := silkSUB32ovflw(silkLSHIFT32(lpcPredQ10, 2), nARQ12)
		tmp1 = silkSUB32ovflw(tmp1, nLFQ12)
		if lag > 0 {
			nLTPQ13 := silkSMULWB(silkADDSAT32(NSQ.sLTPShpQ14[shpLagPtr], NSQ.sLTPShpQ14[shpLagPtr-2]), harmShapeFIRPackedQ14)
			nLTPQ13 = silkSMLAWT(nLTPQ13, NSQ.sLTPShpQ14[shpLagPtr-1], harmShapeFIRPackedQ14)
			nLTPQ13 = silkLSHIFT(nLTPQ13, 1)
			shpLagPtr++
			tmp2 := ltpPredQ13 - nLTPQ13
			tmp1 = silkADD32ovflw(tmp2, silkLSHIFT32(tmp1, 1))
			tmp1 = silkRSHIFTROUND(tmp1, 3)
		} else {
			tmp1 = silkRSHIFTROUND(tmp1, 2)
		}

		rQ10 := xScQ10[i] - tmp1
		if NSQ.randSeed < 0 {
			rQ10 = -rQ10
		}
		rQ10 = silkLIMIT(rQ10, -(31 << 10), 30<<10)

		q1Q10, q2Q10, rd1Q20, rd2Q20 := nsqQuantCandidates(rQ10, offsetQ10, lambdaQ10)
		rrQ10 := rQ10 - q1Q10
		rd1Q20 = silkSMLABB(rd1Q20, rrQ10, rrQ10)
		rrQ10 = rQ10 - q2Q10
		rd2Q20 = silkSMLABB(rd2Q20, rrQ10, rrQ10)
		if rd2Q20 < rd1Q20 {
			q1Q10 = q2Q10
		}

		pulses[i] = int8(silkRSHIFTROUND(q1Q10, 10))

		excQ14 := silkLSHIFT(q1Q10, 4)
		if NSQ.randSeed < 0 {
			excQ14 = -excQ14
		}

		lpcExcQ14 := excQ14 + silkLSHIFT32(ltpPredQ13, 1)
		xqQ14 := silkADD32ovflw(lpcExcQ14, silkLSHIFT32(lpcPredQ10, 4))

		xq[i] = int16(silkSAT16(silkRSHIFTROUND(silkSMULWW(xqQ14, gainQ10), 8)))

		psLPC++
		NSQ.sLPCQ14[psLPC] = xqQ14
		NSQ.sDiffShpQ14 = silkSUB32ovflw(xqQ14, silkLSHIFT32(xScQ10[i], 4))
		sLFARShpQ14 := silkSUB32ovflw(NSQ.sDiffShpQ14, silkLSHIFT32(nARQ12, 2))
		NSQ.sLFARShpQ14 = sLFARShpQ14
		NSQ.sLTPShpQ14[NSQ.sLTPShpBufIdx] = silkSUB32ovflw(sLFARShpQ14, silkLSHIFT32(nLFQ12, 2))
		sLTPQ15[NSQ.sLTPBufIdx] = silkLSHIFT(lpcExcQ14, 1)
		NSQ.sLTPShpBufIdx++
		NSQ.sLTPBufIdx++

		NSQ.randSeed = silkADD32ovflw(NSQ.randSeed, int32(pulses[i]))
	}

	copy(NSQ.sLPCQ14[:nsqLPCBufLength], NSQ.sLPCQ14[length:length+nsqLPCBufLength])
}

func nsqQuantCandidates(rQ10, offsetQ10, lambdaQ10 int32) (q1Q10, q2Q10, rd1, rd2 int32) {
	q1Q10 = rQ10 - offsetQ10
	q1Q0 := silkRSHIFT(q1Q10, 10)
	if lambdaQ10 > 2048 {
		rdoOffset := lambdaQ10/2 - 512
		switch {
		case q1Q10 > rdoOffset:
			q1Q0 = silkRSHIFT(q1Q10-rdoOffset, 10)
		case q1Q10 < -rdoOffset:
			q1Q0 = silkRSHIFT(q1Q10+rdoOffset, 10)
		case q1Q10 < 0:
			q1Q0 = -1
		default:
			q1Q0 = 0
		}
	}
	switch {
	case q1Q0 > 0:
		q1Q10 = silkLSHIFT(q1Q0, 10) - quantLevelAdjustQ10
		q1Q10 += offsetQ10
		q2Q10 = q1Q10 + 1024
		rd1 = silkSMULBB(q1Q10, lambdaQ10)
		rd2 = silkSMULBB(q2Q10, lambdaQ10)
	case q1Q0 == 0:
		q1Q10 = offsetQ10
		q2Q10 = q1Q10 + (1024 - quantLevelAdjustQ10)
		rd1 = silkSMULBB(q1Q10, lambdaQ10)
		rd2 = silkSMULBB(q2Q10, lambdaQ10)
	case q1Q0 == -1:
		q2Q10 = offsetQ10
		q1Q10 = q2Q10 - (1024 - quantLevelAdjustQ10)
		rd1 = silkSMULBB(-q1Q10, lambdaQ10)
		rd2 = silkSMULBB(q2Q10, lambdaQ10)
	default:
		q1Q10 = silkLSHIFT(q1Q0, 10) + quantLevelAdjustQ10
		q1Q10 += offsetQ10
		q2Q10 = q1Q10 + 1024
		rd1 = silkSMULBB(-q1Q10, lambdaQ10)
		rd2 = silkSMULBB(-q2Q10, lambdaQ10)
	}
	return
}

func (ch *silkEncoderChannel) nsqScaleStates(NSQ *silkNSQState, x16 []int16, xScQ10 []int32,
	sLTP []int16, sLTPQ15 []int32, subfr int, ltpScaleQ14 int32, gainsQ16 []int32, pitchL []int, signalType int) {

	lag := pitchL[subfr]
	invGainQ31 := silkINVERSE32varQ(silkMaxInt(gainsQ16[subfr], 1), 47)

	invGainQ26 := silkRSHIFTROUND(invGainQ31, 5)
	for i := 0; i < ch.subfrLength; i++ {
		xScQ10[i] = silkSMULWW(int32(x16[i]), invGainQ26)
	}

	if NSQ.rewhiteFlag != 0 {
		if subfr == 0 {
			invGainQ31 = silkLSHIFT(silkSMULWB(invGainQ31, ltpScaleQ14), 2)
		}
		for i := NSQ.sLTPBufIdx - lag - silkLTPOrder/2; i < NSQ.sLTPBufIdx; i++ {
			sLTPQ15[i] = silkSMULWB(invGainQ31, int32(sLTP[i]))
		}
	}

	if gainsQ16[subfr] != NSQ.prevGainQ16 {
		gainAdjQ16 := silkDIV32varQ(NSQ.prevGainQ16, gainsQ16[subfr], 16)
		for i := NSQ.sLTPShpBufIdx - ch.ltpMemLength; i < NSQ.sLTPShpBufIdx; i++ {
			NSQ.sLTPShpQ14[i] = silkSMULWW(gainAdjQ16, NSQ.sLTPShpQ14[i])
		}
		if signalType == typeVoiced && NSQ.rewhiteFlag == 0 {
			for i := NSQ.sLTPBufIdx - lag - silkLTPOrder/2; i < NSQ.sLTPBufIdx; i++ {
				sLTPQ15[i] = silkSMULWW(gainAdjQ16, sLTPQ15[i])
			}
		}
		NSQ.sLFARShpQ14 = silkSMULWW(gainAdjQ16, NSQ.sLFARShpQ14)
		NSQ.sDiffShpQ14 = silkSMULWW(gainAdjQ16, NSQ.sDiffShpQ14)
		for i := 0; i < nsqLPCBufLength; i++ {
			NSQ.sLPCQ14[i] = silkSMULWW(gainAdjQ16, NSQ.sLPCQ14[i])
		}
		for i := 0; i < maxShapeLPCOrder; i++ {
			NSQ.sAR2Q14[i] = silkSMULWW(gainAdjQ16, NSQ.sAR2Q14[i])
		}
		NSQ.prevGainQ16 = gainsQ16[subfr]
	}
}

type nsqDelDecState struct {
	sLPCQ14   [silkMaxSubfrLen + nsqLPCBufLength]int32
	RandState [decisionDelay]int32
	QQ10      [decisionDelay]int32
	XqQ14     [decisionDelay]int32
	PredQ15   [decisionDelay]int32
	ShapeQ14  [decisionDelay]int32
	sAR2Q14   [maxShapeLPCOrder]int32
	LFARQ14   int32
	DiffQ14   int32
	Seed      int32
	SeedInit  int32
	RDQ10     int32
}

type nsqSampleState struct {
	QQ10       int32
	RDQ10      int32
	xqQ14      int32
	LFARQ14    int32
	DiffQ14    int32
	sLTPShpQ14 int32
	LPCExcQ14  int32
}

func (ch *silkEncoderChannel) nsqDelDec(NSQ *silkNSQState, psIndices *silkIndices, x16 []int16, pulses []int8,
	predCoefQ12 *[2][silkMaxLPCOrder]int16, ltpCoefQ14 []int16, arQ13 []int16,
	harmShapeGainQ14, tiltQ14 []int32, lfShpQ14 []int32, gainsQ16 []int32, pitchL []int,
	lambdaQ10 int32, ltpScaleQ14 int32) {

	lag := NSQ.lagPrev

	psDelDec := make([]nsqDelDecState, ch.nStatesDelayedDecision)
	for k := range psDelDec {
		psDD := &psDelDec[k]
		psDD.Seed = (int32(k) + int32(psIndices.Seed)) & 3
		psDD.SeedInit = psDD.Seed
		psDD.LFARQ14 = NSQ.sLFARShpQ14
		psDD.DiffQ14 = NSQ.sDiffShpQ14
		psDD.ShapeQ14[0] = NSQ.sLTPShpQ14[ch.ltpMemLength-1]
		copy(psDD.sLPCQ14[:nsqLPCBufLength], NSQ.sLPCQ14[:nsqLPCBufLength])
		psDD.sAR2Q14 = NSQ.sAR2Q14
	}

	offsetQ10 := int32(silk_Quantization_Offsets_Q10[psIndices.signalType>>1][psIndices.quantOffsetType])
	smplBufIdx := 0
	dd := decisionDelay
	if dd > ch.subfrLength {
		dd = ch.subfrLength
	}
	if psIndices.signalType == typeVoiced {
		for k := 0; k < ch.nbSubfr; k++ {
			if v := pitchL[k] - silkLTPOrder/2 - 1; v < dd {
				dd = v
			}
		}
	} else if lag > 0 {
		if v := lag - silkLTPOrder/2 - 1; v < dd {
			dd = v
		}
	}

	lsfInterpolationFlag := 0
	if psIndices.NLSFInterpCoefQ2 != 4 {
		lsfInterpolationFlag = 1
	}

	sLTPQ15 := make([]int32, ch.ltpMemLength+ch.frameLength)
	sLTP := make([]int16, ch.ltpMemLength+ch.frameLength)
	xScQ10 := make([]int32, ch.subfrLength)
	var delayedGainQ10 [decisionDelay]int32

	pxqOff := ch.ltpMemLength
	NSQ.sLTPShpBufIdx = ch.ltpMemLength
	NSQ.sLTPBufIdx = ch.ltpMemLength
	subfr := 0
	xOff, pulsesOff := 0, 0
	for k := 0; k < ch.nbSubfr; k++ {
		aQ12 := predCoefQ12[(k>>1)|(1-lsfInterpolationFlag)][:]
		bQ14 := ltpCoefQ14[k*silkLTPOrder:]
		arShpQ13 := arQ13[k*maxShapeLPCOrder:]

		harmShapeFIRPackedQ14 := silkRSHIFT(harmShapeGainQ14[k], 2)
		harmShapeFIRPackedQ14 |= silkLSHIFT(silkRSHIFT(harmShapeGainQ14[k], 1), 16)

		NSQ.rewhiteFlag = 0
		if psIndices.signalType == typeVoiced {
			lag = pitchL[k]
			if k&(3-(lsfInterpolationFlag<<1)) == 0 {
				if k == 2 {
					rdMin := psDelDec[0].RDQ10
					winnerInd := 0
					for i := 1; i < ch.nStatesDelayedDecision; i++ {
						if psDelDec[i].RDQ10 < rdMin {
							rdMin = psDelDec[i].RDQ10
							winnerInd = i
						}
					}
					for i := range psDelDec {
						if i != winnerInd {
							psDelDec[i].RDQ10 += silkInt32Max >> 4
						}
					}
					psDD := &psDelDec[winnerInd]
					lastSmpleIdx := smplBufIdx + dd
					for i := 0; i < dd; i++ {
						lastSmpleIdx = (lastSmpleIdx - 1) % decisionDelay
						if lastSmpleIdx < 0 {
							lastSmpleIdx += decisionDelay
						}
						pulses[pulsesOff+i-dd] = int8(silkRSHIFTROUND(psDD.QQ10[lastSmpleIdx], 10))
						NSQ.xq[pxqOff+i-dd] = int16(silkSAT16(silkRSHIFTROUND(
							silkSMULWW(psDD.XqQ14[lastSmpleIdx], gainsQ16[1]), 14)))
						NSQ.sLTPShpQ14[NSQ.sLTPShpBufIdx-dd+i] = psDD.ShapeQ14[lastSmpleIdx]
					}
					subfr = 0
				}
				startIdx := ch.ltpMemLength - lag - ch.predictLPCOrder - silkLTPOrder/2
				silkLPCAnalysisFilter(sLTP[startIdx:], NSQ.xq[startIdx+k*ch.subfrLength:],
					aQ12, ch.ltpMemLength-startIdx, ch.predictLPCOrder)
				NSQ.sLTPBufIdx = ch.ltpMemLength
				NSQ.rewhiteFlag = 1
			}
		}

		ch.nsqDelDecScaleStates(NSQ, psDelDec, x16[xOff:], xScQ10, sLTP, sLTPQ15, k,
			ch.nStatesDelayedDecision, ltpScaleQ14, gainsQ16, pitchL, int(psIndices.signalType), dd)
		ch.noiseShapeQuantizerDelDec(NSQ, psDelDec, int(psIndices.signalType), xScQ10, pulses, pulsesOff, NSQ.xq[:], pxqOff, sLTPQ15,
			delayedGainQ10[:], aQ12, bQ14, arShpQ13, lag, harmShapeFIRPackedQ14, tiltQ14[k], lfShpQ14[k],
			gainsQ16[k], lambdaQ10, offsetQ10, ch.subfrLength, subfr, ch.shapingLPCOrder,
			ch.predictLPCOrder, ch.warpingQ16, ch.nStatesDelayedDecision, &smplBufIdx, dd)
		subfr++

		xOff += ch.subfrLength
		pulsesOff += ch.subfrLength
		pxqOff += ch.subfrLength
	}

	rdMin := psDelDec[0].RDQ10
	winnerInd := 0
	for k := 1; k < ch.nStatesDelayedDecision; k++ {
		if psDelDec[k].RDQ10 < rdMin {
			rdMin = psDelDec[k].RDQ10
			winnerInd = k
		}
	}

	psDD := &psDelDec[winnerInd]
	psIndices.Seed = int8(psDD.SeedInit)
	lastSmpleIdx := smplBufIdx + dd
	gainQ10 := silkRSHIFT(gainsQ16[ch.nbSubfr-1], 6)
	for i := 0; i < dd; i++ {
		lastSmpleIdx = (lastSmpleIdx - 1) % decisionDelay
		if lastSmpleIdx < 0 {
			lastSmpleIdx += decisionDelay
		}
		pulses[pulsesOff+i-dd] = int8(silkRSHIFTROUND(psDD.QQ10[lastSmpleIdx], 10))
		NSQ.xq[pxqOff+i-dd] = int16(silkSAT16(silkRSHIFTROUND(
			silkSMULWW(psDD.XqQ14[lastSmpleIdx], gainQ10), 8)))
		NSQ.sLTPShpQ14[NSQ.sLTPShpBufIdx-dd+i] = psDD.ShapeQ14[lastSmpleIdx]
	}
	copy(NSQ.sLPCQ14[:nsqLPCBufLength], psDD.sLPCQ14[ch.subfrLength:ch.subfrLength+nsqLPCBufLength])
	NSQ.sAR2Q14 = psDD.sAR2Q14

	NSQ.sLFARShpQ14 = psDD.LFARQ14
	NSQ.sDiffShpQ14 = psDD.DiffQ14
	NSQ.lagPrev = pitchL[ch.nbSubfr-1]

	copy(NSQ.xq[:ch.ltpMemLength], NSQ.xq[ch.frameLength:ch.frameLength+ch.ltpMemLength])
	copy(NSQ.sLTPShpQ14[:ch.ltpMemLength], NSQ.sLTPShpQ14[ch.frameLength:ch.frameLength+ch.ltpMemLength])
}

func (ch *silkEncoderChannel) noiseShapeQuantizerDelDec(NSQ *silkNSQState, psDelDec []nsqDelDecState,
	signalType int, xQ10 []int32, pulses []int8, pulsesOff int, xq []int16, xqOff int, sLTPQ15 []int32, delayedGainQ10 []int32,
	aQ12, bQ14, arShpQ13 []int16, lag int, harmShapeFIRPackedQ14 int32, tiltQ14, lfShpQ14, gainQ16 int32,
	lambdaQ10, offsetQ10 int32, length, subfr, shapingLPCOrder, predictLPCOrder int,
	warpingQ16 int32, nStatesDelayedDecision int, smplBufIdx *int, dd int) {

	psSampleState := make([][2]nsqSampleState, nStatesDelayedDecision)

	shpLagPtr := NSQ.sLTPShpBufIdx - lag + harmShapeFIRTaps/2
	predLagPtr := NSQ.sLTPBufIdx - lag + silkLTPOrder/2
	gainQ10 := silkRSHIFT(gainQ16, 6)

	for i := 0; i < length; i++ {
		var ltpPredQ14 int32
		if signalType == typeVoiced {
			ltpPredQ14 = 2
			ltpPredQ14 = silkSMLAWB(ltpPredQ14, sLTPQ15[predLagPtr], int32(bQ14[0]))
			ltpPredQ14 = silkSMLAWB(ltpPredQ14, sLTPQ15[predLagPtr-1], int32(bQ14[1]))
			ltpPredQ14 = silkSMLAWB(ltpPredQ14, sLTPQ15[predLagPtr-2], int32(bQ14[2]))
			ltpPredQ14 = silkSMLAWB(ltpPredQ14, sLTPQ15[predLagPtr-3], int32(bQ14[3]))
			ltpPredQ14 = silkSMLAWB(ltpPredQ14, sLTPQ15[predLagPtr-4], int32(bQ14[4]))
			ltpPredQ14 = silkLSHIFT(ltpPredQ14, 1)
			predLagPtr++
		}

		var nLTPQ14 int32
		if lag > 0 {
			nLTPQ14 = silkSMULWB(silkADDSAT32(NSQ.sLTPShpQ14[shpLagPtr], NSQ.sLTPShpQ14[shpLagPtr-2]), harmShapeFIRPackedQ14)
			nLTPQ14 = silkSMLAWT(nLTPQ14, NSQ.sLTPShpQ14[shpLagPtr-1], harmShapeFIRPackedQ14)
			nLTPQ14 = ltpPredQ14 - silkLSHIFT32(nLTPQ14, 2)
			shpLagPtr++
		}

		for k := 0; k < nStatesDelayedDecision; k++ {
			psDD := &psDelDec[k]
			psSS := &psSampleState[k]

			psDD.Seed = silkRAND(psDD.Seed)

			psLPC := nsqLPCBufLength - 1 + i
			lpcPredQ14 := nsqShortPrediction(psDD.sLPCQ14[:], psLPC, aQ12, predictLPCOrder)
			lpcPredQ14 = silkLSHIFT(lpcPredQ14, 4)

			tmp2 := silkSMLAWB(psDD.DiffQ14, psDD.sAR2Q14[0], warpingQ16)
			tmp1 := silkSMLAWB(psDD.sAR2Q14[0], silkSUB32ovflw(psDD.sAR2Q14[1], tmp2), warpingQ16)
			psDD.sAR2Q14[0] = tmp2
			nARQ14 := int32(shapingLPCOrder >> 1)
			nARQ14 = silkSMLAWB(nARQ14, tmp2, int32(arShpQ13[0]))
			for j := 2; j < shapingLPCOrder; j += 2 {
				tmp2 = silkSMLAWB(psDD.sAR2Q14[j-1], silkSUB32ovflw(psDD.sAR2Q14[j], tmp1), warpingQ16)
				psDD.sAR2Q14[j-1] = tmp1
				nARQ14 = silkSMLAWB(nARQ14, tmp1, int32(arShpQ13[j-1]))
				tmp1 = silkSMLAWB(psDD.sAR2Q14[j], silkSUB32ovflw(psDD.sAR2Q14[j+1], tmp2), warpingQ16)
				psDD.sAR2Q14[j] = tmp2
				nARQ14 = silkSMLAWB(nARQ14, tmp2, int32(arShpQ13[j]))
			}
			psDD.sAR2Q14[shapingLPCOrder-1] = tmp1
			nARQ14 = silkSMLAWB(nARQ14, tmp1, int32(arShpQ13[shapingLPCOrder-1]))
			nARQ14 = silkLSHIFT(nARQ14, 1)
			nARQ14 = silkSMLAWB(nARQ14, psDD.LFARQ14, tiltQ14)
			nARQ14 = silkLSHIFT(nARQ14, 2)

			nLFQ14 := silkSMULWB(psDD.ShapeQ14[*smplBufIdx], lfShpQ14)
			nLFQ14 = silkSMLAWT(nLFQ14, psDD.LFARQ14, lfShpQ14)
			nLFQ14 = silkLSHIFT(nLFQ14, 2)

			tmp1 = silkADDSAT32(nARQ14, nLFQ14)
			tmp2 = silkADD32ovflw(nLTPQ14, lpcPredQ14)
			tmp1 = silkSUBSAT32(tmp2, tmp1)
			tmp1 = silkRSHIFTROUND(tmp1, 4)

			rQ10 := xQ10[i] - tmp1
			if psDD.Seed < 0 {
				rQ10 = -rQ10
			}
			rQ10 = silkLIMIT(rQ10, -(31 << 10), 30<<10)

			q1Q10, q2Q10, rd1, rd2 := nsqQuantCandidates(rQ10, offsetQ10, lambdaQ10)
			rrQ10 := rQ10 - q1Q10
			rd1Q10 := silkRSHIFT(silkSMLABB(rd1, rrQ10, rrQ10), 10)
			rrQ10 = rQ10 - q2Q10
			rd2Q10 := silkRSHIFT(silkSMLABB(rd2, rrQ10, rrQ10), 10)

			if rd1Q10 < rd2Q10 {
				psSS[0].RDQ10 = psDD.RDQ10 + rd1Q10
				psSS[1].RDQ10 = psDD.RDQ10 + rd2Q10
				psSS[0].QQ10 = q1Q10
				psSS[1].QQ10 = q2Q10
			} else {
				psSS[0].RDQ10 = psDD.RDQ10 + rd2Q10
				psSS[1].RDQ10 = psDD.RDQ10 + rd1Q10
				psSS[0].QQ10 = q2Q10
				psSS[1].QQ10 = q1Q10
			}

			for c := 0; c < 2; c++ {
				excQ14 := silkLSHIFT32(psSS[c].QQ10, 4)
				if psDD.Seed < 0 {
					excQ14 = -excQ14
				}
				lpcExcQ14 := excQ14 + ltpPredQ14
				xqQ14 := silkADD32ovflw(lpcExcQ14, lpcPredQ14)

				psSS[c].DiffQ14 = silkSUB32ovflw(xqQ14, silkLSHIFT32(xQ10[i], 4))
				sLFARShpQ14 := silkSUB32ovflw(psSS[c].DiffQ14, nARQ14)
				psSS[c].sLTPShpQ14 = silkSUBSAT32(sLFARShpQ14, nLFQ14)
				psSS[c].LFARQ14 = sLFARShpQ14
				psSS[c].LPCExcQ14 = lpcExcQ14
				psSS[c].xqQ14 = xqQ14
			}
		}

		*smplBufIdx = (*smplBufIdx - 1) % decisionDelay
		if *smplBufIdx < 0 {
			*smplBufIdx += decisionDelay
		}
		lastSmpleIdx := (*smplBufIdx + dd) % decisionDelay

		rdMin := psSampleState[0][0].RDQ10
		winnerInd := 0
		for k := 1; k < nStatesDelayedDecision; k++ {
			if psSampleState[k][0].RDQ10 < rdMin {
				rdMin = psSampleState[k][0].RDQ10
				winnerInd = k
			}
		}

		winnerRandState := psDelDec[winnerInd].RandState[lastSmpleIdx]
		for k := 0; k < nStatesDelayedDecision; k++ {
			if psDelDec[k].RandState[lastSmpleIdx] != winnerRandState {
				psSampleState[k][0].RDQ10 += silkInt32Max >> 4
				psSampleState[k][1].RDQ10 += silkInt32Max >> 4
			}
		}

		rdMax := psSampleState[0][0].RDQ10
		rdMin = psSampleState[0][1].RDQ10
		rdMaxInd, rdMinInd := 0, 0
		for k := 1; k < nStatesDelayedDecision; k++ {
			if psSampleState[k][0].RDQ10 > rdMax {
				rdMax = psSampleState[k][0].RDQ10
				rdMaxInd = k
			}
			if psSampleState[k][1].RDQ10 < rdMin {
				rdMin = psSampleState[k][1].RDQ10
				rdMinInd = k
			}
		}
		if rdMin < rdMax {
			dst, src := &psDelDec[rdMaxInd], &psDelDec[rdMinInd]
			copy(dst.sLPCQ14[i:], src.sLPCQ14[i:])
			dst.RandState = src.RandState
			dst.QQ10 = src.QQ10
			dst.XqQ14 = src.XqQ14
			dst.PredQ15 = src.PredQ15
			dst.ShapeQ14 = src.ShapeQ14
			dst.sAR2Q14 = src.sAR2Q14
			dst.LFARQ14 = src.LFARQ14
			dst.DiffQ14 = src.DiffQ14
			dst.Seed = src.Seed
			dst.SeedInit = src.SeedInit
			dst.RDQ10 = src.RDQ10
			psSampleState[rdMaxInd][0] = psSampleState[rdMinInd][1]
		}

		psDD := &psDelDec[winnerInd]
		if subfr > 0 || i >= dd {
			pulses[pulsesOff+i-dd] = int8(silkRSHIFTROUND(psDD.QQ10[lastSmpleIdx], 10))
			xq[xqOff+i-dd] = int16(silkSAT16(silkRSHIFTROUND(
				silkSMULWW(psDD.XqQ14[lastSmpleIdx], delayedGainQ10[lastSmpleIdx]), 8)))
			NSQ.sLTPShpQ14[NSQ.sLTPShpBufIdx-dd] = psDD.ShapeQ14[lastSmpleIdx]
			sLTPQ15[NSQ.sLTPBufIdx-dd] = psDD.PredQ15[lastSmpleIdx]
		}
		NSQ.sLTPShpBufIdx++
		NSQ.sLTPBufIdx++

		for k := 0; k < nStatesDelayedDecision; k++ {
			psDD := &psDelDec[k]
			psSS := &psSampleState[k][0]
			psDD.LFARQ14 = psSS.LFARQ14
			psDD.DiffQ14 = psSS.DiffQ14
			psDD.sLPCQ14[nsqLPCBufLength+i] = psSS.xqQ14
			psDD.XqQ14[*smplBufIdx] = psSS.xqQ14
			psDD.QQ10[*smplBufIdx] = psSS.QQ10
			psDD.PredQ15[*smplBufIdx] = silkLSHIFT32(psSS.LPCExcQ14, 1)
			psDD.ShapeQ14[*smplBufIdx] = psSS.sLTPShpQ14
			psDD.Seed = silkADD32ovflw(psDD.Seed, silkRSHIFTROUND(psSS.QQ10, 10))
			psDD.RandState[*smplBufIdx] = psDD.Seed
			psDD.RDQ10 = psSS.RDQ10
		}
		delayedGainQ10[*smplBufIdx] = gainQ10
	}

	for k := range psDelDec {
		psDD := &psDelDec[k]
		copy(psDD.sLPCQ14[:nsqLPCBufLength], psDD.sLPCQ14[length:length+nsqLPCBufLength])
	}
}

func (ch *silkEncoderChannel) nsqDelDecScaleStates(NSQ *silkNSQState, psDelDec []nsqDelDecState,
	x16 []int16, xScQ10 []int32, sLTP []int16, sLTPQ15 []int32, subfr, nStatesDelayedDecision int,
	ltpScaleQ14 int32, gainsQ16 []int32, pitchL []int, signalType int, dd int) {

	lag := pitchL[subfr]
	invGainQ31 := silkINVERSE32varQ(silkMaxInt(gainsQ16[subfr], 1), 47)

	invGainQ26 := silkRSHIFTROUND(invGainQ31, 5)
	for i := 0; i < ch.subfrLength; i++ {
		xScQ10[i] = silkSMULWW(int32(x16[i]), invGainQ26)
	}

	if NSQ.rewhiteFlag != 0 {
		if subfr == 0 {
			invGainQ31 = silkLSHIFT(silkSMULWB(invGainQ31, ltpScaleQ14), 2)
		}
		for i := NSQ.sLTPBufIdx - lag - silkLTPOrder/2; i < NSQ.sLTPBufIdx; i++ {
			sLTPQ15[i] = silkSMULWB(invGainQ31, int32(sLTP[i]))
		}
	}

	if gainsQ16[subfr] != NSQ.prevGainQ16 {
		gainAdjQ16 := silkDIV32varQ(NSQ.prevGainQ16, gainsQ16[subfr], 16)
		for i := NSQ.sLTPShpBufIdx - ch.ltpMemLength; i < NSQ.sLTPShpBufIdx; i++ {
			NSQ.sLTPShpQ14[i] = silkSMULWW(gainAdjQ16, NSQ.sLTPShpQ14[i])
		}
		if signalType == typeVoiced && NSQ.rewhiteFlag == 0 {
			for i := NSQ.sLTPBufIdx - lag - silkLTPOrder/2; i < NSQ.sLTPBufIdx-dd; i++ {
				sLTPQ15[i] = silkSMULWW(gainAdjQ16, sLTPQ15[i])
			}
		}
		for k := 0; k < nStatesDelayedDecision; k++ {
			psDD := &psDelDec[k]
			psDD.LFARQ14 = silkSMULWW(gainAdjQ16, psDD.LFARQ14)
			psDD.DiffQ14 = silkSMULWW(gainAdjQ16, psDD.DiffQ14)
			for i := 0; i < nsqLPCBufLength; i++ {
				psDD.sLPCQ14[i] = silkSMULWW(gainAdjQ16, psDD.sLPCQ14[i])
			}
			for i := 0; i < maxShapeLPCOrder; i++ {
				psDD.sAR2Q14[i] = silkSMULWW(gainAdjQ16, psDD.sAR2Q14[i])
			}
			for i := 0; i < decisionDelay; i++ {
				psDD.PredQ15[i] = silkSMULWW(gainAdjQ16, psDD.PredQ15[i])
				psDD.ShapeQ14[i] = silkSMULWW(gainAdjQ16, psDD.ShapeQ14[i])
			}
		}
		NSQ.prevGainQ16 = gainsQ16[subfr]
	}
}
