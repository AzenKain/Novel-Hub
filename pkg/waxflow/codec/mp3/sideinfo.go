package mp3

const (
	blockNormal = 0
	blockStart  = 1
	blockShort  = 2
	blockStop   = 3
)

type grInfo struct {
	part23Len    int
	bigValues    int
	globalGain   int
	scfCompress  int
	blockType    int
	mixed        bool
	tableSelect  [3]int
	subblockGain [3]int
	regionCount  [2]int
	preflag      bool
	scfScale     int
	count1Table  bool
}

type sideInfo struct {
	mainDataBegin int
	scfsi         [2][4]bool
	gr            [2][2]grInfo
}

func parseSideInfo(h Header, b []byte, si *sideInfo) bool {
	r := bitReader{data: b}
	granules := 2
	if h.Version == MPEG1 {
		si.mainDataBegin = int(r.bits(9))
		if h.Channels == 1 {
			r.bits(5)
		} else {
			r.bits(3)
		}
		for ch := 0; ch < h.Channels; ch++ {
			for g := 0; g < 4; g++ {
				si.scfsi[ch][g] = r.bit() == 1
			}
		}
	} else {
		granules = 1
		si.mainDataBegin = int(r.bits(8))
		r.bits(uint(h.Channels))
	}

	for gri := 0; gri < granules; gri++ {
		for ch := 0; ch < h.Channels; ch++ {
			g := &si.gr[gri][ch]
			*g = grInfo{}
			g.part23Len = int(r.bits(12))
			g.bigValues = int(r.bits(9))
			if g.bigValues > 288 {
				return false
			}
			g.globalGain = int(r.bits(8))
			if h.Version == MPEG1 {
				g.scfCompress = int(r.bits(4))
			} else {
				g.scfCompress = int(r.bits(9))
			}
			if r.bit() == 1 {
				g.blockType = int(r.bits(2))
				if g.blockType == blockNormal {
					return false
				}
				g.mixed = r.bit() == 1
				g.tableSelect[0] = int(r.bits(5))
				g.tableSelect[1] = int(r.bits(5))
				g.subblockGain[0] = int(r.bits(3))
				g.subblockGain[1] = int(r.bits(3))
				g.subblockGain[2] = int(r.bits(3))
				g.regionCount[0] = 7
				if g.blockType == blockShort && !g.mixed {
					g.regionCount[0] = 8
				}
				g.regionCount[1] = 36
			} else {
				g.blockType = blockNormal
				g.tableSelect[0] = int(r.bits(5))
				g.tableSelect[1] = int(r.bits(5))
				g.tableSelect[2] = int(r.bits(5))
				g.regionCount[0] = int(r.bits(4))
				g.regionCount[1] = int(r.bits(3))
			}
			if h.Version == MPEG1 {
				g.preflag = r.bit() == 1
			} else {
				g.preflag = g.scfCompress >= 500
			}
			g.scfScale = int(r.bit())
			g.count1Table = r.bit() == 1
		}
	}
	return !r.err
}
