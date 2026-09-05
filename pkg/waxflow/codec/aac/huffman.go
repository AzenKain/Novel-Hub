package aac

type huffBook struct {
	tree []int32
}

var (
	spectralBooks   [11]huffBook
	scalefactorBook huffBook
)

func init() {
	for cb := 0; cb < 11; cb++ {
		codes := make([]uint32, len(spectralCodes[cb]))
		for i, c := range spectralCodes[cb] {
			codes[i] = uint32(c)
		}
		spectralBooks[cb] = buildBook(codes, spectralBits[cb])
	}
	scalefactorBook = buildBook(scalefactorCodes[:], scalefactorBits[:])
}

func buildBook(codes []uint32, bits []uint8) huffBook {
	tree := make([]int32, 2)
	for i := range codes {
		cw, l := codes[i], int(bits[i])
		node := 0
		for b := l - 1; b >= 0; b-- {
			slot := 2*node + int(cw>>uint(b)&1)
			if b == 0 {
				tree[slot] = -(int32(i) + 1)
				break
			}
			if tree[slot] == 0 {
				tree[slot] = int32(len(tree) / 2)
				tree = append(tree, 0, 0)
			}
			node = int(tree[slot])
		}
	}
	return huffBook{tree}
}

func (b *huffBook) decode(r *bitReader) (int, bool) {
	node := 0
	for {
		v := b.tree[2*node+int(r.bit())]
		switch {
		case v < 0:
			return int(-v - 1), true
		case v == 0:
			return 0, false
		default:
			node = int(v)
		}
	}
}

func decodeSpectral(r *bitReader, cb int, out []int) bool {
	idx, ok := spectralBooks[cb-1].decode(r)
	if !ok {
		return false
	}
	dim := hcbDim[cb-1]
	mod := hcbMod[cb-1]
	for d := dim - 1; d >= 0; d-- {
		out[d] = idx % mod
		idx /= mod
	}
	if !hcbUnsigned[cb-1] {
		off := hcbOff[cb-1]
		for d := 0; d < dim; d++ {
			out[d] -= off
		}
		return true
	}
	var neg [4]bool
	for d := 0; d < dim; d++ {
		if out[d] != 0 {
			neg[d] = r.bit() != 0
		}
	}
	if cb == escHCB {
		for d := 0; d < dim; d++ {
			if out[d] == 16 {
				out[d] = decodeEscape(r)
			}
		}
	}
	for d := 0; d < dim; d++ {
		if neg[d] {
			out[d] = -out[d]
		}
	}
	return true
}

func decodeEscape(r *bitReader) int {
	n := 0
	for r.bit() == 1 {
		n++
		if n > 24 {
			break
		}
	}
	return (1 << uint(n+4)) + int(r.read(uint(n+4)))
}

func decodeScalefactor(r *bitReader) (int, bool) {
	idx, ok := scalefactorBook.decode(r)
	if !ok {
		return 0, false
	}
	return idx - 60, true
}
