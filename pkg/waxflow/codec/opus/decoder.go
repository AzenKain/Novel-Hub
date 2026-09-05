package opus

import (
	"math"
	"math/bits"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
)

// maxFrameSamples is the largest single Opus frame at 48 kHz (SILK 60 ms).
const Version = "opus-dec-2"

const maxFrameSamples = 2880

const (
	celtF5   = 240
	celtF2_5 = 120
)

// Decoder decodes an Opus stream to 48 kHz planar float32 (codec.Decoder).
type Decoder struct {
	cfg            Config
	fmt            audio.Format
	celt           *celtDecoder
	silk           *silkDecoder
	prevMode       int
	prevRedundancy bool
	gainMult       float32

	silkBuf [2][]int16
	redCh   [][]float32
	redBuf  [2][]float32
	out     *audio.Buffer
	outCh   [][]float32
}

// NewDecoder returns a decoder for a parsed OpusHead Config.
func NewDecoder(cfg Config, f audio.Format) (*Decoder, error) {
	if err := f.Valid(); err != nil {
		return nil, err
	}
	if f.Type != audio.Float || f.Channels != cfg.Channels || f.Rate != SampleRate {
		return nil, malformed("track format %v does not match Opus %dch", f, cfg.Channels)
	}
	// One elementary Opus stream carries at most two channels; more requires multistream de-framing (RFC 7845 family 1 with several streams), which this decoder does not implement.
	if cfg.Channels > 2 {
		return nil, malformed("%d-channel Opus needs multistream decoding (mono and stereo only)", cfg.Channels)
	}
	d := &Decoder{
		cfg:      cfg,
		fmt:      f,
		celt:     newCELTDecoder(cfg.Channels),
		silk:     newSILKDecoder(),
		prevMode: -1,
		gainMult: float32(math.Exp2(6.48814081e-4 * float64(cfg.Gain))),
	}
	for c := 0; c < cfg.Channels; c++ {
		d.silkBuf[c] = make([]int16, maxFrameSamples)
		d.redBuf[c] = make([]float32, celtF5)
	}
	d.out = audio.Get(f, maxFrameSamples)
	d.outCh = make([][]float32, cfg.Channels)
	d.redCh = make([][]float32, cfg.Channels)
	return d, nil
}

// Decode decodes one Opus packet, emitting one buffer per contained frame.
func (d *Decoder) Decode(pkt []byte, emit func(*audio.Buffer) error) error {
	frames, err := splitPacket(pkt)
	if err != nil {
		return err
	}
	for i := range frames {
		fr := frames[i]
		n := fr.cfg.frameSize
		d.out.N = n
		for c := 0; c < d.cfg.Channels; c++ {
			d.outCh[c] = d.out.ChanF(c)
		}
		if err := d.decodeFrame(fr, d.outCh); err != nil {
			return err
		}
		if d.gainMult != 1 {
			for c := 0; c < d.cfg.Channels; c++ {
				ch := d.outCh[c]
				for j := 0; j < n; j++ {
					ch[j] *= d.gainMult
				}
			}
		}
		if err := emit(d.out); err != nil {
			return err
		}
	}
	return nil
}

func endBandFor(bw int) int {
	switch bw {
	case bandNB:
		return 13
	case bandMB, bandWB:
		return 17
	case bandSWB:
		return 19
	default:
		return 21
	}
}

// decodeFrame decodes one Opus frame into out[ch][0:frameSize] at 48 kHz, mirroring libopus opus_decode_frame (opus_decoder.c).
func (d *Decoder) decodeFrame(fr frame, out [][]float32) error {
	mode := fr.cfg.mode
	bw := fr.cfg.bandwidth
	audiosize := fr.cfg.frameSize
	end := endBandFor(bw)
	C := 1
	if fr.stereo {
		C = 2
	}
	CC := d.cfg.Channels

	for c := 0; c < CC; c++ {
		clear(out[c][:audiosize])
	}

	dec := newRangeDecoder(fr.data)

	if mode != modeCELT {
		if d.prevMode == modeCELT {
			d.silk.reset()
		}
		d.silk.channel[0].nFramesDecoded = 0
		d.silk.channel[1].nFramesDecoded = 0
		internalRate := 16000
		if mode == modeSILK {
			switch bw {
			case bandNB:
				internalRate = 8000
			case bandMB:
				internalRate = 12000
			}
		}
		payloadMs := max(10, 1000*audiosize/SampleRate)
		ctrl := silkControl{payloadMs, internalRate, C, CC, SampleRate}
		decoded := 0
		for decoded < audiosize {
			subOut := make([][]int16, CC)
			for c := 0; c < CC; c++ {
				subOut[c] = d.silkBuf[c][decoded:]
			}
			n := d.silk.decode(dec, ctrl, subOut)
			if n <= 0 {
				return malformed("silk produced no samples")
			}
			decoded += n
		}
	}

	celtLen := len(fr.data)
	redundancy := false
	celtToSilk := false
	redundancyBytes := 0
	startBand := 0
	if mode != modeCELT {
		extra := 0
		if mode == modeHybrid {
			extra = 20
		}
		if dec.tell()+17+extra <= 8*celtLen {
			if mode == modeHybrid {
				redundancy = dec.decodeBitLogp(12) != 0
			} else {
				redundancy = true
			}
			if redundancy {
				celtToSilk = dec.decodeBitLogp(1) != 0
				if mode == modeHybrid {
					redundancyBytes = int(dec.decodeUint(256)) + 2
				} else {
					redundancyBytes = celtLen - (dec.tell()+7)/8
				}
				celtLen -= redundancyBytes
				if celtLen*8 < dec.tell() || celtLen < 0 || redundancyBytes < 0 ||
					celtLen+redundancyBytes > len(fr.data) {
					celtLen = 0
					redundancyBytes = 0
					redundancy = false
				} else {
					dec.storage -= redundancyBytes
				}
			}
		}
		startBand = 17
	}
	redData := fr.data[celtLen : celtLen+redundancyBytes]

	if redundancy && celtToSilk {
		d.redViews()
		if err := d.celt.celtDecode(redData, 1, C, 0, end, d.redCh); err != nil {
			return err
		}
	}

	if mode != modeSILK {
		if mode != d.prevMode && d.prevMode != -1 && !d.prevRedundancy {
			d.celt.Reset()
		}
		LM := bits.Len(uint(audiosize/celtShortMDCTSize)) - 1
		if err := d.celt.celtDecodeInner(dec, fr.data[:celtLen], LM, C, startBand, end, out); err != nil {
			return err
		}
	} else if d.prevMode == modeHybrid && !(redundancy && celtToSilk && d.prevRedundancy) {
		d.celt.celtDecode(silenceCELT[:], 0, C, 0, end, out)
	}

	if mode != modeCELT {
		for c := 0; c < CC; c++ {
			oc := out[c]
			sc := d.silkBuf[c]
			for i := 0; i < audiosize; i++ {
				oc[i] += float32(sc[i]) * (1.0 / 32768.0)
			}
		}
	}

	if redundancy && !celtToSilk {
		d.celt.Reset()
		d.redViews()
		if err := d.celt.celtDecode(redData, 1, C, 0, end, d.redCh); err != nil {
			return err
		}
		for c := 0; c < CC; c++ {
			smoothFade(out[c][audiosize-celtF2_5:], d.redCh[c][celtF2_5:], out[c][audiosize-celtF2_5:], d.celt.window)
		}
	}

	if redundancy && celtToSilk && (d.prevMode != modeSILK || d.prevRedundancy) {
		for c := 0; c < CC; c++ {
			copy(out[c][:celtF2_5], d.redCh[c][:celtF2_5])
			smoothFade(d.redCh[c][celtF2_5:], out[c][celtF2_5:], out[c][celtF2_5:], d.celt.window)
		}
	}

	d.prevMode = mode
	d.prevRedundancy = redundancy && !celtToSilk
	return nil
}

var silenceCELT = [2]byte{0xFF, 0xFF}

func (d *Decoder) redViews() {
	for c := 0; c < d.cfg.Channels; c++ {
		d.redCh[c] = d.redBuf[c]
	}
}

func smoothFade(in1, in2, out []float32, window []float64) {
	for i := 0; i < celtF2_5; i++ {
		w := float32(window[i] * window[i])
		out[i] = w*in2[i] + (1-w)*in1[i]
	}
}

// Drain flushes decoder latency.
func (d *Decoder) Drain(func(*audio.Buffer) error) error { return nil }

// Reset discards inter-frame state after a seek so the next packet primes.
func (d *Decoder) Reset() {
	d.celt.Reset()
	d.silk.reset()
	d.prevMode = -1
}

// Release returns the pooled output buffer.
func (d *Decoder) Release() {
	if d.out != nil {
		audio.Put(d.out)
		d.out = nil
	}
}

func (d *silkDecoder) reset() {
	d.channel[0].reset()
	d.channel[1].reset()
	d.sStereo = stereoState{}
	d.prevDecodeOnlyMiddle = 0
	d.nChannelsAPI = 0
	d.nChannelsInternal = 0
}

var _ codec.Decoder = (*Decoder)(nil)
