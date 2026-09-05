package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

var _ container.Muxer = (*ProgressiveMuxer)(nil)

// ProgressiveMuxer writes one audio track as a flat (non-fragmented) MP4: an ftyp, an mdat holding every sample, then a moov whose stbl carries the full sample tables (stsd/stts/stsc/stsz/stco).
type ProgressiveMuxer struct {
	w    io.Writer
	ws   io.WriteSeeker
	opts MuxerOptions

	track container.Track
	rate  int
	entry []byte
	began bool
	ended bool

	off         int64
	mdatSizeOff int64
	mdatStart   int64

	durs, sizes []uint32
	dataBytes   int64
}

// NewProgressiveMuxer returns a progressive MP4 muxer writing to w, which must be an io.WriteSeeker (NeedsSeek is true).
func NewProgressiveMuxer(w io.Writer, opts *MuxerOptions) *ProgressiveMuxer {
	m := &ProgressiveMuxer{w: w, mdatSizeOff: -1}
	if ws, ok := w.(io.WriteSeeker); ok {
		m.ws = ws
	}
	if opts != nil {
		m.opts = *opts
	}
	return m
}

// NeedsSeek reports true: the mdat size is back-patched and the moov is written after the samples.
func (m *ProgressiveMuxer) NeedsSeek() bool { return true }

// Begin validates the track, writes the ftyp, and opens the mdat.
func (m *ProgressiveMuxer) Begin(tracks []container.Track) error {
	if m.began {
		return waxerr.New(waxerr.CodeInternal, "mp4: Begin called twice")
	}
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mp4: muxers are single-track, got %d", len(tracks)))
	}
	if m.ws == nil {
		return waxerr.New(waxerr.CodeInvalidRequest, "mp4: progressive output requires a seekable destination")
	}
	t := tracks[0]
	if err := t.Fmt.Valid(); err != nil {
		return err
	}
	if t.Delay < 0 || t.Padding < 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, "mp4: negative gapless trims")
	}
	entry, err := sampleEntryFor(t)
	if err != nil {
		return err
	}
	m.track = t
	m.rate = t.Fmt.Rate
	m.entry = entry
	m.began = true

	if err := m.write(progFtypBox()); err != nil {
		return err
	}
	mdatHeader := append(u32(1), []byte("mdat")...)
	mdatHeader = append(mdatHeader, u64(0)...)
	m.mdatSizeOff = m.off + 8
	m.mdatStart = m.off + 16
	return m.write(mdatHeader)
}

// WritePacket streams one sample into the mdat and records its size and duration.
func (m *ProgressiveMuxer) WritePacket(pkt container.Packet) error {
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
	if int64(len(m.durs)) >= maxSamples {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mp4: more than %d samples", int64(maxSamples)))
	}
	if err := m.write(pkt.Data); err != nil {
		return err
	}
	m.durs = append(m.durs, uint32(pkt.Dur))
	m.sizes = append(m.sizes, uint32(len(pkt.Data)))
	m.dataBytes += int64(len(pkt.Data))
	return nil
}

// End back-patches the mdat size and writes the moov with the full sample tables and the gapless edit list.
func (m *ProgressiveMuxer) End(trailer codec.Trailer) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mp4: End outside Begin")
	}
	m.ended = true

	var rawTotal int64
	for _, d := range m.durs {
		rawTotal += int64(d)
	}
	var edts []byte
	movieDur := rawTotal
	if trailer.Delay > 0 || trailer.Padding > 0 {
		edts = elstBox(trailer.Delay, max(trailer.Samples, 0))
		movieDur = max(trailer.Samples, 0)
	}
	chap := buildChapterTrack(m.opts.Chapters, movieDur, m.rate, m.mdatStart+m.dataBytes)
	var textBytes int64
	if chap != nil {
		if err := m.write(chap.data); err != nil {
			return err
		}
		textBytes = int64(len(chap.data))
	}

	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(16+m.dataBytes+textBytes))
	if _, err := m.ws.Seek(m.mdatSizeOff, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: mdat size seek", err)
	}
	if _, err := m.ws.Write(size[:]); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: mdat size patch", err)
	}
	if _, err := m.ws.Seek(m.off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: seek to end", err)
	}

	var smpb []byte
	if m.track.Codec == codec.AACLC && trailer.Delay > 0 {
		smpb = freeformAtom("iTunSMPB", smpbPayload(trailer.Delay, max(trailer.Samples, 0)))
	}
	udta := udtaBox(m.opts.Tags, m.opts.Chapters, m.opts.Art, smpb)
	moov := progMoovBox(m.rate, m.entry, edts, udta, m.durs, m.sizes, m.mdatStart, movieDur, rawTotal, chap)
	return m.write(moov)
}

func (m *ProgressiveMuxer) write(b []byte) error {
	if _, err := m.w.Write(b); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mp4: write", err)
	}
	m.off += int64(len(b))
	return nil
}

func progFtypBox() []byte {
	return makeBox("ftyp",
		[]byte("M4A "), u32(0),
		[]byte("M4A "), []byte("mp42"), []byte("isom"), []byte("iso2"))
}

func progMoovBox(rate int, entry, edts, udta []byte, durs, sizes []uint32, chunkOff, movieDur, mediaDur int64, chap *chapterTrack) []byte {
	nextTrack := uint32(trackID + 1)
	if chap != nil {
		nextTrack = chapterTrackID + 1
	}
	mvhdVer := durVersion(movieDur)
	mvhd := makeFullBox("mvhd", mvhdVer, 0,
		zeroTimes(mvhdVer),
		u32(uint32(rate)),
		durField(mvhdVer, movieDur),
		u32(0x00010000), u16(0x0100), u16(0), u32(0), u32(0),
		identityMatrix(),
		make([]byte, 24),
		u32(nextTrack))
	parts := [][]byte{mvhd, progTrakBox(rate, entry, edts, durs, sizes, chunkOff, movieDur, mediaDur, chap != nil)}
	if chap != nil {
		parts = append(parts, chap.trakBox(rate, movieDur))
	}
	if udta != nil {
		parts = append(parts, udta)
	}
	return makeBox("moov", parts...)
}

func progTrakBox(rate int, entry, edts []byte, durs, sizes []uint32, chunkOff, movieDur, mediaDur int64, hasChapters bool) []byte {
	tkhdVer := durVersion(movieDur)
	tkhd := makeFullBox("tkhd", tkhdVer, 0x000007,
		zeroTimes(tkhdVer),
		u32(trackID),
		u32(0),
		durField(tkhdVer, movieDur),
		u32(0), u32(0),
		u16(0), u16(0),
		u16(0x0100), u16(0),
		identityMatrix(),
		u32(0), u32(0))
	mdia := progMdiaBox(rate, entry, durs, sizes, chunkOff, mediaDur)
	parts := [][]byte{tkhd}
	if edts != nil {
		parts = append(parts, edts)
	}
	if hasChapters {
		parts = append(parts, makeBox("tref", makeBox("chap", u32(chapterTrackID))))
	}
	return makeBox("trak", append(parts, mdia)...)
}

func progMdiaBox(rate int, entry []byte, durs, sizes []uint32, chunkOff, mediaDur int64) []byte {
	mdhdVer := durVersion(mediaDur)
	mdhd := makeFullBox("mdhd", mdhdVer, 0,
		zeroTimes(mdhdVer),
		u32(uint32(rate)),
		durField(mdhdVer, mediaDur),
		u16(0x55C4), u16(0))
	hdlr := makeFullBox("hdlr", 0, 0,
		u32(0),
		[]byte("soun"),
		u32(0), u32(0), u32(0),
		append([]byte("SoundHandler"), 0))
	minf := progMinfBox(entry, durs, sizes, chunkOff)
	return makeBox("mdia", mdhd, hdlr, minf)
}

func progMinfBox(entry []byte, durs, sizes []uint32, chunkOff int64) []byte {
	smhd := makeFullBox("smhd", 0, 0, u16(0), u16(0))
	dref := makeFullBox("dref", 0, 0, u32(1), makeFullBox("url ", 0, 0x000001))
	dinf := makeBox("dinf", dref)
	stbl := progStblBox(entry, durs, sizes, chunkOff)
	return makeBox("minf", smhd, dinf, stbl)
}

func progStblBox(entry []byte, durs, sizes []uint32, chunkOff int64) []byte {
	n := uint32(len(durs))
	stsd := makeFullBox("stsd", 0, 0, u32(1), entry)

	sttsBody := u32(0)
	entries := uint32(0)
	for i := 0; i < len(durs); {
		j := i + 1
		for j < len(durs) && durs[j] == durs[i] {
			j++
		}
		sttsBody = append(sttsBody, u32(uint32(j-i))...)
		sttsBody = append(sttsBody, u32(durs[i])...)
		entries++
		i = j
	}
	binary.BigEndian.PutUint32(sttsBody, entries)
	stts := makeFullBox("stts", 0, 0, sttsBody)

	stsc := makeFullBox("stsc", 0, 0, u32(1), u32(1), u32(n), u32(1))

	stszBody := append(u32(0), u32(n)...)
	for _, s := range sizes {
		stszBody = append(stszBody, u32(s)...)
	}
	stsz := makeFullBox("stsz", 0, 0, stszBody)

	stco := makeFullBox("stco", 0, 0, u32(1), u32(uint32(chunkOff)))
	if chunkOff > math.MaxUint32 {
		stco = makeFullBox("co64", 0, 0, u32(1), u64(uint64(chunkOff)))
	}

	return makeBox("stbl", stsd, stts, stsc, stsz, stco)
}

const (
	chapterTrackID = trackID + 1

	chapterTimescale = 1000

	maxChapterTitleBytes = 1<<16 - 2
)

type chapterTrack struct {
	data     []byte
	durs     []uint32
	sizes    []uint32
	chunkOff int64
	mediaDur int64

	startTicks int64
}

func buildChapterTrack(chapters []container.Chapter, movieSamples int64, rate int, chunkOff int64) *chapterTrack {
	if len(chapters) == 0 {
		return nil
	}
	movieEnd := mulDivSat(movieSamples, chapterTimescale, int64(rate))
	starts := make([]int64, len(chapters))
	for i, ch := range chapters {
		starts[i] = min(mulDivSat(int64(ch.Start), chapterTimescale, int64(time.Second)), movieEnd)
	}
	end := movieEnd
	if last := chapters[len(chapters)-1]; last.End > 0 {
		end = min(mulDivSat(int64(last.End), chapterTimescale, int64(time.Second)), movieEnd)
	}

	c := &chapterTrack{chunkOff: chunkOff, startTicks: starts[0]}
	for i, ch := range chapters {
		next := end
		if i+1 < len(chapters) {
			next = starts[i+1]
		}
		dur := min(max(next-starts[i], 1), math.MaxUint32)
		title := truncateRunes(ch.Title, maxChapterTitleBytes)
		c.data = append(c.data, u16(uint16(len(title)))...)
		c.data = append(c.data, title...)
		c.durs = append(c.durs, uint32(dur))
		c.sizes = append(c.sizes, uint32(2+len(title)))
		c.mediaDur += dur
	}
	return c
}

func (c *chapterTrack) trakBox(movieRate int, movieDur int64) []byte {
	empty, played := c.editDurs(movieRate, movieDur)
	dur := empty + played
	tkhdVer := durVersion(dur)
	tkhd := makeFullBox("tkhd", tkhdVer, 0x000002,
		zeroTimes(tkhdVer),
		u32(chapterTrackID),
		u32(0),
		durField(tkhdVer, dur),
		u32(0), u32(0),
		u16(0), u16(0),
		u16(0), u16(0),
		identityMatrix(),
		u32(0), u32(0))
	mdia := makeBox("mdia", c.mdhdBox(), chapterHdlrBox(), c.minfBox())
	if edts := c.edtsBox(empty, played); edts != nil {
		return makeBox("trak", tkhd, edts, mdia)
	}
	return makeBox("trak", tkhd, mdia)
}

func (c *chapterTrack) editDurs(movieRate int, movieDur int64) (empty, played int64) {
	empty = min(mulDivSat(c.startTicks, int64(movieRate), chapterTimescale), movieDur)
	played = min(mulDivSat(c.mediaDur, int64(movieRate), chapterTimescale), movieDur-empty)
	return empty, played
}

func (c *chapterTrack) edtsBox(empty, played int64) []byte {
	if empty <= 0 {
		return nil
	}
	elst := makeFullBox("elst", 1, 0,
		u32(2),
		u64(uint64(empty)), u64(math.MaxUint64),
		u16(1), u16(0),
		u64(uint64(played)), u64(0),
		u16(1), u16(0))
	return makeBox("edts", elst)
}

func (c *chapterTrack) mdhdBox() []byte {
	ver := durVersion(c.mediaDur)
	return makeFullBox("mdhd", ver, 0,
		zeroTimes(ver),
		u32(chapterTimescale),
		durField(ver, c.mediaDur),
		u16(0x55C4), u16(0))
}

func chapterHdlrBox() []byte {
	return makeFullBox("hdlr", 0, 0,
		u32(0),
		[]byte("text"),
		u32(0), u32(0), u32(0),
		append([]byte("SubtitleHandler"), 0))
}

func (c *chapterTrack) minfBox() []byte {
	gmin := makeFullBox("gmin", 0, 0,
		u16(0x0040),
		u16(0x8000), u16(0x8000), u16(0x8000),
		u16(0), u16(0))
	gmhd := makeBox("gmhd", gmin, makeBox("text", identityMatrix()))
	dref := makeFullBox("dref", 0, 0, u32(1), makeFullBox("url ", 0, 0x000001))
	dinf := makeBox("dinf", dref)
	stbl := progStblBox(textSampleEntry(), c.durs, c.sizes, c.chunkOff)
	return makeBox("minf", gmhd, dinf, stbl)
}

func textSampleEntry() []byte {
	return makeBox("text",
		make([]byte, 6), u16(1),
		u32(0),
		u32(0),
		u16(0), u16(0), u16(0),
		u16(0), u16(0), u16(0), u16(0),
		u64(0),
		u16(0), u16(0),
		[]byte{0},
		u16(0),
		u16(0), u16(0), u16(0),
		[]byte{0})
}
