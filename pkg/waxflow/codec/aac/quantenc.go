package aac

import "math"

const (
	ampMax      = 30
	maxAmpIter  = 10
	sfSearchMax = 300
)

type encBand struct {
	off, n int
	maxAbs float64
	energy float64
	thr    float64
	minSf  int
	amp    int
	sf     int
	cb     int
}

type bandMemo struct {
	epoch uint32
	bits  int32
	cb    int8
	zero  bool
	noise float64
}

type chanQuant struct {
	spec  *[1024]float64
	pos   [1024]int32
	absv  [1024]float64
	xrpow [1024]float64
	q     [1024]int
	qtmp  [1024]int

	bands   []encBand
	memo    []bandMemo
	epoch   uint32
	maxSfb  int
	nGroups int
	lenBits int

	globalGain int
	demand     float64
}

func (cq *chanQuant) buildBands(spec *[1024]float64, groupLen []int, swb []uint16, maxSfb int, thr func(g, sfb int) float64, short bool) {
	cq.spec = spec
	cq.maxSfb = maxSfb
	cq.nGroups = len(groupLen)
	cq.lenBits = 5
	if short {
		cq.lenBits = 3
	}
	cq.bands = cq.bands[:0]
	n := 0
	winBase := 0
	for g, L := range groupLen {
		for sfb := 0; sfb < maxSfb; sfb++ {
			b := encBand{off: n}
			lo, hi := int(swb[sfb]), int(swb[sfb+1])
			for w := 0; w < L; w++ {
				base := (winBase + w) * 128
				for k := lo; k < hi; k++ {
					v := spec[base+k]
					av := math.Abs(v)
					cq.pos[n] = int32(base + k)
					cq.absv[n] = av
					cq.xrpow[n] = math.Sqrt(av * math.Sqrt(av))
					if av > b.maxAbs {
						b.maxAbs = av
					}
					b.energy += av * av
					n++
				}
			}
			b.n = n - b.off
			b.thr = thr(g, sfb)
			b.minSf = minSfFor(b.maxAbs)
			cq.bands = append(cq.bands, b)
		}
		winBase += L
	}
	need := len(cq.bands) * 256
	if cap(cq.memo) < need {
		cq.memo = make([]bandMemo, need)
	}
	cq.memo = cq.memo[:need]
	cq.epoch++

	d := 0.0
	for _, b := range cq.bands {
		if b.energy > b.thr && b.thr > 0 {
			d += float64(b.n) * math.Log2(b.energy/b.thr)
		}
	}
	cq.demand = d
}

func minSfFor(maxAbs float64) int {
	if maxAbs < 1 {
		return 0
	}
	sf := int(math.Ceil(100 + (16.0/3.0)*(0.75*math.Log2(maxAbs)-math.Log2(8191.4))))
	if sf < 0 {
		return 0
	}
	if sf > sfClampMax {
		return sfClampMax
	}
	return sf
}

const sfClampMax = 255

func (b *encBand) sfFor(delta int) int {
	sf := delta - b.amp
	if sf < b.minSf {
		sf = b.minSf
	}
	if sf > sfClampMax {
		sf = sfClampMax
	}
	return sf
}

func (cq *chanQuant) quantAt(b *encBand, sf int, q []int) int {
	f := math.Exp2(-0.1875 * float64(sf-100))
	maxQ := 0
	for i := 0; i < b.n; i++ {
		m := int(cq.xrpow[b.off+i]*f + 0.4054)
		if m > escMaxValue {
			m = escMaxValue
		}
		if m > maxQ {
			maxQ = m
		}
		q[i] = m
	}
	return maxQ
}

var bookTiers = [...][2]int{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9, 10}, {11, 0}}
var tierMax = [...]int{1, 2, 4, 7, 12, escMaxValue}

func (cq *chanQuant) bandAt(bi, sf int) *bandMemo {
	m := &cq.memo[bi*256+sf]
	if m.epoch == cq.epoch {
		return m
	}
	b := &cq.bands[bi]
	q := cq.qtmp[:b.n]
	maxQ := cq.quantAt(b, sf, q)

	if maxQ == 0 {
		*m = bandMemo{epoch: cq.epoch, zero: true, noise: b.energy}
		return m
	}
	gain := math.Exp2(0.25 * float64(sf-sfOffset))
	noise := 0.0
	for i := 0; i < b.n; i++ {
		e := cq.absv[b.off+i] - iquant(float64(q[i]))*gain
		noise += e * e
	}

	tier := 0
	for tierMax[tier] < maxQ {
		tier++
	}
	bestBits, bestCb := -1, 0
	try := func(cb int) {
		if cb == 0 {
			return
		}
		bits := specRunBits(cb, q)
		if bits >= 0 && (bestBits < 0 || bits < bestBits) {
			bestBits, bestCb = bits, cb
		}
	}
	try(bookTiers[tier][0])
	try(bookTiers[tier][1])
	if tier+1 < len(bookTiers) {
		try(bookTiers[tier+1][0])
		try(bookTiers[tier+1][1])
	}
	if tierMax[tier] < 16 {
		try(escHCB)
	}
	*m = bandMemo{epoch: cq.epoch, bits: int32(bestBits), cb: int8(bestCb), noise: noise}
	return m
}

func (cq *chanQuant) totalBits(delta int) int {
	total := 0
	lenEsc := 1<<uint(cq.lenBits) - 1
	perGroup := cq.maxSfb
	prevSf := -1
	for g := 0; g < cq.nGroups; g++ {
		runCb := -1
		runLen := 0
		flush := func() {
			if runLen > 0 {
				total += 4 + cq.lenBits*(runLen/lenEsc+1)
			}
		}
		for k := 0; k < perGroup; k++ {
			bi := g*perGroup + k
			b := &cq.bands[bi]
			m := cq.bandAt(bi, b.sfFor(delta))
			cb := 0
			if !m.zero {
				cb = int(m.cb)
				total += int(m.bits)
				sf := b.sfFor(delta)
				if prevSf >= 0 {
					d := sf - prevSf
					if d < -60 {
						d = -60
					} else if d > 60 {
						d = 60
					}
					total += sfDeltaBits(d)
				}
				prevSf = sf
			}
			if cb != runCb {
				flush()
				runCb, runLen = cb, 1
			} else {
				runLen++
			}
		}
		flush()
	}
	return total
}

func (cq *chanQuant) rateSearch(budget int) int {
	lo, hi := 0, sfSearchMax
	for lo < hi {
		mid := (lo + hi) / 2
		if cq.totalBits(mid) <= budget {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

func (cq *chanQuant) quantizeChannel(budget, hardCap int) {
	if hardCap < 0 {
		hardCap = 0
	}
	if budget < 0 {
		budget = 0
	}
	if budget > hardCap {
		budget = hardCap
	}
	bestDelta := -1
	bestScore := math.Inf(1)
	delta := 0
	for iter := 0; ; iter++ {
		delta = cq.rateSearch(budget)
		score := 0.0
		violated := false
		for bi := range cq.bands {
			b := &cq.bands[bi]
			m := cq.bandAt(bi, b.sfFor(delta))
			if b.thr > 0 && m.noise > b.thr {
				score += math.Log2(m.noise / b.thr)
				if b.amp < ampMax && b.sfFor(delta) > b.minSf {
					violated = true
				}
			}
		}
		if score < bestScore {
			bestScore = score
			bestDelta = delta
		}
		if !violated || iter >= maxAmpIter {
			break
		}
		for bi := range cq.bands {
			b := &cq.bands[bi]
			m := cq.bandAt(bi, b.sfFor(delta))
			if b.thr > 0 && m.noise > b.thr && b.amp < ampMax && b.sfFor(delta) > b.minSf {
				b.amp++
			}
		}
	}
	if bestDelta < 0 || cq.totalBits(bestDelta) > hardCap {
		bestDelta = delta
	}
	cq.assemble(bestDelta)
}

func (cq *chanQuant) assemble(delta int) {
	prevSf := -1
	cq.globalGain = 0
	firstCoded := true
	for bi := range cq.bands {
		b := &cq.bands[bi]
		sf := b.sfFor(delta)
		m := cq.bandAt(bi, sf)
		if m.zero {
			b.cb = 0
			b.sf = 0
			for i := 0; i < b.n; i++ {
				cq.q[b.off+i] = 0
			}
			continue
		}
		if prevSf >= 0 {
			if sf < prevSf-60 {
				sf = prevSf - 60
			} else if sf > prevSf+60 {
				sf = prevSf + 60
			}
		}
		b.sf = sf
		b.cb = int(cq.bandAt(bi, sf).cb)
		if firstCoded {
			cq.globalGain = sf
			firstCoded = false
		}
		prevSf = sf
		q := cq.qtmp[:b.n]
		cq.quantAt(b, sf, q)
		for i := 0; i < b.n; i++ {
			if math.Signbit(cq.spec[cq.pos[b.off+i]]) {
				cq.q[b.off+i] = -q[i]
			} else {
				cq.q[b.off+i] = q[i]
			}
		}
	}
	if firstCoded {
		cq.globalGain = 100
	}
}
