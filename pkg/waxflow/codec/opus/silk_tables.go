package opus

func silkFixConst(c float64, q uint) int32 {
	return int32(c*float64(int64(1)<<q) + 0.5)
}

const (
	silkMaxLPCOrder  = 16
	silkMinLPCOrder  = 10
	silkLTPOrder     = 5
	silkMaxNBSubfr   = 4
	silkSubFrameMS   = 5
	silkLTPMemMS     = 20
	silkMaxFrameLen  = 320
	silkMaxSubfrLen  = 80
	shellFrameLen    = 16
	log2ShellFrame   = 4
	maxNBShellBlocks = silkMaxFrameLen / shellFrameLen
	nRateLevels      = 10
	silkMaxPulses    = 16
	nlsfQuantMaxAmp  = 4
	quantLevelAdjQ10 = 80
	nLevelsQGain     = 64
	minQGainDB       = 2
	maxQGainDB       = 88
	minDeltaGain     = -4
	maxDeltaGain     = 36

	typeNoVoiceActivity = 0
	typeUnvoiced        = 1
	typeVoiced          = 2

	codeIndependently           = 0
	codeIndependentlyNoLTPScale = 1
	codeConditionally           = 2

	stereoInterpLenMS   = 8
	stereoQuantSubSteps = 5

	maxLPCStabilizeIters = 16
	bweAfterLossQ16      = 63570

	peMaxNBSubfr = 4
	peMinLagMS   = 2
	peMaxLagMS   = 18
)

type nlsfCB struct {
	nVectors, order    int
	quantStepSizeQ16   int32
	invQuantStepSizeQ6 int32
	cb1NLSFQ8          []uint8
	cb1WghtQ9          []int16
	cb1ICDF            []uint8
	predQ8             []uint8
	ecSel              []uint8
	ecICDF             []uint8
	ecRatesQ5          []uint8
	deltaMinQ15        []int16
}

var silkNLSFCBNBMB = &nlsfCB{
	nVectors: 32, order: 10,
	quantStepSizeQ16:   silkFixConst(0.18, 16),
	invQuantStepSizeQ6: silkFixConst(1.0/0.18, 6),
	cb1NLSFQ8:          silk_NLSF_CB1_NB_MB_Q8,
	cb1WghtQ9:          silk_NLSF_CB1_Wght_Q9,
	cb1ICDF:            silk_NLSF_CB1_iCDF_NB_MB,
	predQ8:             silk_NLSF_PRED_NB_MB_Q8,
	ecSel:              silk_NLSF_CB2_SELECT_NB_MB,
	ecICDF:             silk_NLSF_CB2_iCDF_NB_MB,
	ecRatesQ5:          silk_NLSF_CB2_BITS_NB_MB_Q5,
	deltaMinQ15:        silk_NLSF_DELTA_MIN_NB_MB_Q15,
}

var silkNLSFCBWB = &nlsfCB{
	nVectors: 32, order: 16,
	quantStepSizeQ16:   silkFixConst(0.15, 16),
	invQuantStepSizeQ6: silkFixConst(1.0/0.15, 6),
	cb1NLSFQ8:          silk_NLSF_CB1_WB_Q8,
	cb1WghtQ9:          silk_NLSF_CB1_WB_Wght_Q9,
	cb1ICDF:            silk_NLSF_CB1_iCDF_WB,
	predQ8:             silk_NLSF_PRED_WB_Q8,
	ecSel:              silk_NLSF_CB2_SELECT_WB,
	ecICDF:             silk_NLSF_CB2_iCDF_WB,
	ecRatesQ5:          silk_NLSF_CB2_BITS_WB_Q5,
	deltaMinQ15:        silk_NLSF_DELTA_MIN_WB_Q15,
}

var silkLTPGainICDFPtrs = [][]uint8{silk_LTP_gain_iCDF_0, silk_LTP_gain_iCDF_1, silk_LTP_gain_iCDF_2}
var silkLTPVQPtrsQ7 = [][][]int8{silk_LTP_gain_vq_0, silk_LTP_gain_vq_1, silk_LTP_gain_vq_2}
var silkLBRRFlagsICDFPtr = [][]uint8{silk_LBRR_flags_2_iCDF, silk_LBRR_flags_3_iCDF}

const (
	resamplerOrderFIR12 = 8
	resamplerMaxBatchMS = 10
	resamplerMaxFsKHz   = 48
)
