package mpa

import (
	"encoding/binary"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/mp3"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

var _ container.Muxer = (*Muxer)(nil)

// Muxer writes one MP3 track as a bare Layer III elementary stream led by a Xing/Info metadata frame with a LAME-format gapless extension.
type Muxer struct {
	w    io.Writer
	ws   io.WriteSeeker
	opts MuxerOptions

	samples int64
	rate    int
	off     int64
	id3Len  int64
	infoLen int

	hdr [mp3.HeaderLen]byte
	h   mp3.Header

	frameOff []int64
	stride   int

	audioFrames  int
	began, ended bool
	wroteInfo    bool
}

// MuxerOptions configures writing.
type MuxerOptions struct {
	Delay int
	VBR   bool
	Tags  []container.Tag
}

const tocSampleCap = 2048

// NewMuxer returns an MP3 muxer writing to w.
func NewMuxer(w io.Writer, opts *MuxerOptions) *Muxer {
	m := &Muxer{w: w, stride: 1}
	if ws, ok := w.(io.WriteSeeker); ok {
		m.ws = ws
	}
	if opts != nil {
		m.opts = *opts
	}
	return m
}

// NeedsSeek reports false: the elementary stream has a compliant streaming form and the gapless tag is written up front from the projected length.
func (m *Muxer) NeedsSeek() bool { return false }

// Begin validates the track and records the projected length; the metadata frame is deferred to the first packet, whose header is its template.
func (m *Muxer) Begin(tracks []container.Track) error {
	if m.began {
		return waxerr.New(waxerr.CodeInternal, "mpa: Begin called twice")
	}
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mpa: muxers are single-track, got %d", len(tracks)))
	}
	t := tracks[0]
	if t.Codec != codec.MP3 {
		return waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf("mpa: cannot mux codec %q", t.Codec))
	}
	m.samples = t.Samples
	m.rate = t.Fmt.Rate
	m.began = true
	if id3 := id3v2Tag(m.opts.Tags); id3 != nil {
		if err := m.write(id3); err != nil {
			return err
		}
		m.id3Len = int64(len(id3))
	}
	return nil
}

// WritePacket writes the metadata frame (once, from the first packet's header) followed by the audio frame.
func (m *Muxer) WritePacket(pkt container.Packet) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mpa: WritePacket outside Begin/End")
	}
	if pkt.Track != 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mpa: no track %d", pkt.Track))
	}
	if len(pkt.Data) < mp3.HeaderLen {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf("mpa: packet of %d bytes", len(pkt.Data)))
	}
	if !m.wroteInfo {
		h, err := mp3.ParseHeader(pkt.Data)
		if err != nil {
			return waxerr.Wrap(waxerr.CodeInternal, "mpa: first packet header", err)
		}
		copy(m.hdr[:], pkt.Data[:mp3.HeaderLen])
		m.h = h
		if m.opts.VBR {
			m.h = xingHeader(h)
		}
		delay, padding, frames := m.projectGapless()
		if info := m.buildInfoFrame(delay, padding, frames, nil, 0); info != nil {
			if err := m.write(info); err != nil {
				return err
			}
			m.infoLen = len(info)
		}
		m.wroteInfo = true
	}
	if m.opts.VBR {
		if m.audioFrames%m.stride == 0 {
			if len(m.frameOff) == tocSampleCap {
				for i := 0; i < tocSampleCap/2; i++ {
					m.frameOff[i] = m.frameOff[2*i]
				}
				m.frameOff = m.frameOff[:tocSampleCap/2]
				m.stride *= 2
			}
			m.frameOff = append(m.frameOff, m.off)
		}
	}
	if err := m.write(pkt.Data); err != nil {
		return err
	}
	m.audioFrames++
	return nil
}

// End back-patches the metadata frame with the encoder's exact gapless trailer, audio-frame count, and (VBR) the measured byte count and TOC when the writer is seekable.
func (m *Muxer) End(trailer codec.Trailer) error {
	if !m.began || m.ended {
		return waxerr.New(waxerr.CodeInternal, "mpa: End outside Begin")
	}
	m.ended = true
	if m.ws == nil || m.infoLen == 0 {
		return nil
	}
	var toc []byte
	if m.opts.VBR {
		toc = m.measureTOC()
	}
	info := m.buildInfoFrame(int(trailer.Delay), int(trailer.Padding), m.audioFrames, toc, m.off-m.id3Len)
	if info == nil || len(info) != m.infoLen {
		return nil
	}
	if _, err := m.ws.Seek(m.id3Len, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mpa: seek for patch", err)
	}
	if _, err := m.ws.Write(info); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mpa: patch", err)
	}
	if _, err := m.ws.Seek(m.off, io.SeekStart); err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mpa: seeking to end", err)
	}
	return nil
}

func (m *Muxer) measureTOC() []byte {
	toc := make([]byte, 100)
	total := m.off - m.id3Len
	if len(m.frameOff) == 0 || total <= 0 || m.audioFrames == 0 {
		for i := range toc {
			toc[i] = byte(min(i*256/100, 255))
		}
		return toc
	}
	for i := range toc {
		frame := m.audioFrames * i / 100
		j := min(frame/m.stride, len(m.frameOff)-1)
		toc[i] = byte(min((m.frameOff[j]-m.id3Len)*256/total, 255))
	}
	return toc
}

func (m *Muxer) projectGapless() (delay, padding, frames int) {
	delay = m.opts.Delay
	if m.samples < 0 {
		return delay, 0, 0
	}
	frames = mp3.FramesFor(m.samples, m.rate)
	total := int64(frames) * int64(m.h.SamplesPerFrame())
	if p := total - m.samples - int64(delay); p > 0 {
		padding = int(p)
	}
	return delay, padding, frames
}

func (m *Muxer) write(b []byte) error {
	n, err := m.w.Write(b)
	m.off += int64(n)
	if err != nil {
		return waxerr.Wrap(waxerr.CodeOutputUnwritable, "mpa: write", err)
	}
	return nil
}

func xingHeader(h mp3.Header) mp3.Header {
	need := mp3.HeaderLen + h.SideInfoLen() + xingLayoutLen
	for _, kbps := range legalRates(h) {
		h.Bitrate = kbps * 1000
		h.Padding = false
		if h.Size() >= need {
			return h
		}
	}
	return h
}

func legalRates(h mp3.Header) []int {
	if h.Version == mp3.MPEG1 {
		return []int{32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}
	}
	return []int{8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}
}

const (
	xingFlagFrames = 1
	xingFlagBytes  = 2
	xingFlagTOC    = 4
	xingLayoutLen  = 4 + 4 + 4 + 4 + 100 + 9 + 12 + 3
)

func (m *Muxer) buildInfoFrame(delay, padding, frames int, toc []byte, bytes int64) []byte {
	h := m.h
	size := h.Size()
	off := mp3.HeaderLen + h.SideInfoLen()
	if size == 0 || off+12 > size {
		return nil
	}
	frame := make([]byte, size)
	hdr := headerBytesFor(h, m.hdr)
	copy(frame[:mp3.HeaderLen], hdr[:])
	frame[1] |= 1

	magic, flags := "Info", uint32(xingFlagFrames)
	if m.opts.VBR {
		magic, flags = "Xing", xingFlagFrames|xingFlagBytes|xingFlagTOC
	}
	copy(frame[off:], magic)
	binary.BigEndian.PutUint32(frame[off+4:], flags)
	p := off + 8
	binary.BigEndian.PutUint32(frame[p:], uint32(frames))
	p += 4
	if flags&xingFlagBytes != 0 {
		if p+4 > size {
			return frame
		}
		if bytes > 0 && bytes <= int64(^uint32(0)) {
			binary.BigEndian.PutUint32(frame[p:], uint32(bytes))
		}
		p += 4
	}
	if flags&xingFlagTOC != 0 {
		if p+100 > size {
			return frame
		}
		if toc == nil {
			for i := 0; i < 100; i++ {
				frame[p+i] = byte(min(i*256/100, 255))
			}
		} else {
			copy(frame[p:], toc)
		}
		p += 100
	}
	if p+24 <= size {
		copy(frame[p:], "WaxFlow01")
		if delay >= 0 && delay < 1<<12 && padding >= 0 && padding < 1<<12 {
			frame[p+21] = byte(delay >> 4)
			frame[p+22] = byte((delay&0xF)<<4 | (padding>>8)&0xF)
			frame[p+23] = byte(padding)
		}
	}
	return frame
}

func headerBytesFor(h mp3.Header, tmpl [mp3.HeaderLen]byte) [mp3.HeaderLen]byte {
	b := tmpl
	bi := 0
	for i, kbps := range legalRates(h) {
		if kbps*1000 == h.Bitrate {
			bi = i + 1
			break
		}
	}
	pad := byte(0)
	if h.Padding {
		pad = 1
	}
	b[2] = byte(bi)<<4 | b[2]&0x0C | pad<<1
	return b
}
