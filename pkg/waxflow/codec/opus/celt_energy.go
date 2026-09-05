package opus

import "math"

// CELT band-energy dequantization (RFC 6716 section 4.3.2).

const maxFineBits = 8

func unquantCoarseEnergy(oldE []float32, nbEBands, start, end int, intra bool, d *rangeDecoder, C, LM int) {
	var pm [42]byte
	var coef, beta float32
	if intra {
		pm = celtEProbModel[LM][1]
		coef, beta = 0, celtBetaIntra
	} else {
		pm = celtEProbModel[LM][0]
		coef, beta = celtPredCoef[LM], celtBetaCoef[LM]
	}
	budget := d.storage * 8
	var prev [2]float32
	for i := start; i < end; i++ {
		for c := 0; c < C; c++ {
			var qi int
			tell := d.tell()
			switch {
			case budget-tell >= 15:
				pi := 2 * min(i, 20)
				qi = d.laplaceDecode(uint32(pm[pi])<<7, int(pm[pi+1])<<6)
			case budget-tell >= 2:
				q := d.decodeICDF(celtSmallEnergyICDF, 2)
				qi = (q >> 1) ^ -(q & 1)
			case budget-tell >= 1:
				qi = -d.decodeBitLogp(1)
			default:
				qi = -1
			}
			q := float32(qi)
			idx := i + c*nbEBands
			if oldE[idx] < -9 {
				oldE[idx] = -9
			}
			tmp := coef*oldE[idx] + prev[c] + q
			oldE[idx] = tmp
			prev[c] = prev[c] + q - beta*q
		}
	}
}

func unquantFineEnergy(oldE []float32, nbEBands, start, end int, fineQuant []int, d *rangeDecoder, C int) {
	for i := start; i < end; i++ {
		extra := fineQuant[i]
		if extra <= 0 {
			continue
		}
		if d.tell()+C*extra > d.storage*8 {
			continue
		}
		for c := 0; c < C; c++ {
			q2 := int(d.decodeRawBits(uint(extra)))
			offset := (float32(q2)+0.5)*float32(int(1)<<(14-extra))*(1.0/16384) - 0.5
			oldE[i+c*nbEBands] += offset
		}
	}
}

func unquantEnergyFinalise(oldE []float32, nbEBands, start, end int, fineQuant, finePriority []int, bitsLeft int, d *rangeDecoder, C int) {
	for prio := 0; prio < 2; prio++ {
		for i := start; i < end && bitsLeft >= C; i++ {
			if fineQuant[i] >= maxFineBits || finePriority[i] != prio {
				continue
			}
			for c := 0; c < C; c++ {
				q2 := int(d.decodeRawBits(1))
				offset := (float32(q2) - 0.5) * float32(int(1)<<(14-fineQuant[i]-1)) * (1.0 / 16384)
				oldE[i+c*nbEBands] += offset
				bitsLeft--
			}
		}
	}
}

func computeBandEnergies(X, bandE []float32, end, C, LM int) {
	N := celtShortMDCTSize << LM
	for c := 0; c < C; c++ {
		for i := 0; i < end; i++ {
			lo := int(celtEBands[i]) << LM
			hi := int(celtEBands[i+1]) << LM
			var sum float32 = 1e-27
			for j := lo; j < hi; j++ {
				v := X[c*N+j]
				sum += v * v
			}
			bandE[i+c*celtNBands] = float32(math.Sqrt(float64(sum)))
		}
	}
}

func normaliseBands(freq, X, bandE []float32, end, C, M int) {
	N := M * celtShortMDCTSize
	for c := 0; c < C; c++ {
		for i := 0; i < end; i++ {
			g := 1.0 / (1e-27 + bandE[i+c*celtNBands])
			for j := M * int(celtEBands[i]); j < M*int(celtEBands[i+1]); j++ {
				X[j+c*N] = freq[j+c*N] * g
			}
		}
	}
}

func amp2Log2(effEnd, end int, bandE, bandLogE []float32, C int) {
	for c := 0; c < C; c++ {
		for i := 0; i < effEnd; i++ {
			bandLogE[i+c*celtNBands] = float32(math.Log2(float64(bandE[i+c*celtNBands]))) - celtEMeans[i]
		}
		for i := effEnd; i < end; i++ {
			bandLogE[c*celtNBands+i] = -14
		}
	}
}

func lossDistortion(eBands, oldEBands []float32, start, end, C int) float32 {
	var dist float32
	for c := 0; c < C; c++ {
		for i := start; i < end; i++ {
			d := eBands[i+c*celtNBands] - oldEBands[i+c*celtNBands]
			dist += d * d
		}
	}
	return min(200, dist)
}

func quantCoarseEnergyImpl(start, end int, eBands, oldEBands []float32, budget, tell int,
	probModel []byte, errorArr []float32, e *rangeEncoder, C, LM, intra int, maxDecay float32) int {

	badness := 0
	var prev [2]float32
	var coef, beta float32
	if tell+3 <= budget {
		e.encodeBitLogp(intra, 3)
	}
	if intra != 0 {
		coef, beta = 0, celtBetaIntra
	} else {
		coef, beta = celtPredCoef[LM], celtBetaCoef[LM]
	}
	for i := start; i < end; i++ {
		for c := 0; c < C; c++ {
			x := eBands[i+c*celtNBands]
			oldE := max(-9.0, oldEBands[i+c*celtNBands])
			f := x - coef*oldE - prev[c]
			qi := int(math.Floor(float64(0.5 + f)))
			decayBound := max(-28.0, oldEBands[i+c*celtNBands]) - maxDecay
			if qi < 0 && x < decayBound {
				qi += int(decayBound - x)
				if qi > 0 {
					qi = 0
				}
			}
			qi0 := qi
			tell = e.tell()
			bitsLeft := budget - tell - 3*C*(end-i)
			if i != start && bitsLeft < 30 {
				if bitsLeft < 24 {
					qi = min(1, qi)
				}
				if bitsLeft < 16 {
					qi = max(-1, qi)
				}
			}
			switch {
			case budget-tell >= 15:
				pi := 2 * min(i, 20)
				qi = e.laplaceEncode(qi, uint32(probModel[pi])<<7, int(probModel[pi+1])<<6)
			case budget-tell >= 2:
				qi = max(-1, min(qi, 1))
				e.encodeICDF(2*qi^-b2i(qi < 0), celtSmallEnergyICDF, 2)
			case budget-tell >= 1:
				qi = min(0, qi)
				e.encodeBitLogp(-qi, 1)
			default:
				qi = -1
			}
			errorArr[i+c*celtNBands] = f - float32(qi)
			badness += iabs(qi0 - qi)
			q := float32(qi)
			tmp := coef*oldE + prev[c] + q
			oldEBands[i+c*celtNBands] = tmp
			prev[c] = prev[c] + q - beta*q
		}
	}
	return badness
}

func quantCoarseEnergy(start, end, effEnd int, eBands, oldEBands []float32, budget int,
	errorArr []float32, e *rangeEncoder, C, LM, nbAvailableBytes int, forceIntra bool, delayedIntra *float32, twoPass bool) {

	nb := celtNBands
	intra := 0
	if forceIntra || (!twoPass && *delayedIntra > float32(2*C*(end-start)) && nbAvailableBytes > (end-start)*C) {
		intra = 1
	}
	const intraBias = 0
	newDistortion := lossDistortion(eBands, oldEBands, start, effEnd, C)
	tell := e.tell()
	if tell+3 > budget {
		twoPass = false
		intra = 0
	}
	maxDecay := float32(16.0)
	if end-start > 10 {
		maxDecay = min(maxDecay, 0.125*float32(nbAvailableBytes))
	}
	encStart := e.snapshot()

	oldEIntra := append([]float32(nil), oldEBands...)
	errorIntra := make([]float32, C*nb)
	badness1 := 0
	if twoPass || intra != 0 {
		badness1 = quantCoarseEnergyImpl(start, end, eBands, oldEIntra, budget, tell,
			celtEProbModel[LM][1][:], errorIntra, e, C, LM, 1, maxDecay)
	}
	if intra == 0 {
		tellIntra := e.tellFrac()
		encIntra := e.snapshot()
		intraBits := e.tailBytes(&encStart, nil)
		e.restore(&encStart)
		badness2 := quantCoarseEnergyImpl(start, end, eBands, oldEBands, budget, tell,
			celtEProbModel[LM][0][:], errorArr, e, C, LM, 0, maxDecay)
		if twoPass && (badness1 < badness2 || (badness1 == badness2 && e.tellFrac()+intraBias > tellIntra)) {
			e.restore(&encIntra)
			e.restoreTail(&encStart, intraBits)
			copy(oldEBands, oldEIntra)
			copy(errorArr, errorIntra)
			intra = 1
		}
	} else {
		copy(oldEBands, oldEIntra)
		copy(errorArr, errorIntra)
	}
	if intra != 0 {
		*delayedIntra = newDistortion
	} else {
		pc := celtPredCoef[LM]
		*delayedIntra = pc*pc*(*delayedIntra) + newDistortion
	}
}

func quantFineEnergy(start, end int, oldEBands, errorArr []float32, extraQuant []int, e *rangeEncoder, C int) {
	for i := start; i < end; i++ {
		if extraQuant[i] <= 0 {
			continue
		}
		extra := 1 << extraQuant[i]
		if e.tell()+C*extraQuant[i] > e.storage*8 {
			continue
		}
		for c := 0; c < C; c++ {
			q2 := int(math.Floor(float64((errorArr[i+c*celtNBands] + 0.5) * float32(extra))))
			if q2 > extra-1 {
				q2 = extra - 1
			}
			if q2 < 0 {
				q2 = 0
			}
			e.encodeRawBits(uint32(q2), uint(extraQuant[i]))
			offset := (float32(q2)+0.5)*float32(int(1)<<(14-extraQuant[i]))*(1.0/16384) - 0.5
			oldEBands[i+c*celtNBands] += offset
			errorArr[i+c*celtNBands] -= offset
		}
	}
}

func quantEnergyFinalise(start, end int, oldEBands, errorArr []float32, fineQuant, finePriority []int, bitsLeft int, e *rangeEncoder, C int) {
	for prio := 0; prio < 2; prio++ {
		for i := start; i < end && bitsLeft >= C; i++ {
			if fineQuant[i] >= maxFineBits || finePriority[i] != prio {
				continue
			}
			for c := 0; c < C; c++ {
				q2 := 0
				if errorArr[i+c*celtNBands] >= 0 {
					q2 = 1
				}
				e.encodeRawBits(uint32(q2), 1)
				offset := (float32(q2) - 0.5) * float32(int(1)<<(14-fineQuant[i]-1)) * (1.0 / 16384)
				oldEBands[i+c*celtNBands] += offset
				errorArr[i+c*celtNBands] -= offset
				bitsLeft--
			}
		}
	}
}
