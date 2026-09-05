// Package aac implements an AAC-LC decoder (ISO/IEC 14496-3), written from the specification and Bosi/Goldberg (clean-room: AAC reference decoders were behavioral references only, never opened while implementing).
package aac

import (
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/waxerr"
)

// Version is the decoder's cache-key version constant (ADR-0004): bump on any change that alters decoded samples.
const Version = "aac-dec-2"

const (
	aotAACMain = 1
	aotAACLC   = 2
	aotAACSSR  = 3
	aotAACLTP  = 4
	aotSBR     = 5
	aotPS      = 29
)

var sampleRates = [...]int{
	96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050,
	16000, 12000, 11025, 8000, 7350, 0, 0, 0,
}

var channelConfigs = [...]int{0, 1, 2, 3, 4, 5, 6, 8}

// Config is a parsed AudioSpecificConfig.
type Config struct {
	ObjectType    int
	SampleRate    int
	Channels      int
	ChannelConfig int
	FrameLength   int
	ASC           []byte
	SBR           bool
	PS            bool
	ExtensionRate int
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "aac: "+fmt.Sprintf(format, args...))
}

// ParseASC parses an AudioSpecificConfig, resolving the AAC-LC base rate, channel count, and frame length.
func ParseASC(b []byte) (Config, error) {
	if len(b) < 2 {
		return Config{}, malformed("AudioSpecificConfig of %d bytes", len(b))
	}
	r := ascReader{data: b}
	aot := r.objectType()
	rate, err := r.samplingRate()
	if err != nil {
		return Config{}, err
	}
	chanConfig := int(r.read(4))

	var sbr, ps bool
	extRate := 0
	if aot == aotSBR || aot == aotPS {
		sbr, ps = true, aot == aotPS
		er, err := r.samplingRate()
		if err != nil {
			return Config{}, err
		}
		extRate = er
		aot = r.objectType()
	}
	if aot != aotAACLC {
		return Config{}, malformed("audio object type %d is not AAC-LC", aot)
	}

	frameLen := 1024
	if r.read(1) != 0 {
		frameLen = 960
	}

	channels := 0
	if chanConfig >= 1 && chanConfig < len(channelConfigs) {
		channels = channelConfigs[chanConfig]
	}
	if chanConfig >= len(channelConfigs) {
		return Config{}, malformed("channel configuration %d is not supported", chanConfig)
	}
	if chanConfig == 7 {
		return Config{}, malformed("channel configuration 7 is not supported: the specification and the common encoder convention disagree on its channel order")
	}
	if rate <= 0 {
		return Config{}, malformed("sampling frequency index reserved")
	}
	return Config{
		ObjectType:    aot,
		SampleRate:    rate,
		Channels:      channels,
		ChannelConfig: chanConfig,
		FrameLength:   frameLen,
		ASC:           append([]byte(nil), b...),
		SBR:           sbr,
		PS:            ps,
		ExtensionRate: extRate,
	}, nil
}

// SBRWarning returns the note a demuxer should record for an explicitly signalled SBR/PS config, or "" when there is nothing to warn about.
func (c Config) SBRWarning() string {
	if !c.SBR {
		return ""
	}
	name := "SBR"
	if c.PS {
		name = "SBR/PS"
	}
	if c.ExtensionRate > 0 {
		return fmt.Sprintf("%s signalled at %d Hz: high band not synthesized, decoding the AAC-LC base layer at %d Hz",
			name, c.ExtensionRate, c.SampleRate)
	}
	return fmt.Sprintf("%s high band not synthesized; decoding the AAC-LC base layer at %d Hz", name, c.SampleRate)
}

var channelLayouts = [...]audio.ChannelMask{
	1: audio.FrontCenter,
	2: audio.FrontLeft | audio.FrontRight,
	3: audio.FrontLeft | audio.FrontRight | audio.FrontCenter,
	4: audio.FrontLeft | audio.FrontRight | audio.FrontCenter | audio.BackCenter,
	5: audio.FrontLeft | audio.FrontRight | audio.FrontCenter | audio.BackLeft | audio.BackRight,
	6: audio.FrontLeft | audio.FrontRight | audio.FrontCenter | audio.LowFrequency |
		audio.BackLeft | audio.BackRight,
}

func waveSlots(chanConfig, channels int) []int {
	switch chanConfig {
	case 0:
		if channels < 1 || channels > 2 {
			return nil
		}
		return identitySlots(channels)
	case 1:
		return []int{0}
	case 2:
		return []int{0, 1}
	case 3:
		return []int{2, 0, 1}
	case 4:
		return []int{2, 0, 1, 3}
	case 5:
		return []int{2, 0, 1, 3, 4}
	case 6:
		return []int{2, 0, 1, 4, 5, 3}
	}
	return nil
}

func identitySlots(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func channelElements(chanConfig int) []int {
	switch chanConfig {
	case 3:
		return []int{elSCE, elCPE, elCPE}
	case 4:
		return []int{elSCE, elCPE, elCPE, elSCE}
	case 5:
		return []int{elSCE, elCPE, elCPE, elCPE, elCPE}
	case 6:
		return []int{elSCE, elCPE, elCPE, elCPE, elCPE, elLFE}
	}
	return nil
}

func elementName(tag int) string {
	switch tag {
	case elSCE:
		return "a single channel element"
	case elCPE:
		return "a channel pair element"
	case elLFE:
		return "an LFE element"
	}
	return fmt.Sprintf("element type %d", tag)
}

// Format is the pipeline format the decoder emits: the base rate in the pipeline's 32-bit float domain, with the layout the channel configuration names.
func (c Config) Format() (audio.Format, error) {
	ch := c.Channels
	layout := audio.DefaultLayout(ch)
	switch {
	case c.ChannelConfig == 0:
		if ch < 1 {
			return audio.Format{}, malformed("channel configuration 0: the channel count is carried by an in-band program config element, which is not parsed")
		}
		if ch > 2 {
			return audio.Format{}, malformed("channel configuration 0 with %d channels: the element order is carried by an in-band program config element, which is not parsed", ch)
		}
	case c.ChannelConfig > 0 && c.ChannelConfig < len(channelLayouts) &&
		channelLayouts[c.ChannelConfig] != 0:
		layout = channelLayouts[c.ChannelConfig]
	default:
		return audio.Format{}, malformed("channel configuration %d is not supported", c.ChannelConfig)
	}
	return audio.Format{
		Rate:     c.SampleRate,
		Channels: ch,
		Layout:   layout,
		Type:     audio.Float,
		BitDepth: 32,
	}, nil
}

type ascReader struct {
	data []byte
	pos  int
}

func (r *ascReader) read(n uint) uint32 {
	var v uint32
	for i := uint(0); i < n; i++ {
		bit := uint32(0)
		if idx := r.pos >> 3; idx < len(r.data) {
			bit = uint32(r.data[idx]>>(7-uint(r.pos&7))) & 1
		}
		v = v<<1 | bit
		r.pos++
	}
	return v
}

func (r *ascReader) objectType() int {
	aot := int(r.read(5))
	if aot == 31 {
		aot = 32 + int(r.read(6))
	}
	return aot
}

func (r *ascReader) samplingRate() (int, error) {
	idx := r.read(4)
	if idx == 15 {
		return int(r.read(24)), nil
	}
	if idx >= uint32(len(sampleRates)) || sampleRates[idx] == 0 {
		return 0, malformed("reserved sampling frequency index %d", idx)
	}
	return sampleRates[idx], nil
}
