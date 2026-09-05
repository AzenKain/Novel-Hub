package mp4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/codec/alac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

var _ container.Muxer = (*Muxer)(nil)

const trackID = 1

const defaultFragmentSeconds = 1

// MuxerOptions configures the fragmented writer.
type MuxerOptions struct {
	FragmentSamples int
	Tags            []container.Tag
	Chapters        []container.Chapter
	Art             *container.Picture
}

// Muxer writes one audio track as a progressive fragmented MP4 (fMP4): an ftyp+moov init header declaring an empty sample table plus a movie-extends (mvex) box, then a moof+mdat fragment per bounded run of samples.
type Muxer struct {
	w    io.Writer
	ws   io.WriteSeeker
	opts MuxerOptions

	rate     int
	fragTgt  int
	began    bool
	ended    bool
	seq      uint32
	baseTime int64

	elstDurOff int64
	delay      int64
	knownLen   int64

	smpbOff int64

	off int64

	fragSamples int
	fragData    []byte
	durs        []uint32
	sizes       []uint32
}

// NewMuxer returns a fragmented MP4 muxer writing to w.
func NewMuxer(w io.Writer, opts *MuxerOptions) *Muxer {
	m := &Muxer{w: w, seq: 1, elstDurOff: -1, smpbOff: -1}
	if ws, ok := w.(io.WriteSeeker); ok {
		m.ws = ws
	}
	if opts != nil {
		m.opts = *opts
		m.fragTgt = opts.FragmentSamples
	}
	return m
}

// NeedsSeek reports false: fragmented MP4 has a compliant streaming form.
func (m *Muxer) NeedsSeek() bool { return false }

// Begin validates the track and writes the ftyp and moov init header.
func (m *Muxer) Begin(tracks []container.Track) error {
	if m.began {
		return waxerr.New(waxerr.CodeInternal, "mp4: Begin called twice")
	}
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mp4: muxers are single-track, got %d", len(tracks)))
	}
	t := tracks[0]
	if err := t.Fmt.Valid(); err != nil {
		return err
	}

	var entry, edts []byte
	defaultDur := 0
	switch t.Codec {
	case codec.ALAC:
		if t.Delay != 0 || t.Padding != 0 {
			return waxerr.New(waxerr.CodeUnsupportedFormat, "mp4: ALAC signals no gapless trims (lossless streams have none)")
		}
		cfg, err := alac.ParseMagicCookie(t.CodecConfig)
		if err != nil {
			return err
		}
		if want := cfg.Format(); t.Fmt.Rate != want.Rate || t.Fmt.Channels != want.Channels ||
			t.Fmt.Type != want.Type || t.Fmt.BitDepth != want.BitDepth {
			return waxerr.New(waxerr.CodeUnsupportedFormat,
				fmt.Sprintf("mp4: track format %v does not match the ALAC cookie (%v)", t.Fmt, want))
		}
		entry = alacSampleEntry(t.Fmt, cfg.Cookie)
		defaultDur = alac.FrameSize
	case codec.AACLC:
		if t.Delay < 0 || t.Padding < 0 {
			return waxerr.New(waxerr.CodeInvalidRequest, "mp4: negative gapless trims")
		}
		var err error
		entry, err = aacSampleEntry(t)
		if err != nil {
			return err
		}
		cfg, _ := aac.ParseASC(t.CodecConfig)
		defaultDur = cfg.FrameLength
		if t.Delay > 0 || t.Samples > 0 {
			edts = elstBox(t.Delay, max(t.Samples, 0))
		}
		m.delay = t.Delay
		m.knownLen = t.Samples
	default:
		return waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf("mp4: cannot mux codec %q (alac, aac-lc)", t.Codec))
	}

	m.rate = t.Fmt.Rate
	if m.fragTgt <= 0 {
		m.fragTgt = defaultFragmentSeconds * m.rate
	}
	m.began = true

	if err := m.write(ftypBox()); err != nil {
		return err
	}
	var smpb []byte
	if t.Codec == codec.AACLC && m.ws != nil && t.Delay > 0 {
		smpb = freeformAtom("iTunSMPB", smpbPayload(t.Delay, max(t.Samples, 0)))
	}
	udta := udtaBox(m.opts.Tags, m.opts.Chapters, m.opts.Art, smpb)
	moov := moovBox(t.Fmt.Rate, entry, edts, defaultDur, udta)
	if edts != nil {
		if i := bytes.Index(moov, edts); i >= 0 {
			m.elstDurOff = m.off + int64(i) + elstDurOffset
		}
	}
	if smpb != nil {
		if i := bytes.Index(moov, smpb); i >= 0 {
			if j := bytes.Index(smpb, []byte(" 00000000 ")); j >= 0 {
				m.smpbOff = m.off + int64(i) + int64(j)
			}
		}
	}
	return m.write(moov)
}

// WritePacket appends one ALAC frame to the current fragment, flushing when the fragment reaches its target length.
func (m *Muxer) WritePacket(pkt container.Packet) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mp4: WritePacket outside Begin/End")
	}
	if pkt.Track != 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mp4: no track %d", pkt.Track))
	}
	if len(pkt.Data) == 0 || pkt.Dur <= 0 {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mp4: packet of %d bytes, %d samples", len(pkt.Data), pkt.Dur))
	}
	if len(pkt.Data) > maxSampleBytes || pkt.Dur > maxSampleDur {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mp4: packet too large (%d bytes, %d samples)", len(pkt.Data), pkt.Dur))
	}

	m.fragData = append(m.fragData, pkt.Data...)
	m.durs = append(m.durs, uint32(pkt.Dur))
	m.sizes = append(m.sizes, uint32(len(pkt.Data)))
	m.fragSamples += int(pkt.Dur)

	if m.fragSamples >= m.fragTgt || len(m.fragData) >= maxFragmentBytes {
		return m.flush()
	}
	return nil
}

// End flushes the final fragment.
func (m *Muxer) End(trailer codec.Trailer) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mp4: End outside Begin")
	}
	m.ended = true
	if trailer.Delay != m.delay {
		return waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("mp4: trailer delay %d disagrees with the init header's %d", trailer.Delay, m.delay))
	}
	if m.delay == 0 && trailer.Padding != 0 {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mp4: this track signals no gapless delay, so %d samples of end trim cannot be written",
				trailer.Padding))
	}
	if len(m.durs) > 0 {
		if err := m.flush(); err != nil {
			return err
		}
	}
	if m.ws != nil && m.elstDurOff >= 0 && trailer.Samples > 0 && trailer.Samples != m.knownLen {
		var dur [8]byte
		binary.BigEndian.PutUint64(dur[:], uint64(trailer.Samples))
		if err := m.patch(m.elstDurOff, dur[:], "elst"); err != nil {
			return err
		}
	}
	if m.ws != nil && m.smpbOff >= 0 {
		pad := fmt.Sprintf("%08X", uint32(trailer.Padding))
		length := fmt.Sprintf("%016X", uint64(max(trailer.Samples, 0)))
		if err := m.patch(m.smpbOff+smpbPaddingOff, []byte(pad), "iTunSMPB"); err != nil {
			return err
		}
		if err := m.patch(m.smpbOff+smpbLengthOff, []byte(length), "iTunSMPB"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Muxer) patch(off int64, b []byte, what string) error {
	if _, err := m.ws.Seek(off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: "+what+" seek", err)
	}
	if _, err := m.ws.Write(b); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: "+what+" patch", err)
	}
	if _, err := m.ws.Seek(m.off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: seeking to end", err)
	}
	return nil
}

func (m *Muxer) flush() error {
	frag := fragmentBoxes(m.seq, m.baseTime, m.durs, m.sizes, m.fragData)
	if err := m.write(frag); err != nil {
		return err
	}
	m.seq++
	m.baseTime += int64(m.fragSamples)
	m.fragSamples = 0
	m.fragData = m.fragData[:0]
	m.durs = m.durs[:0]
	m.sizes = m.sizes[:0]
	return nil
}

func (m *Muxer) write(b []byte) error {
	if _, err := m.w.Write(b); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: write", err)
	}
	m.off += int64(len(b))
	return nil
}

const (
	maxSampleBytes   = 1 << 24
	maxSampleDur     = 1 << 20
	maxFragmentBytes = 8 << 20
)

func ftypBox() []byte {
	return makeBox("ftyp",
		[]byte("M4A "), u32(0),
		[]byte("M4A "), []byte("mp42"), []byte("isom"), []byte("iso2"), []byte("iso5"))
}

func moovBox(rate int, entry, edts []byte, defaultDur int, udta []byte) []byte {
	mvhd := makeFullBox("mvhd", 0, 0,
		u32(0), u32(0),
		u32(uint32(rate)),
		u32(0),
		u32(0x00010000), u16(0x0100), u16(0), u32(0), u32(0),
		identityMatrix(),
		make([]byte, 24),
		u32(trackID+1))
	trak := trakBox(rate, entry, edts)
	mvex := makeBox("mvex", trexBox(defaultDur))
	if udta != nil {
		return makeBox("moov", mvhd, trak, mvex, udta)
	}
	return makeBox("moov", mvhd, trak, mvex)
}

func trakBox(rate int, entry, edts []byte) []byte {
	tkhd := makeFullBox("tkhd", 0, 0x000007,
		u32(0), u32(0),
		u32(trackID),
		u32(0),
		u32(0),
		u32(0), u32(0),
		u16(0), u16(0),
		u16(0x0100), u16(0),
		identityMatrix(),
		u32(0), u32(0))
	mdia := mdiaBox(rate, entry)
	if edts != nil {
		return makeBox("trak", tkhd, edts, mdia)
	}
	return makeBox("trak", tkhd, mdia)
}

func mdiaBox(rate int, entry []byte) []byte {
	mdhd := makeFullBox("mdhd", 0, 0,
		u32(0), u32(0),
		u32(uint32(rate)),
		u32(0),
		u16(0x55C4), u16(0))
	hdlr := makeFullBox("hdlr", 0, 0,
		u32(0),
		[]byte("soun"),
		u32(0), u32(0), u32(0),
		append([]byte("SoundHandler"), 0))
	minf := minfBox(entry)
	return makeBox("mdia", mdhd, hdlr, minf)
}

func minfBox(entry []byte) []byte {
	smhd := makeFullBox("smhd", 0, 0, u16(0), u16(0))
	dref := makeFullBox("dref", 0, 0, u32(1),
		makeFullBox("url ", 0, 0x000001))
	dinf := makeBox("dinf", dref)
	stbl := stblBox(entry)
	return makeBox("minf", smhd, dinf, stbl)
}

func stblBox(entry []byte) []byte {
	stsd := makeFullBox("stsd", 0, 0, u32(1), entry)
	stts := makeFullBox("stts", 0, 0, u32(0))
	stsc := makeFullBox("stsc", 0, 0, u32(0))
	stsz := makeFullBox("stsz", 0, 0, u32(0), u32(0))
	stco := makeFullBox("stco", 0, 0, u32(0))
	return makeBox("stbl", stsd, stts, stsc, stsz, stco)
}

func alacSampleEntry(f audio.Format, cookie []byte) []byte {
	inner := makeFullBox("alac", 0, 0, cookie)
	sampleRate := uint32(f.Rate)
	if sampleRate > 0xFFFF {
		sampleRate = 0xFFFF
	}
	return makeBox("alac",
		make([]byte, 6), u16(1),
		u16(0), u16(0), u32(0),
		u16(uint16(f.Channels)), u16(16),
		u16(0), u16(0),
		u32(sampleRate<<16),
		inner)
}

func trexBox(defaultDur int) []byte {
	return makeFullBox("trex", 0, 0,
		u32(trackID),
		u32(1),
		u32(uint32(defaultDur)),
		u32(0),
		u32(syncSampleFlags))
}

const syncSampleFlags = 0x02000000

func fragmentBoxes(seq uint32, baseTime int64, durs, sizes []uint32, data []byte) []byte {
	n := len(durs)

	mfhd := makeFullBox("mfhd", 0, 0, u32(seq))
	tfhd := makeFullBox("tfhd", 0, 0x020000, u32(trackID))
	tfdt := makeFullBox("tfdt", 1, 0, u64(uint64(baseTime)))

	body := make([]byte, 0, 12+8*n)
	body = append(body, 0)
	body = append(body, 0x00, 0x03, 0x01)
	body = binary.BigEndian.AppendUint32(body, uint32(n))
	dataOffPos := len(body)
	body = binary.BigEndian.AppendUint32(body, 0)
	for i := 0; i < n; i++ {
		body = binary.BigEndian.AppendUint32(body, durs[i])
		body = binary.BigEndian.AppendUint32(body, sizes[i])
	}
	trun := makeBox("trun", body)

	traf := makeBox("traf", tfhd, tfdt, trun)
	moof := makeBox("moof", mfhd, traf)

	dataOffset := uint32(len(moof) + 8)
	patchAt := len(moof) - len(trun) + 8 + dataOffPos
	binary.BigEndian.PutUint32(moof[patchAt:], dataOffset)

	mdat := makeBox("mdat", data)
	return append(moof, mdat...)
}

func identityMatrix() []byte {
	return bytes.Join([][]byte{
		u32(0x00010000), u32(0), u32(0),
		u32(0), u32(0x00010000), u32(0),
		u32(0), u32(0), u32(0x40000000),
	}, nil)
}
