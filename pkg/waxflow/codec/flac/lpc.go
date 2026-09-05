package flac

import "math"

const maxLPCOrder = 12

const lpcPrecision = 15

const maxLPCShift = 15

func tukey(w []float64, p float64) {
	n := len(w)
	taper := int(p * float64(n-1) / 2)
	for i := 0; i <= taper && i < n; i++ {
		v := 0.5 * (1 + math.Cos(math.Pi*(float64(i)/float64(taper+1)-1)))
		w[i] = v
		w[n-1-i] = v
	}
	for i := taper + 1; i < n-1-taper; i++ {
		w[i] = 1
	}
}

func autocorr(x []float64, r []float64) {
	for lag := range r {
		sum := 0.0
		for i := lag; i < len(x); i++ {
			sum += x[i] * x[i-lag]
		}
		r[lag] = sum
	}
}

func levinson(r []float64, coeffs [][]float64, errs []float64) int {
	maxOrder := len(r) - 1
	e := r[0]
	errs[0] = e
	if e <= 0 {
		return 0
	}
	var a [maxLPCOrder]float64
	for m := 1; m <= maxOrder; m++ {
		acc := r[m]
		for j := 0; j < m-1; j++ {
			acc -= a[j] * r[m-1-j]
		}
		k := acc / e
		a[m-1] = k
		for j := 0; j < (m-1)/2; j++ {
			a[j], a[m-2-j] = a[j]-k*a[m-2-j], a[m-2-j]-k*a[j]
		}
		if (m-1)%2 == 1 {
			mid := (m - 1) / 2
			a[mid] -= k * a[mid]
		}
		e *= 1 - k*k
		if e <= 0 || math.IsNaN(e) {
			return m - 1
		}
		errs[m] = e
		copy(coeffs[m-1], a[:m])
	}
	return maxOrder
}

func estimateOrder(errs []float64, usable, n int, bps uint) int {
	best, bestBits := 1, math.Inf(1)
	for m := 1; m <= usable; m++ {
		perSample := 0.5 * math.Log2(errs[m]/float64(n))
		if perSample < 0 {
			perSample = 0
		}
		bits := float64(n-m)*perSample + float64(m)*float64(bps+lpcPrecision)
		if bits < bestBits {
			best, bestBits = m, bits
		}
	}
	return best
}

func quantizeLPC(c []float64, q []int64) (shift uint) {
	cmax := 0.0
	for _, v := range c {
		if a := math.Abs(v); a > cmax {
			cmax = a
		}
	}
	if cmax <= 0 {
		return 0
	}
	_, exp := math.Frexp(cmax)
	s := lpcPrecision - 1 - exp
	if s > maxLPCShift {
		s = maxLPCShift
	}
	if s < 0 {
		s = 0
	}
	const lim = int64(1)<<(lpcPrecision-1) - 1
	e := 0.0
	scale := float64(int64(1) << uint(s))
	for i, v := range c {
		t := v*scale + e
		qi := int64(math.Round(t))
		if qi > lim {
			qi = lim
		} else if qi < -lim-1 {
			qi = -lim - 1
		}
		e = t - float64(qi)
		q[i] = qi
	}
	return uint(s)
}

func lpcResidual(x []int64, q []int64, shift uint, res []int64) {
	order := len(q)
	for i := order; i < len(x); i++ {
		var sum int64
		for j, cf := range q {
			sum += cf * x[i-1-j]
		}
		res[i] = x[i] - sum>>shift
	}
}

func fixedResidual(x []int64, order int, res []int64) {
	switch order {
	case 0:
		copy(res, x)
	case 1:
		for i := 1; i < len(x); i++ {
			res[i] = x[i] - x[i-1]
		}
	case 2:
		for i := 2; i < len(x); i++ {
			res[i] = x[i] - 2*x[i-1] + x[i-2]
		}
	case 3:
		for i := 3; i < len(x); i++ {
			res[i] = x[i] - 3*x[i-1] + 3*x[i-2] - x[i-3]
		}
	case 4:
		for i := 4; i < len(x); i++ {
			res[i] = x[i] - 4*x[i-1] + 6*x[i-2] - 4*x[i-3] + x[i-4]
		}
	}
}

func bestFixedOrder(x []int64, scratch []int64) int {
	n := len(x)
	maxOrder := 4
	if n <= maxOrder {
		maxOrder = n - 1
	}
	d := scratch[:n]
	copy(d, x)
	var best int
	bestSum := magSum(d)
	for order := 1; order <= maxOrder; order++ {
		for i := n - 1; i >= order; i-- {
			d[i] -= d[i-1]
		}
		if sum := magSum(d[order:]); sum < bestSum {
			best, bestSum = order, sum
		}
	}
	return best
}

func magSum(v []int64) uint64 {
	var sum uint64
	for _, s := range v {
		if s < 0 {
			sum += uint64(-s)
		} else {
			sum += uint64(s)
		}
	}
	return sum
}
