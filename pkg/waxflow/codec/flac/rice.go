package flac

const (
	riceMaxParam4 = 14
	riceMaxParam5 = 30
)

type ricePlan struct {
	partOrder int
	method    int
	params    []uint8
	bits      int64
}

type riceScratch struct {
	zig    []uint64
	sums   []uint64
	merged []uint64
	params []uint8
	best   []uint8
}

func zigzag(v int64) uint64 {
	return uint64(v<<1) ^ uint64(v>>63)
}

func bestRiceParam(sum uint64, count int) (uint, int64) {
	if count == 0 {
		return 0, 0
	}
	k := uint(0)
	for k < riceMaxParam5 && uint64(count)<<(k+1) < sum {
		k++
	}
	bits := func(k uint) int64 {
		return int64(count)*int64(k+1) + int64(sum>>k)
	}
	best, bestBits := k, bits(k)
	if k > 0 {
		if b := bits(k - 1); b < bestBits {
			best, bestBits = k-1, b
		}
	}
	if k < riceMaxParam5 {
		if b := bits(k + 1); b < bestBits {
			best, bestBits = k+1, b
		}
	}
	return best, bestBits
}

func planRice(res []int64, order, blockSize, maxPart int, s *riceScratch) ricePlan {
	top := maxPart
	for top > 0 && (blockSize%(1<<top) != 0 || blockSize>>top < order) {
		top--
	}

	if cap(s.zig) < blockSize {
		s.zig = make([]uint64, blockSize)
		s.sums = make([]uint64, 1<<8)
		s.merged = make([]uint64, 1<<8)
		s.params = make([]uint8, 1<<8)
		s.best = make([]uint8, 1<<8)
	}
	zig := s.zig[:blockSize]
	for i := order; i < blockSize; i++ {
		zig[i] = zigzag(res[i])
	}

	parts := 1 << top
	size := blockSize >> top
	sums := s.sums[:parts]
	for p := 0; p < parts; p++ {
		start := p * size
		if p == 0 {
			start = order
		}
		var sum uint64
		for _, u := range zig[start : (p+1)*size] {
			sum += u
		}
		sums[p] = sum
	}

	plan := ricePlan{partOrder: -1}
	for po := top; ; po-- {
		parts := 1 << po
		size := blockSize >> po
		total := int64(0)
		maxK := uint(0)
		params := s.params[:parts]
		for p := 0; p < parts; p++ {
			count := size
			if p == 0 {
				count -= order
			}
			k, bits := bestRiceParam(sums[p], count)
			params[p] = uint8(k)
			total += bits
			if k > maxK {
				maxK = k
			}
		}
		method := 0
		paramBits := 4
		if maxK > riceMaxParam4 {
			method = 1
			paramBits = 5
		}
		total += int64(parts * paramBits)
		if plan.partOrder < 0 || total < plan.bits {
			best := s.best[:parts]
			copy(best, params)
			plan = ricePlan{partOrder: po, method: method, params: best, bits: total}
		}
		if po == 0 {
			break
		}
		merged := s.merged[:parts/2]
		for p := range merged {
			merged[p] = sums[2*p] + sums[2*p+1]
		}
		s.sums, s.merged = s.merged, s.sums
		sums = s.sums[:parts/2]
	}
	return plan
}

func (w *bitWriter) writeRice(res []int64, order, blockSize int, plan ricePlan) {
	w.writeBits(2, uint64(plan.method))
	w.writeBits(4, uint64(plan.partOrder))
	paramBits := uint(4 + plan.method)
	size := blockSize >> plan.partOrder
	pos := order
	for p, k := range plan.params {
		w.writeBits(paramBits, uint64(k))
		end := (p + 1) * size
		for _, v := range res[pos:end] {
			u := zigzag(v)
			w.writeUnary(u >> k)
			w.writeBits(uint(k), u)
		}
		pos = end
	}
}
