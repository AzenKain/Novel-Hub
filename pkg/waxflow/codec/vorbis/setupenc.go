package vorbis

import "math/bits"

type encConfig struct {
	channels   int
	rate       int
	blockSizes [2]int

	specs []bookSpec
	books []*encBook

	floors         [2]*floor1
	floorRangebits [2]int
	residues       [2]*residue
	mappings       [2]*mapping
	modes          []mode
	modeBits       int
}

const (
	slotShort = 0
	slotLong  = 1
)

var headerOrder = [2]int{slotLong, slotShort}

func (c *encConfig) slotFor(n int) int {
	if n == c.blockSizes[slotLong] {
		return slotLong
	}
	return slotShort
}

func modeForSlot(slot int) int {
	if slot == slotLong {
		return 0
	}
	return 1
}

func blockLog(n int) int { return bits.Len(uint(n)) - 1 }

func newEncConfig(channels, rate int) *encConfig {
	c := &encConfig{
		channels:   channels,
		rate:       rate,
		blockSizes: [2]int{shortBlock, longBlock},
	}
	c.specs = encSpecs
	c.books = encBooks
	c.floors[slotLong], c.floorRangebits[slotLong] = buildFloor1(longBlock/2, floorPartitions)
	c.floors[slotShort], c.floorRangebits[slotShort] = buildFloor1(shortBlock/2, shortFloorPartitions)
	c.residues[slotLong] = buildResidue(longBlock / 2)
	c.residues[slotShort] = buildResidue(shortBlock / 2)
	c.mappings[slotLong] = &mapping{submaps: []submap{{floor: 0, residue: 0}}, mux: make([]int, channels)}
	c.mappings[slotShort] = &mapping{submaps: []submap{{floor: 1, residue: 1}}, mux: make([]int, channels)}
	if channels == 2 {
		for _, m := range c.mappings {
			m.couplingMag = []int{0}
			m.couplingAng = []int{1}
		}
	}
	c.modes = []mode{{blockflag: true, mapping: 0}, {blockflag: false, mapping: 1}}
	c.modeBits = ilog(len(c.modes) - 1)
	return c
}

const (
	shortBlock = 256
	longBlock  = 2048
)

const resPartSize = 32

func buildResidue(n2 int) *residue {
	r := &residue{
		kind:      1,
		begin:     0,
		end:       n2,
		partSize:  resPartSize,
		classes:   numResClass,
		classbook: bookResClass,
		maxPass:   4,
	}
	r.books = make([][]int, numResClass)
	r.books[classSkip] = []int{-1, -1, -1, -1, -1, -1, -1, -1}
	r.books[classNoise] = []int{bookResNoise, -1, -1, -1, -1, -1, -1, -1}
	r.books[classCoarse] = []int{bookResCoarse, -1, -1, -1, -1, -1, -1, -1}
	r.books[classMed] = []int{bookResCoarse, bookResR1, -1, -1, -1, -1, -1, -1}
	r.books[classFine] = []int{bookResCoarse, bookResR1, bookResR2, -1, -1, -1, -1, -1}
	r.books[classSuper] = []int{bookResCoarse, bookResR1, bookResR2, bookResR3, -1, -1, -1, -1}
	return r
}

func writeString(w *bitWriter, typ byte, sig string) {
	w.writeBits(8, uint32(typ))
	for i := 0; i < len(sig); i++ {
		w.writeBits(8, uint32(sig[i]))
	}
}

func (c *encConfig) idHeader() []byte {
	var w bitWriter
	writeString(&w, 0x01, "vorbis")
	w.writeBits(32, 0)
	w.writeBits(8, uint32(c.channels))
	w.writeBits(32, uint32(c.rate))
	w.writeBits(32, 0)
	w.writeBits(32, 0)
	w.writeBits(32, 0)
	w.writeBits(4, uint32(blockLog(c.blockSizes[0])))
	w.writeBits(4, uint32(blockLog(c.blockSizes[1])))
	w.writeBit(1)
	return w.bytes()
}

func commentHeader(vendor string, comments []string) []byte {
	var w bitWriter
	writeString(&w, 0x03, "vorbis")
	w.writeBits(32, uint32(len(vendor)))
	for i := 0; i < len(vendor); i++ {
		w.writeBits(8, uint32(vendor[i]))
	}
	w.writeBits(32, uint32(len(comments)))
	for _, cm := range comments {
		w.writeBits(32, uint32(len(cm)))
		for i := 0; i < len(cm); i++ {
			w.writeBits(8, uint32(cm[i]))
		}
	}
	w.writeBit(1)
	return w.bytes()
}

func (c *encConfig) setupHeader() []byte {
	var w bitWriter
	writeString(&w, 0x05, "vorbis")

	w.writeBits(8, uint32(len(c.specs)-1))
	for i := range c.specs {
		writeCodebook(&w, c.specs[i])
	}

	w.writeBits(6, 0)
	w.writeBits(16, 0)

	w.writeBits(6, uint32(len(headerOrder)-1))
	for _, slot := range headerOrder {
		w.writeBits(16, 1)
		writeFloor1(&w, c.floors[slot], c.floorRangebits[slot])
	}

	w.writeBits(6, uint32(len(headerOrder)-1))
	for _, slot := range headerOrder {
		writeResidue(&w, c.residues[slot])
	}

	w.writeBits(6, uint32(len(headerOrder)-1))
	for _, slot := range headerOrder {
		w.writeBits(16, 0)
		writeMapping(&w, c.mappings[slot], c.channels)
	}

	w.writeBits(6, uint32(len(c.modes)-1))
	for _, m := range c.modes {
		w.writeBit(boolBit(m.blockflag))
		w.writeBits(16, 0)
		w.writeBits(16, 0)
		w.writeBits(8, uint32(m.mapping))
	}

	w.writeBit(1)
	return w.bytes()
}

func (c *encConfig) codecConfig(vendor string, comments []string) []byte {
	return PackHeaders(c.idHeader(), commentHeader(vendor, comments), c.setupHeader())
}

func writeFloor1(w *bitWriter, f *floor1, rangebits int) {
	w.writeBits(5, uint32(len(f.partitionClass)))
	maxClass := -1
	for _, cls := range f.partitionClass {
		w.writeBits(4, uint32(cls))
		if cls > maxClass {
			maxClass = cls
		}
	}
	for cls := 0; cls <= maxClass; cls++ {
		w.writeBits(3, uint32(f.classDims[cls]-1))
		w.writeBits(2, uint32(f.classSubclasses[cls]))
		if f.classSubclasses[cls] > 0 {
			w.writeBits(8, uint32(f.classMasterbook[cls]))
		}
		for _, b := range f.classSubbooks[cls] {
			w.writeBits(8, uint32(b+1))
		}
	}
	w.writeBits(2, uint32(f.multiplier-1))
	w.writeBits(4, uint32(rangebits))
	for i := 2; i < len(f.xs); i++ {
		w.writeBits(uint(rangebits), uint32(f.xs[i]))
	}
}

func writeResidue(w *bitWriter, r *residue) {
	w.writeBits(16, uint32(r.kind))
	w.writeBits(24, uint32(r.begin))
	w.writeBits(24, uint32(r.end))
	w.writeBits(24, uint32(r.partSize-1))
	w.writeBits(6, uint32(r.classes-1))
	w.writeBits(8, uint32(r.classbook))
	cascade := make([]int, r.classes)
	for i := 0; i < r.classes; i++ {
		for j := 0; j < 8; j++ {
			if r.books[i][j] >= 0 {
				cascade[i] |= 1 << uint(j)
			}
		}
		low := cascade[i] & 7
		high := cascade[i] >> 3
		w.writeBits(3, uint32(low))
		if high > 0 {
			w.writeBit(1)
			w.writeBits(5, uint32(high))
		} else {
			w.writeBit(0)
		}
	}
	for i := 0; i < r.classes; i++ {
		for j := 0; j < 8; j++ {
			if cascade[i]&(1<<uint(j)) != 0 {
				w.writeBits(8, uint32(r.books[i][j]))
			}
		}
	}
}

func writeMapping(w *bitWriter, m *mapping, channels int) {
	submaps := len(m.submaps)
	if submaps > 1 {
		w.writeBit(1)
		w.writeBits(4, uint32(submaps-1))
	} else {
		w.writeBit(0)
	}
	if len(m.couplingMag) > 0 {
		w.writeBit(1)
		w.writeBits(8, uint32(len(m.couplingMag)-1))
		magBits := ilog(channels - 1)
		for i := range m.couplingMag {
			w.writeBits(uint(magBits), uint32(m.couplingMag[i]))
			w.writeBits(uint(magBits), uint32(m.couplingAng[i]))
		}
	} else {
		w.writeBit(0)
	}
	w.writeBits(2, 0)
	if submaps > 1 {
		for ch := 0; ch < channels; ch++ {
			w.writeBits(4, uint32(m.mux[ch]))
		}
	}
	for _, sm := range m.submaps {
		w.writeBits(8, 0)
		w.writeBits(8, uint32(sm.floor))
		w.writeBits(8, uint32(sm.residue))
	}
}
