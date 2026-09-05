package aac

import (
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
)

var (
	_ codec.Decoder  = (*Decoder)(nil)
	_ codec.Releaser = (*Decoder)(nil)
)

const (
	elSCE = 0
	elCPE = 1
	elCCE = 2
	elLFE = 3
	elDSE = 4
	elPCE = 5
	elFIL = 6
	elEND = 7
)

const (
	maxWindowGroups = 8
	maxSFBCount     = 64
	sfOffset        = 100
)

type icsInfo struct {
	windowSequence  int
	windowShape     int
	maxSfb          int
	numWindows      int
	numWindowGroups int
	windowGroupLen  [maxWindowGroups]int
	swb             []uint16
	numSwb          int
}

type channelData struct {
	info       icsInfo
	globalGain int
	sfbCb      [maxWindowGroups][maxSFBCount]uint8
	sf         [maxWindowGroups][maxSFBCount]int
	spec       [1024]float64
	tns        tnsInfo
	hasTNS     bool
	hasPulse   bool
	pulse      pulseInfo
	pnsSeed    uint32
}

// Decoder decodes AAC-LC access units into planar float buffers.
type Decoder struct {
	cfg      Config
	fmt      audio.Format
	rateIdx  int
	frameLen int
	slots    []int
	elems    []int

	buf      *audio.Buffer
	ch       [audio.MaxChannels]channelData
	overlap  [audio.MaxChannels][1024]float64
	prevWin  [audio.MaxChannels]int
	pnsState uint32
}

// NewDecoder returns a Decoder for a stream.
func NewDecoder(cfg Config, f audio.Format) (*Decoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if cfg.ObjectType != aotAACLC {
		return nil, malformed("object type %d is not AAC-LC", cfg.ObjectType)
	}
	if cfg.FrameLength != 1024 {
		return nil, malformed("frame length %d unsupported (only 1024)", cfg.FrameLength)
	}
	rateIdx := samplingIndex(cfg.SampleRate)
	if rateIdx < 0 || rateIdx >= len(swbOffsetLong) {
		return nil, malformed("sample rate %d has no scalefactor-band table", cfg.SampleRate)
	}
	slots := waveSlots(cfg.ChannelConfig, f.Channels)
	if len(slots) != f.Channels {
		return nil, malformed("channel configuration %d with %d channels has no known element order",
			cfg.ChannelConfig, f.Channels)
	}
	elems := channelElements(cfg.ChannelConfig)
	if elems != nil && len(elems) != len(slots) {
		return nil, malformed("channel configuration %d maps %d elements onto %d channels",
			cfg.ChannelConfig, len(elems), len(slots))
	}
	d := &Decoder{cfg: cfg, fmt: f, rateIdx: rateIdx,
		frameLen: int(cfg.FrameLength), slots: slots, elems: elems, pnsState: 0x1f2e3d4c}
	return d, nil
}

func (d *Decoder) checkElement(pos int, tag uint32) error {
	if d.elems == nil {
		if tag == elLFE {
			return malformed("channel configuration %d has no LFE channel", d.cfg.ChannelConfig)
		}
		return nil
	}
	if d.elems[pos] == int(tag) {
		return nil
	}
	return malformed("channel configuration %d codes %s at channel %d, found %s",
		d.cfg.ChannelConfig, elementName(d.elems[pos]), pos, elementName(int(tag)))
}

// Decode decodes one access unit and emits one 1024-frame buffer.
func (d *Decoder) Decode(pkt []byte, emit func(*audio.Buffer) error) error {
	if d.buf == nil || d.buf.Cap() < d.frameLen || d.buf.Fmt != d.fmt {
		audio.Put(d.buf)
		d.buf = audio.Get(d.fmt, d.frameLen)
	}
	d.buf.N = d.frameLen
	for c := 0; c < len(d.slots); c++ {
		clear(d.buf.ChanF(c)[:d.frameLen])
	}

	r := newBitReader(pkt)
	elem := 0
	for {
		if r.left() < 3 {
			break
		}
		tag := r.read(3)
		switch tag {
		case elSCE, elLFE:
			r.read(4)
			if elem >= len(d.slots) {
				return malformed("more channels in bitstream than configured")
			}
			if err := d.checkElement(elem, tag); err != nil {
				return err
			}
			cd := &d.ch[0]
			if err := d.decodeChannelData(r, cd, false); err != nil {
				return err
			}
			d.dequant(cd)
			d.applyPNS(cd)
			d.finishChannel(cd, d.slots[elem])
			elem++
		case elCPE:
			r.read(4)
			if elem+2 > len(d.slots) {
				return malformed("channel pair exceeds configured channels")
			}
			if err := d.checkElement(elem, tag); err != nil {
				return err
			}
			if err := d.decodePair(r, d.slots[elem], d.slots[elem+1]); err != nil {
				return err
			}
			elem += 2
		case elDSE:
			skipDSE(r)
		case elPCE:
			skipPCE(r)
		case elFIL:
			skipFIL(r)
		case elEND:
			r.byteAlign()
			goto done
		default:
			return malformed("unsupported element type %d", tag)
		}
		if r.overrun() {
			return malformed("access unit overruns packet")
		}
		if elem >= len(d.slots) {
			break
		}
	}
done:
	return emit(d.buf)
}

func (d *Decoder) decodePair(r *bitReader, leftCh, rightCh int) error {
	common := r.bit() != 0
	var shared icsInfo
	msMask := 0
	var msUsed [maxWindowGroups][maxSFBCount]bool
	if common {
		if !d.parseICSInfo(r, &shared) {
			return malformed("bad shared ics_info")
		}
		msMask = int(r.read(2))
		if msMask == 1 {
			for g := 0; g < shared.numWindowGroups; g++ {
				for sfb := 0; sfb < shared.maxSfb; sfb++ {
					msUsed[g][sfb] = r.bit() != 0
				}
			}
		}
	}
	left, right := &d.ch[0], &d.ch[1]
	if common {
		left.info = shared
		right.info = shared
	}
	if err := d.decodeChannelData(r, left, common); err != nil {
		return err
	}
	if err := d.decodeChannelData(r, right, common); err != nil {
		return err
	}
	d.dequant(left)
	d.dequant(right)
	d.applyPNS(left)
	d.applyPNS(right)
	if common && msMask != 0 {
		applyMS(left, right, msMask, &msUsed)
	}
	applyIntensity(left, right, msMask, &msUsed)
	d.finishChannel(left, leftCh)
	d.finishChannel(right, rightCh)
	return nil
}

// Drain is a no-op: each access unit emits its full 1024-sample frame, and the trailing filterbank overlap belongs to no further frame.
func (d *Decoder) Drain(func(*audio.Buffer) error) error { return nil }

// Reset clears the filterbank overlap and window history after a seek.
func (d *Decoder) Reset() {
	for c := range d.overlap {
		d.overlap[c] = [1024]float64{}
		d.prevWin[c] = shapeSine
	}
}

// Release returns the output buffer to the pool (codec.Releaser).
func (d *Decoder) Release() {
	audio.Put(d.buf)
	d.buf = nil
}
