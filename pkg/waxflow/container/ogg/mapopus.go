package ogg

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
)

// opusMapping decodes the Ogg-Opus mapping (RFC 7845).
type opusMapping struct {
	channels int
	preSkip  int64
	family   int
	head     []byte
}

func (m *opusMapping) codecID() codec.ID { return codec.Opus }

// parseID parses the OpusHead identification header (RFC 7845 section 5.1).
func (m *opusMapping) parseID(pkt []byte) (int, error) {
	if len(pkt) < 19 || string(pkt[:8]) != "OpusHead" {
		return 0, malformed("OpusHead identification header malformed")
	}
	if version := pkt[8]; version&0xF0 != 0 {
		return 0, malformed("unsupported OpusHead version %d", version)
	}
	m.channels = int(pkt[9])
	if m.channels < 1 || m.channels > audio.MaxChannels {
		return 0, malformed("OpusHead channel count %d", m.channels)
	}
	m.preSkip = int64(binary.LittleEndian.Uint16(pkt[10:]))
	m.family = int(pkt[18])
	switch m.family {
	case 0:
		if m.channels > 2 {
			return 0, malformed("OpusHead family 0 with %d channels", m.channels)
		}
	case 1:
		// Family 1 carries a channel-mapping table: stream count, coupled count, then one index per channel (RFC 7845 section 5.1.1).
		if len(pkt) < 21+m.channels {
			return 0, malformed("OpusHead family 1 truncated (%d bytes, %d channels)", len(pkt), m.channels)
		}
		streams, coupled := int(pkt[19]), int(pkt[20])
		if streams < 1 || coupled > streams || streams+coupled > 255 {
			return 0, malformed("OpusHead family 1 stream/coupled counts %d/%d", streams, coupled)
		}
		for i := 0; i < m.channels; i++ {
			if v := pkt[21+i]; v != 255 && int(v) >= streams+coupled {
				return 0, malformed("OpusHead family 1 channel maps to stream %d of %d", v, streams+coupled)
			}
		}
		if streams != 1 || coupled != m.channels-1 {
			return 0, malformed("OpusHead family 1 with %d streams (%d coupled) for %d channels is multistream; mono and stereo single-stream only", streams, coupled, m.channels)
		}
		for i := 0; i < m.channels; i++ {
			if int(pkt[21+i]) != i {
				return 0, malformed("OpusHead family 1 channel table %v is not the identity mapping", pkt[21:21+m.channels])
			}
		}
	default:
		return 0, malformed("OpusHead channel mapping family %d unsupported", m.family)
	}
	m.head = append([]byte(nil), pkt...)
	return 1, nil
}

func (m *opusMapping) parseHeader(pkt []byte) error {
	if len(pkt) < 8 || string(pkt[:8]) != "OpusTags" {
		return malformed("second Opus header is not OpusTags")
	}
	return nil
}

func (m *opusMapping) isAudio(pkt []byte) bool {
	if len(pkt) >= 8 && (string(pkt[:8]) == "OpusHead" || string(pkt[:8]) == "OpusTags") {
		return false
	}
	return len(pkt) > 0
}

func (m *opusMapping) finalizeTrack(lastGranule func() int64) (container.Track, error) {
	f := audio.Format{
		Rate:     48000,
		Channels: m.channels,
		Layout:   audio.DefaultLayout(m.channels),
		Type:     audio.Float,
		BitDepth: 32,
	}
	if err := f.Valid(); err != nil {
		return container.Track{}, err
	}
	samples := int64(-1)
	if lg := lastGranule(); lg >= 0 {
		samples = lg - m.preSkip
		if samples < 0 {
			samples = 0
		}
	}
	return container.Track{
		Codec:        codec.Opus,
		CodecConfig:  m.head,
		Fmt:          f,
		Samples:      samples,
		SamplesExact: samples >= 0,
		Delay:        m.preSkip,
		Default:      true,
	}, nil
}

// opusFrameSamples is the frame length in 48 kHz samples for each TOC config (RFC 6716 Table 2): SILK 10/20/40/60 ms, Hybrid 10/20 ms, CELT 2.5/5/10/20 ms.
var opusFrameSamples = [32]int64{
	480, 960, 1920, 2880,
	480, 960, 1920, 2880,
	480, 960, 1920, 2880,
	480, 960,
	480, 960,
	120, 240, 480, 960,
	120, 240, 480, 960,
	120, 240, 480, 960,
	120, 240, 480, 960,
}

// packetTiming derives an Opus packet's duration from its TOC byte and frame count (RFC 6716 section 3.1).
func (m *opusMapping) packetTiming(pkt []byte, running int64) (pts, dur int64, sync, ok bool) {
	if len(pkt) == 0 {
		return 0, 0, false, false
	}
	toc := pkt[0]
	config := toc >> 3
	frameSize := opusFrameSamples[config]
	var frames int64
	switch toc & 0x3 {
	case 0:
		frames = 1
	case 1, 2:
		frames = 2
	case 3:
		if len(pkt) < 2 {
			return 0, 0, false, false
		}
		frames = int64(pkt[1] & 0x3F)
	}
	total := frameSize * frames
	// RFC 6716: total packet duration must not exceed 120 ms (5760 samples).
	if frames <= 0 || total <= 0 || total > 5760 {
		return 0, 0, false, false
	}
	return running, total, true, true
}

func (m *opusMapping) selfTiming() bool { return false }

// preroll is 80 ms at 48 kHz (RFC 7845 section 4.2): enough for the decoder's SILK/CELT state to reconverge after a seek.
func (m *opusMapping) preroll() int64 { return 3840 }

func (m *opusMapping) resetTiming() {}
