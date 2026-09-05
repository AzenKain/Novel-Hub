package mka

import (
	"encoding/binary"
	"io"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

var _ container.Muxer = (*Muxer)(nil)

const (
	idEBMLVersion        = 0x4286
	idEBMLReadVersion    = 0x42F7
	idEBMLMaxIDLength    = 0x42F2
	idEBMLMaxSizeLength  = 0x42F3
	idDocTypeVersion     = 0x4287
	idDocTypeReadVersion = 0x4285

	idMuxingApp  = 0x4D80
	idWritingApp = 0x5741

	idTrackUID   = 0x73C5
	idFlagLacing = 0x9C
)

const (
	muxTrackUID = 1
	muxAppName  = "WaxFlow"
)

const (
	maxClusterMs       = 4000
	clusterTargetBytes = 1 << 20
)

// MuxerOptions configures the Matroska/WebM muxer.
type MuxerOptions struct {
	WebM bool
	Tags []container.Tag
}

// Muxer writes one audio track as Matroska (.mka/.mkv) or WebM.
type Muxer struct {
	w     io.Writer
	ws    io.WriteSeeker
	opts  MuxerOptions
	begun bool
	ended bool

	codecID codec.ID
	rate    int
	delay   int64

	off int64

	segDataOff int64
	segSizeOff int64
	durOff     int64
	cueSeekOff int64

	cues      []cuePoint
	clusters  int64
	cueStride int64

	rawSamples int64

	cluster     []byte
	clusterMs   int64
	haveCluster bool

	pending     codec.Packet
	havePending bool
}

type cuePoint struct {
	timeMs int64
	pos    int64
}

// NewMuxer returns a Matroska/WebM muxer writing to w.
func NewMuxer(w io.Writer, opts *MuxerOptions) *Muxer {
	m := &Muxer{
		w:          w,
		segDataOff: -1,
		segSizeOff: -1,
		durOff:     -1,
		cueSeekOff: -1,
		cueStride:  1,
	}
	if ws, ok := w.(io.WriteSeeker); ok {
		if at, err := ws.Seek(0, io.SeekCurrent); err == nil && at >= 0 {
			m.ws, m.off = ws, at
		}
	}
	if opts != nil {
		m.opts = *opts
	}
	return m
}

// NeedsSeek is false: Matroska streams with an unknown-size Segment and definite-size Clusters.
func (m *Muxer) NeedsSeek() bool { return false }

// Begin validates the track and writes the EBML header, Segment header, SeekHead, Info, and Tracks, reserving the slots End patches.
func (m *Muxer) Begin(tracks []container.Track) error {
	if m.begun {
		return waxerr.New(waxerr.CodeInternal, "mka: Begin called twice")
	}
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeInvalidRequest, "mka: muxers are single-track")
	}
	t := tracks[0]
	if err := t.Fmt.Valid(); err != nil {
		return err
	}
	codecID, err := matroskaCodecID(t.Codec, t.Fmt)
	if err != nil {
		return err
	}
	if m.opts.WebM && t.Codec != codec.Opus && t.Codec != codec.Vorbis {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			"mka: webm carries only Opus and Vorbis audio; use .mka for "+string(t.Codec))
	}
	priv, err := codecPrivate(t.Codec, t.CodecConfig)
	if err != nil {
		return err
	}
	m.codecID = t.Codec
	m.rate = t.Fmt.Rate
	m.delay = t.Delay
	if t.Codec == codec.Opus {
		m.delay = int64(binary.LittleEndian.Uint16(t.CodecConfig[10:12]))
	}
	m.begun = true

	header := m.ebmlHeader()
	header = m.appendSegmentHeader(header, t, codecID, priv)
	return m.write(header)
}

// WritePacket appends one codec packet as a SimpleBlock.
func (m *Muxer) WritePacket(pkt container.Packet) error {
	if !m.begun || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mka: WritePacket outside Begin/End")
	}
	if pkt.Track != 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, "mka: single-track muxer")
	}
	if m.havePending {
		if err := m.emitBlock(m.pending, 0); err != nil {
			return err
		}
	}
	m.pending = codec.Packet{
		Data: append([]byte(nil), pkt.Data...),
		PTS:  pkt.PTS,
		Dur:  pkt.Dur,
	}
	m.havePending = true
	if pkt.Dur > 0 {
		m.rawSamples += pkt.Dur
	}
	return nil
}

// End emits the held final packet (with DiscardPadding when the trailer carries end padding), flushes the last cluster, and finishes the stream.
func (m *Muxer) End(trailer codec.Trailer) error {
	if !m.begun || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mka: End outside Begin")
	}
	m.ended = true
	if m.havePending {
		discardNS := samplesToNs(trailer.Padding, m.rate)
		if err := m.emitBlock(m.pending, discardNS); err != nil {
			return err
		}
		m.havePending = false
	}
	if err := m.flushCluster(); err != nil {
		return err
	}
	return m.finish(trailer)
}

func (m *Muxer) finish(trailer codec.Trailer) error {
	if m.ws == nil {
		return nil
	}
	cuesPos := int64(-1)
	if len(m.cues) > 0 {
		cuesPos = m.off - m.segDataOff
		if err := m.write(m.cuesElement()); err != nil {
			return err
		}
	}
	if m.durOff >= 0 {
		if n := finalSamples(trailer, m.rawSamples); n > 0 {
			if err := m.patch(m.durOff, appendFloat(nil, idDuration, durationTicks(n, m.rate))); err != nil {
				return err
			}
		}
	}
	if m.cueSeekOff >= 0 {
		if cuesPos >= 0 {
			var v [8]byte
			binary.BigEndian.PutUint64(v[:], uint64(cuesPos))
			if err := m.patch(m.cueSeekOff+seekPosValueOff, v[:]); err != nil {
				return err
			}
		} else if err := m.patch(m.cueSeekOff, appendVoid(nil, seekEntryLen)); err != nil {
			return err
		}
	}
	if size := m.off - m.segDataOff; m.segSizeOff >= 0 && size < (1<<56)-1 {
		var v [8]byte
		binary.BigEndian.PutUint64(v[:], uint64(size))
		v[0] = 0x01
		if err := m.patch(m.segSizeOff, v[:]); err != nil {
			return err
		}
	}
	if _, err := m.ws.Seek(m.off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mka: seeking to end", err)
	}
	return nil
}

func finalSamples(trailer codec.Trailer, raw int64) int64 {
	if trailer.Samples >= 0 {
		return trailer.Samples
	}
	return raw - trailer.Delay - trailer.Padding
}

func (m *Muxer) cuesElement() []byte {
	var body []byte
	for _, c := range m.cues {
		var pos []byte
		pos = appendUint(pos, idCueTrack, muxTrackNumber)
		pos = appendUint(pos, idCueClusterPosition, uint64(c.pos))
		var point []byte
		point = appendUint(point, idCueTime, uint64(c.timeMs))
		point = appendElement(point, idCueTrackPositions, pos)
		body = appendElement(body, idCuePoint, point)
	}
	return appendElement(nil, idCues, body)
}

func (m *Muxer) emitBlock(pkt codec.Packet, discardNS int64) error {
	ptsMs := msAt(pkt.PTS, m.rate)
	if !m.haveCluster || ptsMs-m.clusterMs > maxClusterMs || len(m.cluster) >= clusterTargetBytes {
		if err := m.flushCluster(); err != nil {
			return err
		}
		m.clusterMs = ptsMs
		m.cluster = appendUint(m.cluster[:0], idTimestamp, uint64(ptsMs))
		m.haveCluster = true
	}
	rel := int16(ptsMs - m.clusterMs)
	if discardNS > 0 {
		block := blockBody(rel, pkt.Data, 0x00)
		var group []byte
		group = appendElement(group, idBlock, block)
		group = appendElement(group, idDiscardPadding, beIntBytes(discardNS))
		m.cluster = appendElement(m.cluster, idBlockGroup, group)
		return nil
	}
	block := blockBody(rel, pkt.Data, 0x80)
	m.cluster = appendElement(m.cluster, idSimpleBlock, block)
	return nil
}

func (m *Muxer) flushCluster() error {
	if !m.haveCluster {
		return nil
	}
	m.haveCluster = false
	m.recordCue()
	out := appendElement(nil, idCluster, m.cluster)
	m.cluster = m.cluster[:0]
	return m.write(out)
}

func (m *Muxer) recordCue() {
	if m.ws == nil {
		return
	}
	if m.clusters%m.cueStride == 0 {
		if len(m.cues) == maxCuePoints {
			for i := 0; i < maxCuePoints/2; i++ {
				m.cues[i] = m.cues[2*i]
			}
			m.cues = m.cues[:maxCuePoints/2]
			m.cueStride *= 2
		}
		m.cues = append(m.cues, cuePoint{timeMs: m.clusterMs, pos: m.off - m.segDataOff})
	}
	m.clusters++
}

func blockBody(rel int16, frame []byte, flags byte) []byte {
	b := appendVint(nil, muxTrackNumber)
	b = append(b, byte(uint16(rel)>>8), byte(uint16(rel)), flags)
	return append(b, frame...)
}

const muxTrackNumber = 1

func msAt(sample int64, rate int) int64 {
	if sample <= 0 || rate <= 0 {
		return 0
	}
	r := int64(rate)
	sec := sample / r
	rem := sample % r
	return sec*1000 + (rem*1000+r/2)/r
}

func (m *Muxer) ebmlHeader() []byte {
	docType, docVersion := "matroska", uint64(4)
	if m.opts.WebM {
		docType, docVersion = "webm", 2
	}
	var body []byte
	body = appendUint(body, idEBMLVersion, 1)
	body = appendUint(body, idEBMLReadVersion, 1)
	body = appendUint(body, idEBMLMaxIDLength, 4)
	body = appendUint(body, idEBMLMaxSizeLength, 8)
	body = appendString(body, idDocType, docType)
	body = appendUint(body, idDocTypeVersion, docVersion)
	body = appendUint(body, idDocTypeReadVersion, 2)
	return appendElement(nil, idEBML, body)
}

func (m *Muxer) appendSegmentHeader(dst []byte, t container.Track, codecID string, priv []byte) []byte {
	dst = appendID(dst, idSegment)
	m.segSizeOff = m.off + int64(len(dst))
	dst = append(dst, unknownSizeVint...)
	m.segDataOff = m.off + int64(len(dst))

	info, durSlot := m.infoElement(t)
	entry := appendElement(nil, idTrackEntry, m.trackEntry(t, codecID, priv))
	tracks := appendElement(nil, idTracks, entry)
	seekHead, cueSlot := m.seekHead(int64(len(info)))

	if durSlot >= 0 {
		m.durOff = m.segDataOff + int64(len(seekHead)+durSlot)
	}
	if cueSlot >= 0 {
		m.cueSeekOff = m.segDataOff + int64(cueSlot)
	}
	dst = append(dst, seekHead...)
	dst = append(dst, info...)
	return append(dst, tracks...)
}

var unknownSizeVint = []byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

const (
	seekEntryLen    = 21
	seekPosValueOff = 13
	durationLen     = 11
)

func (m *Muxer) seekHead(infoLen int64) (elem []byte, cueSlot int) {
	n := 2
	if m.ws != nil {
		n = 3
	}
	shLen := int64(len(appendElement(nil, idSeekHead, make([]byte, n*seekEntryLen))))
	var body []byte
	body = appendSeekEntry(body, idInfo, uint64(shLen))
	body = appendSeekEntry(body, idTracks, uint64(shLen+infoLen))
	cueSlot = -1
	if m.ws != nil {
		cueSlot = len(body)
		body = appendSeekEntry(body, idCues, 0)
	}
	elem = appendElement(nil, idSeekHead, body)
	if cueSlot >= 0 {
		cueSlot += len(elem) - len(body)
	}
	return elem, cueSlot
}

func appendSeekEntry(dst []byte, target uint32, pos uint64) []byte {
	var body []byte
	body = appendElement(body, idSeekID, appendID(nil, target))
	body = appendUintFixed(body, idSeekPosition, pos, 8)
	return appendElement(dst, idSeek, body)
}

func (m *Muxer) infoElement(t container.Track) (elem []byte, durSlot int) {
	var body []byte
	body = appendUint(body, idTimestampScale, defaultTimestampScale)
	body = appendString(body, idMuxingApp, muxAppName)
	body = appendString(body, idWritingApp, muxAppName)
	durSlot = -1
	switch {
	case t.Samples > 0:
		durSlot = len(body)
		body = appendFloat(body, idDuration, durationTicks(t.Samples, m.rate))
	case m.ws != nil:
		durSlot = len(body)
		body = appendVoid(body, durationLen)
	}
	elem = appendElement(nil, idInfo, body)
	if durSlot >= 0 {
		durSlot += len(elem) - len(body)
	}
	return elem, durSlot
}

func durationTicks(samples int64, rate int) float64 {
	if samples <= 0 || rate <= 0 {
		return 0
	}
	const ticksPerSecond = 1e9 / defaultTimestampScale
	return float64(samples) * ticksPerSecond / float64(rate)
}

func (m *Muxer) trackEntry(t container.Track, codecID string, priv []byte) []byte {
	var e []byte
	e = appendUint(e, idTrackNumber, muxTrackNumber)
	e = appendUint(e, idTrackUID, muxTrackUID)
	e = appendUint(e, idTrackType, trackTypeAudio)
	e = appendUint(e, idFlagLacing, 0)
	e = appendString(e, idCodecID, codecID)
	if len(priv) > 0 {
		e = appendElement(e, idCodecPrivate, priv)
	}
	if m.delay > 0 {
		e = appendUint(e, idCodecDelay, uint64(samplesToNs(m.delay, m.rate)))
		// Opus carries an 80 ms seek pre-roll (RFC 7845); the demuxer reads it to land a seek far enough ahead for the decoder to reconverge.
		if t.Codec == codec.Opus {
			e = appendUint(e, idSeekPreRoll, uint64(samplesToNs(3840, m.rate)))
		}
	}
	e = appendElement(e, idAudio, m.audioBody(t))
	return e
}

func (m *Muxer) audioBody(t container.Track) []byte {
	var a []byte
	a = appendFloat(a, idSamplingFreq, float64(t.Fmt.Rate))
	a = appendUint(a, idChannels, uint64(t.Fmt.Channels))
	if t.Codec == codec.PCM {
		a = appendUint(a, idBitDepth, uint64(pcmContainerBits(t.Fmt)))
	}
	return a
}

func (m *Muxer) write(b []byte) error {
	n, err := m.w.Write(b)
	m.off += int64(n)
	if err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mka: write", err)
	}
	return nil
}

func (m *Muxer) patch(off int64, b []byte) error {
	if _, err := m.ws.Seek(off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mka: seek for patch", err)
	}
	if _, err := m.ws.Write(b); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mka: patch", err)
	}
	return nil
}

func matroskaCodecID(id codec.ID, f audio.Format) (string, error) {
	switch id {
	case codec.Opus:
		return "A_OPUS", nil
	case codec.Vorbis:
		return "A_VORBIS", nil
	case codec.FLAC:
		return "A_FLAC", nil
	case codec.AACLC:
		return "A_AAC", nil
	case codec.PCM:
		if f.Type == audio.Float {
			return "A_PCM/FLOAT/IEEE", nil
		}
		return "A_PCM/INT/LIT", nil
	}
	return "", waxerr.New(waxerr.CodeUnsupportedFormat,
		"mka: cannot mux codec "+string(id)+" (opus, vorbis, flac, aac-lc, pcm)")
}

func codecPrivate(id codec.ID, cfg []byte) ([]byte, error) {
	switch id {
	case codec.Opus:
		if len(cfg) < 19 || string(cfg[:8]) != "OpusHead" {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mka: Opus track config is not an OpusHead")
		}
		return cfg, nil
	case codec.Vorbis:
		if len(cfg) == 0 {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mka: Vorbis track has no codec config")
		}
		return cfg, nil
	case codec.AACLC:
		if len(cfg) == 0 {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mka: AAC track has no AudioSpecificConfig")
		}
		return cfg, nil
	case codec.FLAC:
		if len(cfg) != flac.StreamInfoLen {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mka: FLAC track config is not a STREAMINFO block")
		}
		priv := append([]byte("fLaC"), 0x80, 0x00, 0x00, byte(flac.StreamInfoLen))
		return append(priv, cfg...), nil
	case codec.PCM:
		return nil, nil
	}
	return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "mka: no CodecPrivate for codec "+string(id))
}

func pcmContainerBits(f audio.Format) int {
	if f.Type == audio.Float {
		return 32
	}
	return (f.BitDepth + 7) / 8 * 8
}
