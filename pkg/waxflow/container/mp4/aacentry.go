package mp4

import (
	"fmt"

	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

func aacSampleEntry(t container.Track) ([]byte, error) {
	cfg, err := aac.ParseASC(t.CodecConfig)
	if err != nil {
		return nil, err
	}
	want, err := cfg.Format()
	if err != nil {
		return nil, err
	}
	if t.Fmt.Rate != want.Rate || t.Fmt.Channels != want.Channels ||
		t.Fmt.Type != want.Type || t.Fmt.BitDepth != want.BitDepth {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: track format %v does not match the AudioSpecificConfig (%v)", t.Fmt, want))
	}
	return audioSampleEntry("mp4a", t.Fmt, esdsBox(t.CodecConfig)), nil
}

func esdsBox(asc []byte) []byte {
	dsi := descriptor(tagDecoderSpecific, asc)
	decCfg := descriptor(tagDecoderConfig, concat(
		[]byte{0x40},
		[]byte{0x05<<2 | 1},
		[]byte{0, 0x18, 0},
		u32(0), u32(0),
		dsi))
	es := descriptor(tagES, concat(
		u16(1),
		[]byte{0},
		decCfg,
		descriptor(0x06, []byte{0x02})))
	return makeFullBox("esds", 0, 0, es)
}

func descriptor(tag byte, body []byte) []byte {
	if len(body) > 127 {
		panic("mp4: descriptor body over 127 bytes")
	}
	out := make([]byte, 0, 2+len(body))
	out = append(out, tag, byte(len(body)))
	return append(out, body...)
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
