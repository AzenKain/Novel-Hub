package opus

import "math"

// CELT PVQ (pyramid vector quantization) shape decoding, RFC 6716 section 4.3.4.

func unext(u []uint32, ln int, ui0 uint32) {
	j := 1
	for ; j < ln; j++ {
		ui1 := u[j] + u[j-1] + ui0
		u[j-1] = ui0
		ui0 = ui1
	}
	u[j-1] = ui0
}

func uprev(u []uint32, ui0 uint32) {
	ln := len(u)
	j := 1
	for ; j < ln; j++ {
		ui1 := u[j] - u[j-1] - ui0
		u[j-1] = ui0
		ui0 = ui1
	}
	u[j-1] = ui0
}

func ncwrsURow(n, k int, u []uint32) uint32 {
	ln := k + 2
	u[0] = 0
	u[1] = 1
	for i := 2; i < ln; i++ {
		u[i] = uint32(i<<1) - 1
	}
	for i := 2; i < n; i++ {
		unext(u[1:], k+1, 1)
	}
	return u[k] + u[k+1]
}

func cwrsi(n, k int, i uint32, y []int, u []uint32) float32 {
	var yy float32
	for j := 0; j < n; j++ {
		p := u[k+1]
		s := 0
		if i >= p {
			s = -1
		}
		i -= p & uint32(s)
		yj := k
		p = u[k]
		for p > i {
			k--
			p = u[k]
		}
		i -= p
		yj -= k
		val := (yj + s) ^ s
		y[j] = val
		yy += float32(val) * float32(val)
		uprev(u[:k+2], 0)
	}
	return yy
}

func decodePulses(y []int, n, k int, d *rangeDecoder, u []uint32) float32 {
	v := ncwrsURow(n, k, u)
	i := d.decodeUint(v)
	return cwrsi(n, k, i, y, u)
}

func expRotation1(X []float32, length, stride int, c, s float32) {
	ms := -s
	for i := 0; i < length-stride; i++ {
		x1 := X[i]
		x2 := X[i+stride]
		X[i+stride] = c*x2 + s*x1
		X[i] = c*x1 + ms*x2
	}
	for i := length - 2*stride - 1; i >= 0; i-- {
		x1 := X[i]
		x2 := X[i+stride]
		X[i+stride] = c*x2 + s*x1
		X[i] = c*x1 + ms*x2
	}
}

var spreadFactor = [3]int{15, 10, 5}

func expRotation(X []float32, n, dir, B, K, spread int) {
	if 2*K >= n || spread == spreadNone {
		return
	}
	factor := spreadFactor[spread-1]
	gain := float32(n) / float32(n+factor*K)
	theta := 0.5 * gain * gain
	c := float32(math.Cos(0.5 * math.Pi * float64(theta)))
	s := float32(math.Cos(0.5 * math.Pi * float64(1-theta)))

	stride2 := 0
	length := n
	if length >= 8*B {
		stride2 = 1
		for (stride2*stride2+stride2)*B+(B>>2) < length {
			stride2++
		}
	}
	length /= B
	for i := 0; i < B; i++ {
		seg := X[i*length:]
		if dir < 0 {
			if stride2 != 0 {
				expRotation1(seg, length, stride2, s, c)
			}
			expRotation1(seg, length, 1, c, s)
		} else {
			expRotation1(seg, length, 1, c, -s)
			if stride2 != 0 {
				expRotation1(seg, length, stride2, s, -c)
			}
		}
	}
}

func normaliseResidual(iy []int, X []float32, n int, ryy, gain float32) {
	g := gain / float32(math.Sqrt(float64(ryy)))
	for i := 0; i < n; i++ {
		X[i] = float32(iy[i]) * g
	}
}

func extractCollapseMask(iy []int, N, B int) uint32 {
	if B <= 1 {
		return 1
	}
	N0 := N / B
	var mask uint32
	for i := 0; i < B; i++ {
		var tmp int
		for j := 0; j < N0; j++ {
			tmp |= iy[i*N0+j]
		}
		if tmp != 0 {
			mask |= 1 << uint(i)
		}
	}
	return mask
}

func algUnquant(X []float32, n, K, spread, B int, d *rangeDecoder, gain float32, iy []int, u []uint32) uint32 {
	ryy := decodePulses(iy, n, K, d, u)
	normaliseResidual(iy, X, n, ryy, gain)
	expRotation(X, n, -1, B, K, spread)
	return extractCollapseMask(iy, n, B)
}

func iabs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func opPVQSearch(X []float32, iy []int, K, N int) float32 {
	y := make([]float32, N)
	signx := make([]int, N)
	var sum, xy, yy float32
	for j := 0; j < N; j++ {
		if X[j] < 0 {
			signx[j] = 1
		} else {
			signx[j] = 0
		}
		X[j] = absf(X[j])
		iy[j] = 0
		y[j] = 0
	}
	pulsesLeft := K
	if K > N>>1 {
		for j := 0; j < N; j++ {
			sum += X[j]
		}
		if !(sum > 1e-15 && sum < 64) {
			X[0] = 1.0
			for j := 1; j < N; j++ {
				X[j] = 0
			}
			sum = 1.0
		}
		rcp := (float32(K) + 0.8) / sum
		for j := 0; j < N; j++ {
			iy[j] = int(math.Floor(float64(rcp * X[j])))
			y[j] = float32(iy[j])
			yy += y[j] * y[j]
			xy += X[j] * y[j]
			y[j] *= 2
			pulsesLeft -= iy[j]
		}
	}
	if pulsesLeft > N+3 {
		tmp := float32(pulsesLeft)
		yy += tmp * tmp
		yy += tmp * y[0]
		iy[0] += pulsesLeft
		pulsesLeft = 0
	}
	for i := 0; i < pulsesLeft; i++ {
		yy += 1
		Rxy := xy + X[0]
		Ryy := yy + y[0]
		Rxy = Rxy * Rxy
		bestDen := Ryy
		bestNum := Rxy
		bestID := 0
		for j := 1; j < N; j++ {
			Rxy = xy + X[j]
			Ryy = yy + y[j]
			Rxy = Rxy * Rxy
			if bestDen*Rxy > Ryy*bestNum {
				bestDen = Ryy
				bestNum = Rxy
				bestID = j
			}
		}
		xy += X[bestID]
		yy += y[bestID]
		y[bestID] += 2
		iy[bestID]++
	}
	for j := 0; j < N; j++ {
		iy[j] = (iy[j] ^ -signx[j]) + signx[j]
	}
	return yy
}

func icwrs(n, k int, y []int, u []uint32) (i, nc uint32) {
	u[0] = 0
	for m := 1; m <= k+1; m++ {
		u[m] = uint32(m<<1) - 1
	}
	run := iabs(y[n-1])
	if y[n-1] < 0 {
		i = 1
	}
	j := n - 2
	i += u[run]
	run += iabs(y[j])
	if y[j] < 0 {
		i += u[run+1]
	}
	for j > 0 {
		j--
		unext(u, k+2, 0)
		i += u[run]
		run += iabs(y[j])
		if y[j] < 0 {
			i += u[run+1]
		}
	}
	return i, u[run] + u[run+1]
}

func encodePulses(y []int, n, k int, e *rangeEncoder, u []uint32) {
	i, nc := icwrs(n, k, y, u)
	e.encodeUint(i, nc)
}

func algQuant(X []float32, n, K, spread, B int, e *rangeEncoder, gain float32, iy []int, u []uint32, resynth bool) uint32 {
	expRotation(X, n, 1, B, K, spread)
	yy := opPVQSearch(X, iy, K, n)
	cm := extractCollapseMask(iy, n, B)
	encodePulses(iy, n, K, e, u)
	if resynth {
		normaliseResidual(iy, X, n, yy, gain)
		expRotation(X, n, -1, B, K, spread)
	}
	return cm
}

func stereoItheta(X, Y []float32, stereo, N int) int {
	var Emid, Eside float32
	if stereo != 0 {
		for i := 0; i < N; i++ {
			m := X[i] + Y[i]
			s := X[i] - Y[i]
			Emid += m * m
			Eside += s * s
		}
	} else {
		for i := 0; i < N; i++ {
			Emid += X[i] * X[i]
			Eside += Y[i] * Y[i]
		}
	}
	if Emid+Eside < 1e-18 {
		return 0
	}
	mid := math.Sqrt(float64(Emid))
	side := math.Sqrt(float64(Eside))
	return int(math.Floor(0.5 + 1073741824.0*(math.Atan2(side, mid)*(2.0/math.Pi))))
}

func renormaliseVector(X []float32, N int, gain float32) {
	E := float32(1e-15)
	for i := 0; i < N; i++ {
		E += X[i] * X[i]
	}
	g := gain / float32(math.Sqrt(float64(E)))
	for i := 0; i < N; i++ {
		X[i] *= g
	}
}
