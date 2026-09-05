package mp4

import (
	"encoding/binary"
	"fmt"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/alac"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

// SegmenterVersion identifies the segment and init-header box layout for the ADR-0004 cache key: cached segments regenerate when this bumps, so a box-layout fix can never serve stale segments next to fresh ones.
const SegmenterVersion = "mp4-seg-3"

const maxSegmentPayload = 1 << 30

// Segment is one emitted media segment: an styp plus one moof+mdat pair, self-contained and independently decodable.
type Segment struct {
	Index   int64
	Data    []byte
	Samples int64
}

// SegmenterOptions configures a Segmenter.
type SegmenterOptions struct {
	SegmentSamples int
	StartSegment   int64
}

// Segmenter packs codec packets into numbered CMAF media segments for HLS: each segment is styp plus exactly one moof+mdat pair whose tfdt carries the decode time in track samples (the media timescale is the sample rate).
type Segmenter struct {
	segTgt int
	index  int64
	ended  bool

	data        []byte
	durs, sizes []uint32
	samples     int64
}

// NewSegmenter validates the track and options and returns a Segmenter.
func NewSegmenter(t container.Track, opts *SegmenterOptions) (*Segmenter, error) {
	if opts == nil || opts.SegmentSamples <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "mp4: segmenter needs a positive SegmentSamples")
	}
	if opts.StartSegment < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "mp4: negative StartSegment")
	}
	if _, err := sampleEntryFor(t); err != nil {
		return nil, err
	}
	return &Segmenter{
		segTgt: opts.SegmentSamples,
		index:  opts.StartSegment,
	}, nil
}

// WritePacket appends one packet to the current segment, emitting the segment once it reaches its target length.
func (s *Segmenter) WritePacket(pkt codec.Packet, emit func(Segment) error) error {
	if s.ended {
		return waxerr.New(waxerr.CodeInternal, "mp4: WritePacket after End")
	}
	if len(pkt.Data) == 0 || pkt.Dur <= 0 {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mp4: packet of %d bytes, %d samples", len(pkt.Data), pkt.Dur))
	}
	if len(pkt.Data) > maxSampleBytes || pkt.Dur > maxSampleDur {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mp4: packet too large (%d bytes, %d samples)", len(pkt.Data), pkt.Dur))
	}
	if s.samples+pkt.Dur > int64(s.segTgt) {
		return waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("mp4: %d-sample packet straddles the segment boundary (%d of %d samples filled)",
				pkt.Dur, s.samples, s.segTgt))
	}
	if len(s.data)+len(pkt.Data) > maxSegmentPayload {
		return waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("mp4: segment payload past %d bytes; no legal configuration reaches this", maxSegmentPayload))
	}

	s.data = append(s.data, pkt.Data...)
	s.durs = append(s.durs, uint32(pkt.Dur))
	s.sizes = append(s.sizes, uint32(len(pkt.Data)))
	s.samples += pkt.Dur

	if s.samples >= int64(s.segTgt) {
		return s.emitSegment(emit)
	}
	return nil
}

// End flushes the final, possibly short, segment.
func (s *Segmenter) End(emit func(Segment) error) error {
	if s.ended {
		return waxerr.New(waxerr.CodeInternal, "mp4: End called twice")
	}
	s.ended = true
	if len(s.durs) > 0 {
		return s.emitSegment(emit)
	}
	return nil
}

func (s *Segmenter) emitSegment(emit func(Segment) error) error {
	frag := fragmentBoxes(uint32(s.index)+1, s.index*int64(s.segTgt), s.durs, s.sizes, s.data)
	data := make([]byte, 0, len(stypBox)+len(frag))
	data = append(data, stypBox...)
	data = append(data, frag...)
	seg := Segment{Index: s.index, Data: data, Samples: s.samples}
	s.index++
	s.samples = 0
	s.data = s.data[:0]
	s.durs = s.durs[:0]
	s.sizes = s.sizes[:0]
	return emit(seg)
}

// InitSegment builds the CMAF init header for the track: ftyp plus a moov whose track carries the codec's sample entry, an empty sample table, the movie-extends defaults, and, when the track declares an encoder delay or a known length, an edit list mapping the decode timeline onto the presentation one (the fMP4 gapless convention: the delay is known up front and rides in the init header; end padding is trimmed by the same edit when the length is known).
func InitSegment(t container.Track) ([]byte, error) {
	entry, err := sampleEntryFor(t)
	if err != nil {
		return nil, err
	}
	var edts []byte
	if t.Delay > 0 || t.Samples > 0 {
		edts = elstBox(t.Delay, max(t.Samples, 0))
	}
	init := append([]byte{}, initFtypBox...)
	return append(init, moovBox(t.Fmt.Rate, entry, edts, 0, nil)...), nil
}

var (
	initFtypBox = makeBox("ftyp",
		[]byte("iso6"), u32(0),
		[]byte("iso6"), []byte("iso5"), []byte("cmfc"), []byte("mp41"))
	stypBox = makeBox("styp",
		[]byte("msdh"), u32(0),
		[]byte("msdh"))
)

func elstBox(mediaTime, duration int64) []byte {
	elst := makeFullBox("elst", 1, 0,
		u32(1),
		u64(uint64(duration)),
		u64(uint64(mediaTime)),
		u16(1), u16(0))
	return makeBox("edts", elst)
}

const elstDurOffset = 8 + 8 + 4 + 4

func sampleEntryFor(t container.Track) ([]byte, error) {
	if err := t.Fmt.Valid(); err != nil {
		return nil, err
	}
	switch t.Codec {
	case codec.Opus:
		return opusSampleEntry(t)
	case codec.FLAC:
		return flacSampleEntry(t)
	case codec.AACLC:
		return aacSampleEntry(t)
	case codec.ALAC:
		cfg, err := alac.ParseMagicCookie(t.CodecConfig)
		if err != nil {
			return nil, err
		}
		if want := cfg.Format(); t.Fmt.Rate != want.Rate || t.Fmt.Channels != want.Channels ||
			t.Fmt.Type != want.Type || t.Fmt.BitDepth != want.BitDepth {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
				fmt.Sprintf("mp4: track format %v does not match the ALAC cookie (%v)", t.Fmt, want))
		}
		if t.Delay != 0 {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mp4: ALAC signals no encoder delay")
		}
		return alacSampleEntry(t.Fmt, cfg.Cookie), nil
	}
	return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
		fmt.Sprintf("mp4: cannot segment codec %q (opus, flac, alac, aac-lc)", t.Codec))
}

func opusSampleEntry(t container.Track) ([]byte, error) {
	head := t.CodecConfig
	if len(head) != 19 || string(head[:8]) != "OpusHead" || head[8] != 1 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mp4: malformed OpusHead codec config")
	}
	channels := int(head[9])
	preSkip := binary.LittleEndian.Uint16(head[10:])
	inputRate := binary.LittleEndian.Uint32(head[12:])
	outputGain := binary.LittleEndian.Uint16(head[16:])
	if family := head[18]; family != 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: opus channel mapping family %d (only family 0)", family))
	}
	if channels != t.Fmt.Channels {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: track has %d channels, OpusHead %d", t.Fmt.Channels, channels))
	}
	if int64(preSkip) != t.Delay {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: track delay %d disagrees with OpusHead pre-skip %d", t.Delay, preSkip))
	}
	dops := makeBox("dOps",
		[]byte{0},
		[]byte{byte(channels)},
		u16(preSkip),
		u32(inputRate),
		u16(outputGain),
		[]byte{0})
	return audioSampleEntry("Opus", t.Fmt, dops), nil
}

func flacSampleEntry(t container.Track) ([]byte, error) {
	si, err := flac.ParseStreamInfo(t.CodecConfig)
	if err != nil {
		return nil, err
	}
	if want := si.PCMFormat(); t.Fmt.Rate != want.Rate || t.Fmt.Channels != want.Channels ||
		t.Fmt.Type != want.Type || t.Fmt.BitDepth != want.BitDepth {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: track format %v does not match STREAMINFO (%v)", t.Fmt, want))
	}
	if t.Delay != 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mp4: FLAC signals no encoder delay")
	}
	dfla := makeFullBox("dfLa", 0, 0,
		[]byte{0x80, 0, 0, flac.StreamInfoLen},
		t.CodecConfig)
	return audioSampleEntry("fLaC", t.Fmt, dfla), nil
}

func audioSampleEntry(typ string, f audio.Format, child []byte) []byte {
	sampleRate := uint32(f.Rate)
	if sampleRate > 0xFFFF {
		sampleRate = 0xFFFF
	}
	return makeBox(typ,
		make([]byte, 6), u16(1),
		u16(0), u16(0), u32(0),
		u16(uint16(f.Channels)), u16(16),
		u16(0), u16(0),
		u32(sampleRate<<16),
		child)
}
