package mp3

import "math"

const quantExp = 3.0 / 16.0

const maxQuant = 8191

const part23Max = 4095

const maxOuterIter = 10

const nSfBands = 21

var sfPartCount = [4]int{6, 5, 5, 5}

var sfPartMax = [4]int{15, 15, 7, 7}

var sfPartOf [nSfBands]int

var sfbEdgesLong [8][23]int

func init() {
	for row := 0; row < 8; row++ {
		sum := 0
		for i := 0; i < 22; i++ {
			sfbEdgesLong[row][i] = sum
			sum += int(sfbLong[row][i])
		}
		sfbEdgesLong[row][22] = sum
	}
	p, n := 0, 0
	for b := range sfPartOf {
		if n == sfPartCount[p] {
			p++
			n = 0
		}
		sfPartOf[b] = p
		n++
	}
}

type gcQuant struct {
	globalGain   int
	bigValues    int
	region0Count int
	region1Count int
	region0End   int
	region1End   int
	table        [3]int
	count1Table  int
	count1       int
	part2        int
	part23       int
	scfCompress  int
	preflag      bool
	scfScale     int
	slen         [4]int
	sfTx         [nSfBands]int
	ix           [576]int
}

type quantIn struct {
	xr    *[576]float32
	row   int
	thr   *[nSfBands]float64
	mpeg1 bool
}

func xrPow(xr *[576]float32, out, abs *[576]float64) float64 {
	maxp := 0.0
	for i, v := range xr {
		a := math.Abs(float64(v))
		abs[i] = a
		s := math.Sqrt(a)
		p := s * math.Sqrt(s)
		out[i] = p
		if p > maxp {
			maxp = p
		}
	}
	return maxp
}

func quantizeAt(xrpow *[576]float64, gg int, ix *[576]int) int {
	istep := math.Exp2(-float64(gg-210) * quantExp)
	maxIx := 0
	for i, p := range xrpow {
		v := int(p*istep + 0.4054)
		if v > maxQuant {
			v = maxQuant
		}
		ix[i] = v
		if v > maxIx {
			maxIx = v
		}
	}
	return maxIx
}

func applySigns(xr *[576]float32, ix *[576]int) {
	for i, v := range xr {
		if v < 0 {
			ix[i] = -ix[i]
		}
	}
}

func runLength(ix *[576]int) (bigValues, count1 int) {
	i := 576
	for i > 1 && ix[i-1] == 0 && ix[i-2] == 0 {
		i -= 2
	}
	for i >= 4 && le1(ix[i-1]) && le1(ix[i-2]) && le1(ix[i-3]) && le1(ix[i-4]) {
		i -= 4
		count1++
	}
	return i / 2, count1
}

func le1(v int) bool { return v <= 1 && v >= -1 }

func planHuffman(ix *[576]int, bigValues, count1, row int, q *gcQuant, exact bool) int {
	bigEnd := bigValues * 2
	edges := &sfbEdgesLong[row]

	r0, r1 := 0, 0
	if bigEnd > 0 {
		third := bigEnd / 3
		for r0 < 15 && r0+1 <= 21 && edges[r0+1] <= third {
			r0++
		}
		for r1 < 7 && r0+r1+2 <= 21 && edges[r0+r1+2] <= 2*third {
			r1++
		}
	}
	b0, b1 := regionBounds(r0, r1, bigEnd, row)

	t0, bits0 := selectTable(ix, 0, b0, exact)
	t1, bits1 := selectTable(ix, b0, b1, exact)
	t2, bits2 := selectTable(ix, b1, bigEnd, exact)
	c1Bits, c1Tab := count1Select(ix, bigEnd, count1, exact)

	q.region0Count = r0
	q.region1Count = r1
	q.region0End = b0
	q.region1End = b1
	q.table = [3]int{t0, t1, t2}
	q.count1Table = c1Tab
	return bits0 + bits1 + bits2 + c1Bits
}

func regionBounds(r0, r1, bigEnd, row int) (b0, b1 int) {
	edges := &sfbEdgesLong[row]
	b0 = min(edges[min(r0+1, 22)], bigEnd)
	b1 = min(edges[min(r0+r1+2, 22)], bigEnd)
	if b1 < b0 {
		b1 = b0
	}
	return b0, b1
}

var nlTiers = [...][]int{{1}, {2, 3}, {5, 6}, {7, 8, 9}, {10, 11, 12}, {13, 15}}
var nlTierMax = [...]int{1, 2, 3, 5, 7, 15}

func selectTable(ix *[576]int, lo, hi int, exact bool) (table, bits int) {
	if lo >= hi {
		return 0, 0
	}
	maxv := 0
	for i := lo; i < hi; i++ {
		if a := abs(ix[i]); a > maxv {
			maxv = a
		}
	}
	if maxv == 0 {
		return 0, 0
	}

	if maxv > 15 {
		ta := firstCovering(treeA, maxv)
		tb := firstCovering(treeB, maxv)
		if !exact && ta >= 0 {
			tb = -1
		}
		var bitsA, bitsB int
		for i := lo; i+1 < hi; i += 2 {
			ax, ay := abs(ix[i]), abs(ix[i+1])
			xi, yi, esc, signs := ax, ay, 0, 0
			if ax >= 15 {
				xi = 15
				esc++
			}
			if ay >= 15 {
				yi = 15
				esc++
			}
			if ax != 0 {
				signs++
			}
			if ay != 0 {
				signs++
			}
			idx := xi<<4 | yi
			if ta >= 0 {
				bitsA += int(bigEnc[ta][idx].n) + esc*int(huffTables[ta].linbits) + signs
			}
			if tb >= 0 {
				bitsB += int(bigEnc[tb][idx].n) + esc*int(huffTables[tb].linbits) + signs
			}
		}
		switch {
		case ta < 0:
			return tb, bitsB
		case tb < 0 || bitsA <= bitsB:
			return ta, bitsA
		default:
			return tb, bitsB
		}
	}

	tier := 0
	for nlTierMax[tier] < maxv {
		tier++
	}
	var cands [6]int
	n := copy(cands[:], nlTiers[tier])
	if exact && tier+1 < len(nlTiers) {
		n += copy(cands[n:], nlTiers[tier+1])
	}
	if !exact {
		n = 1
	}
	var sums [6]int
	signBits := 0
	for i := lo; i+1 < hi; i += 2 {
		ax, ay := abs(ix[i]), abs(ix[i+1])
		idx := ax<<4 | ay
		if ax != 0 {
			signBits++
		}
		if ay != 0 {
			signBits++
		}
		for j := 0; j < n; j++ {
			sums[j] += int(bigEnc[cands[j]][idx].n)
		}
	}
	bestT, bestB := cands[0], sums[0]
	for j := 1; j < n; j++ {
		if sums[j] < bestB {
			bestT, bestB = cands[j], sums[j]
		}
	}
	return bestT, bestB + signBits
}

func firstCovering(tree []int, maxv int) int {
	for _, t := range tree {
		if tableLimit(t) >= maxv {
			return t
		}
	}
	return -1
}

func count1Select(ix *[576]int, bigEnd, count1 int, exact bool) (bits, table int) {
	var b0, b1 int
	for i := bigEnd; i+3 < bigEnd+count1*4 && i+3 < 576; i += 4 {
		b0 += quadBits(0, ix[i], ix[i+1], ix[i+2], ix[i+3])
		if exact {
			b1 += quadBits(1, ix[i], ix[i+1], ix[i+2], ix[i+3])
		}
	}
	if exact && b1 < b0 {
		return b1, 1
	}
	return b0, 0
}

func bitsFor(v int) int {
	n := 0
	for 1<<n-1 < v {
		n++
	}
	return n
}

func resolveScalefactors(sf *[nSfBands]int, mpeg1 bool) (part2 int, slen [4]int, compress int, preflag bool, tx [nSfBands]int) {
	if !mpeg1 {
		tx = *sf
		for b, v := range tx {
			if w := bitsFor(v); w > slen[sfPartOf[b]] {
				slen[sfPartOf[b]] = w
			}
		}
		compress = ((slen[0]*5+slen[1])*4+slen[2])*4 + slen[3]
		part2 = 6*slen[0] + 5*slen[1] + 5*slen[2] + 5*slen[3]
		return part2, slen, compress, false, tx
	}

	best := -1
	for _, pre := range []bool{false, true} {
		cand := *sf
		if pre {
			ok := true
			for i, p := range preamp {
				if cand[11+i] < int(p) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			for i, p := range preamp {
				cand[11+i] -= int(p)
			}
		}
		n1, n2 := 0, 0
		for b, v := range cand {
			w := bitsFor(v)
			if b <= 10 {
				n1 = max(n1, w)
			} else {
				n2 = max(n2, w)
			}
		}
		for c, packed := range scfcDecode {
			s1, s2 := int(packed>>2), int(packed&3)
			if s1 < n1 || s2 < n2 {
				continue
			}
			cost := 11*s1 + 10*s2
			if best < 0 || cost < part2 {
				best = c
				part2 = cost
				slen = [4]int{s1, s1, s2, s2}
				compress = c
				preflag = pre
				tx = cand
			}
		}
	}
	return part2, slen, compress, preflag, tx
}

func quantizeGranule(in quantIn, budget int) gcQuant {
	if budget > part23Max {
		budget = part23Max
	}
	var xrpow, absXr [576]float64
	basep := xrPow(in.xr, &xrpow, &absXr)

	var q gcQuant
	if basep == 0 {
		q.globalGain = 210
		return q
	}

	edges := &sfbEdgesLong[in.row]
	var sf [nSfBands]int
	scfScale := 0
	maxp := basep

	ggFloor := func(m float64) int {
		g := 210 + int(math.Ceil(math.Log2(m/float64(maxQuant))/quantExp))
		return min(max(g, 0), 255)
	}

	planAt := func(gg int, r *gcQuant) {
		*r = gcQuant{globalGain: gg}
		quantizeAt(&xrpow, gg, &r.ix)
		bv, c1 := runLength(&r.ix)
		r.bigValues = bv
		r.count1 = c1
		r.part23 = planHuffman(&r.ix, bv, c1, in.row, r, false)
	}

	amplifyStep := func() float64 {
		return math.Exp2(float64(int(1)<<uint(scfScale+1)) * quantExp)
	}

	var probe, best gcQuant
	bestScore := math.Inf(1)
	var bestSf [nSfBands]int
	bestScale := 0
	haveBest := false
	havePrev := false
	prevGG := 0
	stalled := 0

	for iter := 0; ; iter++ {
		part2, _, _, _, _ := resolveScalefactors(&sf, in.mpeg1)
		huffBudget := max(budget-part2, 0)

		ggMin := ggFloor(maxp)
		fitted := false
		if !havePrev {
			planAt(255, &probe)
			lo, hi := ggMin, 255
			fit := probe
			for lo <= hi {
				mid := (lo + hi) / 2
				planAt(mid, &probe)
				if probe.part23 <= huffBudget {
					fit = probe
					hi = mid - 1
				} else {
					lo = mid + 1
				}
			}
			probe = fit
			fitted = probe.part23 <= huffBudget
			havePrev = true
		} else {
			gg := max(prevGG, ggMin)
			for ; gg <= 255; gg++ {
				planAt(gg, &probe)
				if probe.part23 <= huffBudget {
					fitted = true
					break
				}
			}
			if !fitted {
				planAt(255, &probe)
			}
		}
		prevGG = probe.globalGain
		if !fitted {
			break
		}

		score := 0.0
		violated := false
		var wants [nSfBands]bool
		if in.thr != nil {
			for b := 0; b < nSfBands; b++ {
				sb := sf[b] << uint(scfScale+1)
				qexp := max(probe.globalGain-210-sb, -pow2qBias)
				mult := pow2q[qexp+pow2qBias]
				noise := 0.0
				for i := edges[b]; i < edges[b+1]; i++ {
					d := absXr[i] - pow43[probe.ix[i]]*mult
					noise += d * d
				}
				if thr := in.thr[b]; thr > 0 && noise > thr {
					score += math.Log2(noise / thr)
					if sf[b] < sfPartMax[sfPartOf[b]] {
						wants[b] = true
						violated = true
					} else if scfScale == 0 {
						wants[b] = true
						violated = true
					}
				}
			}
		}
		if score < bestScore {
			bestScore = score
			best = probe
			best.part2 = part2
			bestSf = sf
			bestScale = scfScale
			haveBest = true
			stalled = 0
		} else {
			stalled++
		}
		if !violated || iter >= maxOuterIter || stalled >= 2 || in.thr == nil {
			break
		}

		escalate := false
		for b := 0; b < nSfBands; b++ {
			if wants[b] && sf[b] >= sfPartMax[sfPartOf[b]] {
				escalate = true
			}
		}
		if escalate && scfScale == 0 {
			scfScale = 1
			for b := range sf {
				sf[b] = (sf[b] + 1) / 2
			}
			step := amplifyStep()
			maxp = 0
			for b := 0; b < nSfBands; b++ {
				amp := math.Pow(step, float64(sf[b]))
				for i := edges[b]; i < edges[b+1]; i++ {
					p := math.Sqrt(absXr[i])
					p *= math.Sqrt(p)
					p *= amp
					xrpow[i] = p
					if p > maxp {
						maxp = p
					}
				}
			}
			for i := edges[nSfBands]; i < 576; i++ {
				if xrpow[i] > maxp {
					maxp = xrpow[i]
				}
			}
		}
		step := amplifyStep()
		for b := 0; b < nSfBands; b++ {
			if !wants[b] || sf[b] >= sfPartMax[sfPartOf[b]] {
				continue
			}
			sf[b]++
			for i := edges[b]; i < edges[b+1]; i++ {
				xrpow[i] *= step
				if xrpow[i] > maxp {
					maxp = xrpow[i]
				}
			}
		}
	}

	if !haveBest {
		return gcQuant{globalGain: 210}
	}

	best.part23 = planHuffman(&best.ix, best.bigValues, best.count1, in.row, &best, true)
	part2, slen, compress, preflag, tx := resolveScalefactors(&bestSf, in.mpeg1)
	best.part2 = part2
	best.part23 += part2
	best.slen = slen
	best.scfCompress = compress
	best.preflag = preflag
	best.sfTx = tx
	best.scfScale = bestScale
	applySigns(in.xr, &best.ix)
	return best
}
