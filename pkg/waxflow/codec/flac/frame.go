package flac

// Frame header parsing (RFC 9639 section 9.1).

const (
	assignLeftSide  = 8
	assignRightSide = 9
	assignMidSide   = 10
)

// MaxFrameHeaderLen bounds a frame header: 4 fixed bytes, up to 7 coded number bytes, up to 2 block size bytes, up to 2 sample rate bytes, and the CRC-8.
const MaxFrameHeaderLen = 16

// FrameInfo is a parsed frame header.
type FrameInfo struct {
	Variable  bool
	Coded     uint64
	BlockSize int
	Rate      int
	Channels  int
	Bits      int

	assign int
	hdrLen int
}

// Numbering resolves frames' coded numbers to sample positions.
type Numbering struct {
	SampleCoded bool
	ConstBlock  int
}

// Numbering latches the stream's coded-number semantics from the first frame.
func (si StreamInfo) Numbering(first FrameInfo) Numbering {
	return Numbering{
		SampleCoded: first.Variable || si.MinBlock != si.MaxBlock,
		ConstBlock:  first.BlockSize,
	}
}

// Start returns the frame's first sample position.
func (n Numbering) Start(fi FrameInfo) int64 {
	if n.SampleCoded {
		return int64(fi.Coded)
	}
	return int64(fi.Coded) * int64(n.ConstBlock)
}

// Next returns the coded number the frame following fi must carry, the invariant container/flacn confirms packet boundaries with.
func (n Numbering) Next(fi FrameInfo) uint64 {
	if n.SampleCoded {
		return fi.Coded + uint64(fi.BlockSize)
	}
	return fi.Coded + 1
}

var blockSizes = [16]int{
	0, 192, 576, 1152, 2304, 4608, 0, 0,
	256, 512, 1024, 2048, 4096, 8192, 16384, 32768,
}

var sampleRates = [16]int{
	0, 88200, 176400, 192000, 8000, 16000, 22050, 24000,
	32000, 44100, 48000, 96000, 0, 0, 0, 0,
}

var sampleBits = [8]int{0, 8, 12, -1, 16, 20, 24, 32}

// SyncOK reports whether b begins with a frame sync sequence: the 15-bit code 0b111111111111100 followed by the blocking-strategy bit.
func SyncOK(b []byte) bool {
	return len(b) >= 2 && b[0] == 0xFF && b[1]&0xFE == 0xF8
}

// ParseFrameHeader parses and CRC-checks the frame header at the start of b.
func ParseFrameHeader(b []byte) (FrameInfo, error) {
	var fi FrameInfo
	if len(b) < 5 {
		return fi, malformed("frame header truncated")
	}
	if !SyncOK(b) {
		return fi, malformed("bad frame sync")
	}
	fi.Variable = b[1]&0x01 != 0

	bsCode := int(b[2]) >> 4
	rateCode := int(b[2]) & 0xF
	fi.assign = int(b[3]) >> 4
	bitsCode := (int(b[3]) >> 1) & 0x7

	if bsCode == 0 {
		return fi, malformed("reserved block size code")
	}
	if rateCode == 15 {
		return fi, malformed("invalid sample rate code")
	}
	if b[3]&0x01 != 0 {
		return fi, malformed("reserved frame header bit set")
	}
	switch {
	case fi.assign <= 7:
		fi.Channels = fi.assign + 1
	case fi.assign <= 10:
		fi.Channels = 2
	default:
		return fi, malformed("reserved channel assignment %d", fi.assign)
	}
	if fi.Bits = sampleBits[bitsCode]; fi.Bits < 0 {
		return fi, malformed("reserved sample size code")
	}

	pos := 4
	head := b[pos]
	pos++
	extra := 0
	switch {
	case head&0x80 == 0:
		fi.Coded = uint64(head)
	case head&0xC0 == 0x80, head == 0xFF:
		return fi, malformed("invalid coded number")
	default:
		for m := head; m&0x40 != 0; m <<= 1 {
			extra++
		}
		fi.Coded = uint64(head) & (0x3F >> extra)
	}
	if pos+extra > len(b) {
		return fi, malformed("frame header truncated")
	}
	for range extra {
		c := b[pos]
		if c&0xC0 != 0x80 {
			return fi, malformed("invalid coded number continuation")
		}
		fi.Coded = fi.Coded<<6 | uint64(c&0x3F)
		pos++
	}

	need := 0
	if bsCode == 6 {
		need++
	}
	if bsCode == 7 {
		need += 2
	}
	switch rateCode {
	case 12:
		need++
	case 13, 14:
		need += 2
	}
	if pos+need+1 > len(b) {
		return fi, malformed("frame header truncated")
	}
	switch bsCode {
	case 6:
		fi.BlockSize = int(b[pos]) + 1
		pos++
	case 7:
		fi.BlockSize = (int(b[pos])<<8 | int(b[pos+1])) + 1
		pos += 2
	default:
		fi.BlockSize = blockSizes[bsCode]
	}
	if fi.BlockSize > MaxBlockSize {
		return fi, malformed("block size %d exceeds %d", fi.BlockSize, MaxBlockSize)
	}
	switch rateCode {
	case 12:
		fi.Rate = int(b[pos]) * 1000
		pos++
	case 13:
		fi.Rate = int(b[pos])<<8 | int(b[pos+1])
		pos += 2
	case 14:
		fi.Rate = (int(b[pos])<<8 | int(b[pos+1])) * 10
		pos += 2
	default:
		fi.Rate = sampleRates[rateCode]
	}
	if fi.Rate == 0 && rateCode != 0 {
		return fi, malformed("frame sample rate 0")
	}

	if crc8(b[:pos]) != b[pos] {
		return fi, malformed("frame header CRC-8 mismatch")
	}
	fi.hdrLen = pos + 1
	return fi, nil
}
