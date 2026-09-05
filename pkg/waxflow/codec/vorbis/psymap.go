package vorbis

import (
	"math"

	"novelhub/pkg/waxflow/dsp/psy"
)

func newPsyModel(rate, n2 int, offsetDB float64) (*psy.Model, error) {
	offsets := make([]int, n2+1)
	for i := range offsets {
		offsets[i] = i
	}
	return psy.New(psy.Config{
		Rate:        rate,
		Lines:       n2,
		FFTSize:     2 * n2,
		BandOffsets: offsets,
		OffsetDB:    offsetDB,
		ATHOffsetDB: offsetDB,
	})
}

func lineThresholds(res psy.Result, spec []float32, dst []float64, n2 int) {
	var sumMDCT, sumPsy float64
	for l := 0; l < n2; l++ {
		v := float64(spec[l])
		sumMDCT += v * v
		if l < len(res.Energy) {
			sumPsy += res.Energy[l]
		}
	}
	ratio := 0.0
	if sumPsy > 0 {
		ratio = sumMDCT / sumPsy
	}
	for l := 0; l < n2; l++ {
		if l < len(res.Thr) {
			dst[l] = res.Thr[l] * ratio
		} else {
			dst[l] = 0
		}
	}
}

func classifyPartitions(spec, curve []float32, thrLine []float64, classes []int, partSize, n2 int, capNoise bool) {
	nParts := n2 / partSize
	for p := 0; p < nParts; p++ {
		lo, hi := p*partSize, (p+1)*partSize
		var peak, demand, sumE float64
		for l := lo; l < hi; l++ {
			e := float64(spec[l]) * float64(spec[l])
			sumE += e
			if t := thrLine[l]; t > 0 {
				if snr := e / t; snr > peak {
					peak = snr
				}
				if e > t {
					cv := float64(curve[l])
					d := cv * cv
					if maxD := e * demandSlack; maxD < d {
						d = maxD
					}
					d /= t
					if d > demand {
						demand = d
					}
				}
			}
		}
		if peak <= 1 {
			classes[p] = classSkip
			continue
		}
		if capNoise && partitionFlatness(spec, lo, hi, sumE) >= noiseSFM {
			classes[p] = classNoise
			continue
		}
		classes[p] = classFromDemand(demand)
	}
}

func partitionFlatness(spec []float32, lo, hi int, sumE float64) float64 {
	n := float64(hi - lo)
	meanE := sumE / n
	if meanE <= 0 {
		return 0
	}
	floorE := meanE * 1e-6
	sumLog := 0.0
	for l := lo; l < hi; l++ {
		e := float64(spec[l]) * float64(spec[l])
		if e < floorE {
			e = floorE
		}
		sumLog += math.Log(e)
	}
	return math.Exp(sumLog/n) / meanE
}

func maskResidue(spec, resid []float32, thrLine []float64, n2 int) {
	var peak2 float64
	for l := 0; l < n2; l++ {
		if s := float64(spec[l]); s*s > peak2 {
			peak2 = s * s
		}
	}
	globalFloor := peak2 * globalMaskRatio
	for l := 0; l < n2; l++ {
		s := float64(spec[l])
		e := s * s
		if e > globalFloor {
			continue
		}
		if thrLine == nil || e <= thrLine[l] {
			resid[l] = 0
		}
	}
}

const (
	globalMaskDB    = 50.0
	globalMaskRatio = 1e-5
)

func classFromDemand(demand float64) int {
	db := 10 * math.Log10(demand)
	switch {
	case db <= noiseMaxDB:
		return classNoise
	case db <= coarseMaxDB:
		return classCoarse
	case db <= medMaxDB:
		return classMed
	case db <= fineMaxDB:
		return classFine
	default:
		return classSuper
	}
}

func qualityToOffsetDB(quality float64) float64 {
	return (quality - DefaultQuality) * 2.0
}

const (
	classSkip   = 0
	classNoise  = 1
	classCoarse = 2
	classMed    = 3
	classFine   = 4
	classSuper  = 5
	numResClass = 6
	noiseMaxDB  = 26.0
	coarseMaxDB = 32.0
	medMaxDB    = 44.0
	fineMaxDB   = 56.0
	demandSlack = 16.0
	noiseSFM    = 0.25
)
