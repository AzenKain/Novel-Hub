package flacn

import (
	"encoding/binary"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

var _ container.Muxer = (*Muxer)(nil)

const seekInterval = 10

const maxWriteSeekPoints = 1 << 14

// placeholderSample marks an unfilled seek point (RFC 9639 section 8.5); placeholders must trail real points, which holds because points fill in stream order.
const placeholderSample = ^uint64(0)

type seekRec struct {
	sample, off, dur int64
}

// MuxerOptions configures writing.
type MuxerOptions struct {
	MD5  func() [16]byte
	Tags []container.Tag
}

// Muxer writes one FLAC track as a native FLAC stream.
type Muxer struct {
	w    io.Writer
	ws   io.WriteSeeker
	opts MuxerOptions

	si         flac.StreamInfo
	wroteTotal int64
	off        int64
	firstFrame int64

	seekOff  int64
	points   []seekRec
	interval int64
	target   int64
	filled   int

	minFrame, maxFrame int
	samples            int64
	began, ended       bool
}

// NewMuxer returns a FLAC muxer writing to w.
func NewMuxer(w io.Writer, opts *MuxerOptions) *Muxer {
	m := &Muxer{w: w}
	if ws, ok := w.(io.WriteSeeker); ok {
		m.ws = ws
	}
	if opts != nil {
		m.opts = *opts
	}
	return m
}

// NeedsSeek reports false: native FLAC has a compliant streaming form.
func (m *Muxer) NeedsSeek() bool { return false }

// Begin validates the track and writes the stream marker, STREAMINFO, and the SEEKTABLE reservation.
func (m *Muxer) Begin(tracks []container.Track) error {
	if m.began {
		return waxerr.New(waxerr.CodeInternal, "flacn: Begin called twice")
	}
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("flacn: muxers are single-track, got %d", len(tracks)))
	}
	t := tracks[0]
	if t.Codec != codec.FLAC {
		return waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf("flacn: cannot mux codec %q", t.Codec))
	}
	if t.Delay != 0 || t.Padding != 0 {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "flacn: FLAC signals no gapless trims (lossless streams have none)")
	}
	si, err := flac.ParseStreamInfo(t.CodecConfig)
	if err != nil {
		return err
	}
	if want := si.PCMFormat(); t.Fmt != want {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("flacn: track format %v does not match STREAMINFO (want %v)", t.Fmt, want))
	}
	if si.Samples == 0 && t.Samples > 0 && t.Samples < 1<<36 {
		si.Samples = t.Samples
	}
	m.si = si
	m.wroteTotal = si.Samples
	m.began = true

	table := m.ws != nil && si.Samples > 0
	vc := vorbisCommentBlock(m.opts.Tags)
	head := [4]byte{0x80}
	if table || vc != nil {
		head[0] = 0x00
	}
	head[1], head[2], head[3] = 0, 0, flac.StreamInfoLen
	body, err := si.MarshalBinary()
	if err != nil {
		return err
	}
	if err := m.write([]byte("fLaC"), head[:], body); err != nil {
		return err
	}

	if vc != nil {
		hdr := [4]byte{4, byte(len(vc) >> 16), byte(len(vc) >> 8), byte(len(vc))}
		if !table {
			hdr[0] |= 0x80
		}
		if err := m.write(hdr[:], vc); err != nil {
			return err
		}
	}

	if table {
		m.interval = int64(seekInterval) * int64(si.Rate)
		n := (si.Samples + m.interval - 1) / m.interval
		n = min(n, maxWriteSeekPoints)
		m.points = make([]seekRec, 0, n)
		size := n * 18
		hdr := [4]byte{0x80 | 3, byte(size >> 16), byte(size >> 8), byte(size)}
		if err := m.write(hdr[:]); err != nil {
			return err
		}
		m.seekOff = m.off
		point := make([]byte, 18)
		binary.BigEndian.PutUint64(point, placeholderSample)
		for range n {
			if err := m.write(point); err != nil {
				return err
			}
		}
	}
	m.firstFrame = m.off
	return nil
}

// WritePacket appends one frame and accounts for the header patch and the seek table.
func (m *Muxer) WritePacket(pkt container.Packet) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "flacn: WritePacket outside Begin/End")
	}
	if pkt.Track != 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("flacn: no track %d", pkt.Track))
	}
	if len(pkt.Data) == 0 || pkt.Dur <= 0 || pkt.Dur > flac.MaxBlockSize {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("flacn: packet of %d bytes, %d samples", len(pkt.Data), pkt.Dur))
	}
	if m.points != nil && pkt.PTS >= m.target && len(m.points) < cap(m.points) {
		m.points = append(m.points, seekRec{sample: pkt.PTS, off: m.off - m.firstFrame, dur: pkt.Dur})
		m.target = (pkt.PTS/m.interval + 1) * m.interval
	}
	if err := m.write(pkt.Data); err != nil {
		return err
	}
	if n := len(pkt.Data); n < 1<<24 {
		if m.minFrame == 0 || n < m.minFrame {
			m.minFrame = n
		}
		m.maxFrame = max(m.maxFrame, n)
	}
	m.samples += pkt.Dur
	return nil
}

// End finalizes the stream.
func (m *Muxer) End(trailer codec.Trailer) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "flacn: End outside Begin")
	}
	m.ended = true
	if trailer.Delay != 0 || trailer.Padding != 0 {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "flacn: FLAC signals no gapless trims")
	}
	if trailer.Samples >= 0 && trailer.Samples != m.samples {
		return waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("flacn: trailer says %d samples, wrote %d", trailer.Samples, m.samples))
	}

	if m.ws == nil {
		if m.wroteTotal != 0 && m.wroteTotal != m.samples {
			return waxerr.New(waxerr.CodeInternal,
				fmt.Sprintf("flacn: header promised %d samples, wrote %d (unseekable output)", m.wroteTotal, m.samples))
		}
		return nil
	}

	si := m.si
	si.MinFrame, si.MaxFrame = m.minFrame, m.maxFrame
	si.Samples = 0
	if m.samples < 1<<36 {
		si.Samples = m.samples
	}
	if m.opts.MD5 != nil {
		si.MD5 = m.opts.MD5()
	}
	body, err := si.MarshalBinary()
	if err != nil {
		return err
	}
	if err := m.patch(8, body); err != nil {
		return err
	}

	if len(m.points) > m.filled {
		buf := make([]byte, 18*(len(m.points)-m.filled))
		for i, p := range m.points[m.filled:] {
			b := buf[i*18:]
			binary.BigEndian.PutUint64(b, uint64(p.sample))
			binary.BigEndian.PutUint64(b[8:], uint64(p.off))
			b[16], b[17] = byte(p.dur>>8), byte(p.dur)
		}
		if err := m.patch(m.seekOff+int64(m.filled)*18, buf); err != nil {
			return err
		}
		m.filled = len(m.points)
	}
	if _, err := m.ws.Seek(m.off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "flacn: seeking to end", err)
	}
	return nil
}

func (m *Muxer) write(parts ...[]byte) error {
	for _, p := range parts {
		n, err := m.w.Write(p)
		m.off += int64(n)
		if err != nil {
			return waxerr.Wrap(waxerr.CodeOutputUnwritable, "flacn: write", err)
		}
	}
	return nil
}

func (m *Muxer) patch(off int64, parts ...[]byte) error {
	if _, err := m.ws.Seek(off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "flacn: seek for patch", err)
	}
	for _, p := range parts {
		if _, err := m.ws.Write(p); err != nil {
			return waxerr.Wrap(waxerr.CodeOutputUnwritable, "flacn: patch", err)
		}
	}
	return nil
}

const maxCommentBytes = 48 << 10

// vorbisCommentBlock renders tags as a VORBIS_COMMENT body (RFC 9639 section 8.6: little-endian lengths, UTF-8 KEY=value comments), nil for no tags.
func vorbisCommentBlock(tags []container.Tag) []byte {
	if len(tags) == 0 {
		return nil
	}
	const vendor = "WaxFlow"
	body := binary.LittleEndian.AppendUint32(nil, uint32(len(vendor)))
	body = append(body, vendor...)
	countAt := len(body)
	body = binary.LittleEndian.AppendUint32(body, 0)
	count := uint32(0)
	for _, t := range tags {
		if !container.ValidTagKey(t.Key) {
			continue
		}
		c := t.Key + "=" + t.Value
		if len(body)+4+len(c) > maxCommentBytes {
			continue
		}
		body = binary.LittleEndian.AppendUint32(body, uint32(len(c)))
		body = append(body, c...)
		count++
	}
	binary.LittleEndian.PutUint32(body[countAt:], count)
	return body
}
