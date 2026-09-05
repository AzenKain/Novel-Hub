package mp3

type hcode struct {
	bits uint32
	n    uint8
}

var bigEnc [32][]hcode

var bigMax [32]int

var cnt1Enc [2][16]hcode

func init() {
	for t := 1; t < 32; t++ {
		ht := &huffTables[t]
		if ht.treeLen == 0 {
			continue
		}
		row := make([]hcode, 256)
		bigMax[t] = walkEncode(huffTree[ht.off:ht.off+ht.treeLen], func(val byte, c hcode) {
			row[val] = c
		})
		bigEnc[t] = row
	}
	for i, t := range [2]int{32, 33} {
		ht := &huffTables[t]
		walkEncode(huffTree[ht.off:ht.off+ht.treeLen], func(val byte, c hcode) {
			cnt1Enc[i][val] = c
		})
	}
}

func walkEncode(tree []uint16, emit func(val byte, c hcode)) int {
	maxLeaf := 0
	type frame struct {
		point int
		code  uint32
		n     uint8
	}
	stack := []frame{{0, 0, 0}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := tree[f.point]
		if node&0xFF00 == 0 {
			val := byte(node & 0xFF)
			emit(val, hcode{bits: f.code, n: f.n})
			if hi := int(val >> 4); hi > maxLeaf {
				maxLeaf = hi
			}
			if lo := int(val & 0xF); lo > maxLeaf {
				maxLeaf = lo
			}
			continue
		}
		p0 := f.point
		for tree[p0]>>8 >= 250 {
			p0 += int(tree[p0] >> 8)
		}
		p0 += int(tree[p0] >> 8)
		p1 := f.point
		for tree[p1]&0xFF >= 250 {
			p1 += int(tree[p1] & 0xFF)
		}
		p1 += int(tree[p1] & 0xFF)
		stack = append(stack,
			frame{p0, f.code << 1, f.n + 1},
			frame{p1, f.code<<1 | 1, f.n + 1})
	}
	return maxLeaf
}

func tableLimit(t int) int {
	if t < 0 || t >= 32 || bigEnc[t] == nil {
		if t == 0 {
			return 0
		}
		return -1
	}
	if lb := huffTables[t].linbits; lb != 0 {
		return 15 + (1 << lb) - 1
	}
	return bigMax[t]
}

var bigCandidates = [...]int{
	1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15,
	16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
}

var (
	noLinbitsTables = []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15}
	treeA           = []int{16, 17, 18, 19, 20, 21, 22, 23}
	treeB           = []int{24, 25, 26, 27, 28, 29, 30, 31}
)

func pairBits(t, ax, ay int) int {
	lb := huffTables[t].linbits
	xi, yi := ax, ay
	extra := 0
	if lb != 0 {
		if ax >= 15 {
			xi = 15
			extra += lb
		}
		if ay >= 15 {
			yi = 15
			extra += lb
		}
	} else if ax > bigMax[t] || ay > bigMax[t] {
		return -1
	}
	c := bigEnc[t][xi<<4|yi]
	if c.n == 0 && !(xi == 0 && yi == 0) {
		return -1
	}
	bits := int(c.n) + extra
	if ax != 0 {
		bits++
	}
	if ay != 0 {
		bits++
	}
	return bits
}

func (w *bitWriter) writePair(t, x, y int) {
	if bigEnc[t] == nil {
		return
	}
	lb := huffTables[t].linbits
	ax, ay := abs(x), abs(y)
	xi, yi := ax, ay
	if lb != 0 {
		if ax >= 15 {
			xi = 15
		}
		if ay >= 15 {
			yi = 15
		}
	}
	c := bigEnc[t][xi<<4|yi]
	w.writeBits(uint(c.n), c.bits)
	if lb != 0 && ax >= 15 {
		w.writeBits(uint(lb), uint32(ax-15))
	}
	if x != 0 {
		w.writeBits(1, boolBit(x < 0))
	}
	if lb != 0 && ay >= 15 {
		w.writeBits(uint(lb), uint32(ay-15))
	}
	if y != 0 {
		w.writeBits(1, boolBit(y < 0))
	}
}

func quadBits(sel int, v, w, x, y int) int {
	idx := abs1(v)<<3 | abs1(w)<<2 | abs1(x)<<1 | abs1(y)
	bits := int(cnt1Enc[sel][idx].n)
	for _, c := range [4]int{v, w, x, y} {
		if c != 0 {
			bits++
		}
	}
	return bits
}

func (bw *bitWriter) writeQuad(sel, v, w, x, y int) {
	idx := abs1(v)<<3 | abs1(w)<<2 | abs1(x)<<1 | abs1(y)
	c := cnt1Enc[sel][idx]
	bw.writeBits(uint(c.n), c.bits)
	for _, comp := range [4]int{v, w, x, y} {
		if comp != 0 {
			bw.writeBits(1, boolBit(comp < 0))
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func abs1(x int) int {
	if x != 0 {
		return 1
	}
	return 0
}

func boolBit(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}
