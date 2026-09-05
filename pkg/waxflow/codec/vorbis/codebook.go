package vorbis

import (
	"math"
	"math/bits"
)

type codebook struct {
	dimensions int
	entries    int
	tree       huffTree

	lookupType    int
	minimum       float32
	delta         float32
	sequenceP     bool
	multiplicands []float32
	lookupValues  int
}

func ilog(x int) int {
	if x <= 0 {
		return 0
	}
	return bits.Len(uint(x))
}

func float32Unpack(x uint32) float64 {
	mantissa := float64(x & 0x1fffff)
	exponent := int((x & 0x7fe00000) >> 21)
	if x&0x80000000 != 0 {
		mantissa = -mantissa
	}
	return mantissa * math.Ldexp(1, exponent-788)
}

func lookup1Values(entries, dimensions int) int {
	v := 0
	for {
		next := v + 1
		p := 1.0
		for i := 0; i < dimensions; i++ {
			p *= float64(next)
			if p > float64(entries) {
				break
			}
		}
		if p > float64(entries) {
			return v
		}
		v = next
		if v > entries {
			return v
		}
	}
}

func parseCodebook(r *bitReader) (codebook, error) {
	var c codebook
	if sync := r.read(24); sync != 0x564342 {
		return c, malformed("codebook sync pattern %06x", sync)
	}
	c.dimensions = int(r.read(16))
	c.entries = int(r.read(24))
	if c.entries > maxCodebooks {
		return c, malformed("codebook has %d entries (max %d)", c.entries, maxCodebooks)
	}
	if c.entries == 0 {
		return c, malformed("codebook has zero entries")
	}
	if c.dimensions == 0 {
		return c, malformed("codebook has zero dimensions")
	}

	lengths := make([]uint8, c.entries)
	ordered := r.bit() == 1
	if ordered {
		length := int(r.read(5)) + 1
		entry := 0
		for entry < c.entries {
			bitsNeeded := ilog(c.entries - entry)
			num := int(r.read(bitsNeeded))
			if entry+num > c.entries {
				return c, malformed("ordered codebook overruns entry count")
			}
			for i := 0; i < num; i++ {
				lengths[entry+i] = uint8(length)
			}
			entry += num
			length++
			if length > maxCodewordLen+1 {
				return c, malformed("ordered codebook length exceeds %d", maxCodewordLen)
			}
		}
	} else {
		sparse := r.bit() == 1
		for i := 0; i < c.entries; i++ {
			if sparse && r.bit() == 0 {
				lengths[i] = 0
				continue
			}
			lengths[i] = uint8(r.read(5)) + 1
		}
	}
	if r.eof {
		return c, malformed("codebook truncated in lengths")
	}
	if err := c.tree.build(lengths); err != nil {
		return c, err
	}

	c.lookupType = int(r.read(4))
	switch c.lookupType {
	case 0:
	case 1, 2:
		c.minimum = float32(float32Unpack(r.read(32)))
		c.delta = float32(float32Unpack(r.read(32)))
		valueBits := int(r.read(4)) + 1
		c.sequenceP = r.bit() == 1
		var count int
		if c.lookupType == 1 {
			c.lookupValues = lookup1Values(c.entries, c.dimensions)
			count = c.lookupValues
		} else {
			count = c.entries * c.dimensions
		}
		if count < 0 || count > 1<<24 {
			return c, malformed("codebook lookup table too large (%d)", count)
		}
		c.multiplicands = make([]float32, count)
		for i := range c.multiplicands {
			c.multiplicands[i] = float32(r.read(valueBits))
		}
		if r.eof {
			return c, malformed("codebook truncated in lookup table")
		}
	default:
		return c, malformed("codebook lookup type %d", c.lookupType)
	}
	return c, nil
}

func (c *codebook) decodeScalar(r *bitReader) (int, error) {
	return c.tree.decode(r)
}

func (c *codebook) decodeVector(r *bitReader, out []float32) error {
	entry, err := c.tree.decode(r)
	if err != nil {
		return err
	}
	c.valueVector(entry, out)
	return nil
}

func (c *codebook) valueVector(entry int, out []float32) {
	switch c.lookupType {
	case 1:
		last := float32(0)
		divisor := 1
		for i := 0; i < c.dimensions; i++ {
			off := (entry / divisor) % c.lookupValues
			out[i] = c.multiplicands[off]*c.delta + c.minimum + last
			if c.sequenceP {
				last = out[i]
			}
			divisor *= c.lookupValues
		}
	case 2:
		last := float32(0)
		off := entry * c.dimensions
		for i := 0; i < c.dimensions; i++ {
			out[i] = c.multiplicands[off+i]*c.delta + c.minimum + last
			if c.sequenceP {
				last = out[i]
			}
		}
	default:
		for i := range out {
			out[i] = 0
		}
	}
}

type huffTree struct {
	child []int32
}

func (t *huffTree) build(lengths []uint8) error {
	codes, ok := assignCodewords(lengths)
	if !ok {
		return malformed("over-subscribed codebook")
	}
	t.child = make([]int32, 2)
	next := int32(1)
	used := 0
	single := -1
	for i, l := range lengths {
		if l == 0 {
			continue
		}
		used++
		single = i
		node := int32(0)
		code := codes[i]
		for b := 0; b < int(l); b++ {
			bit := (code >> (31 - uint(b))) & 1
			slot := node*2 + int32(bit)
			if b == int(l)-1 {
				if t.child[slot] != 0 {
					return malformed("codebook is not prefix-free")
				}
				t.child[slot] = -(int32(i) + 1)
				break
			}
			if t.child[slot] == 0 {
				t.child[slot] = next
				next++
				t.child = append(t.child, 0, 0)
			} else if t.child[slot] < 0 {
				return malformed("codebook is not prefix-free")
			}
			node = t.child[slot]
		}
	}
	if used == 1 {
		leaf := -(int32(single) + 1)
		t.child = []int32{leaf, leaf}
	}
	return nil
}

func (t *huffTree) decode(r *bitReader) (int, error) {
	node := int32(0)
	for {
		slot := node*2 + int32(r.bit())
		if r.eof {
			return 0, errEndOfPacket
		}
		v := t.child[slot]
		switch {
		case v < 0:
			return int(-v - 1), nil
		case v == 0:
			return 0, malformed("codeword not in codebook")
		default:
			node = v
		}
	}
}

func assignCodewords(lengths []uint8) (codes []uint32, ok bool) {
	n := len(lengths)
	codes = make([]uint32, n)
	var available [33]uint32
	k := 0
	for k < n && lengths[k] == 0 {
		k++
	}
	if k == n {
		return codes, true
	}
	l0 := int(lengths[k])
	codes[k] = 0
	for i := 1; i <= l0; i++ {
		available[i] = 1 << (32 - uint(i))
	}
	for i := k + 1; i < n; i++ {
		l := int(lengths[i])
		if l == 0 {
			continue
		}
		z := l
		for z > 0 && available[z] == 0 {
			z--
		}
		if z == 0 {
			return codes, false
		}
		res := available[z]
		available[z] = 0
		codes[i] = res
		if z != l {
			for y := l; y > z; y-- {
				available[y] = res + (1 << (32 - uint(y)))
			}
		}
	}
	return codes, true
}
