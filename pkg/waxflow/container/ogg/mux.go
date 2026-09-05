package ogg

import (
	"encoding/binary"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

// Muxer writes an Ogg stream: header pages (identification, comment), then audio pages batching multiple packets each.
type Muxer struct {
	w       io.Writer
	serial  uint32
	seq     uint32
	vendor  string
	tags    []container.Tag
	mapping muxMapping

	granule        int64
	pending        []byte
	pendingSeg     []byte
	pendingGranule int64
	pendingDur     int64
	audioPages     int
	begun          bool
}

const (
	pageTargetBytes   = 4096
	maxPageGranules   = 48000
	maxPageSegEntries = 255
)

// MuxerOptions configures the Ogg muxer.
type MuxerOptions struct {
	Vendor string
	Serial uint32
	Tags   []container.Tag
}

const oggOpusSerial = 0x4F707573

// NewMuxer returns an Ogg muxer writing to w.
func NewMuxer(w io.Writer, opts *MuxerOptions) *Muxer {
	m := &Muxer{w: w, serial: oggOpusSerial, vendor: "WaxFlow"}
	if opts != nil {
		if opts.Vendor != "" {
			m.vendor = opts.Vendor
		}
		if opts.Serial != 0 {
			m.serial = opts.Serial
		}
		m.tags = opts.Tags
	}
	return m
}

// NeedsSeek is false: Ogg-Opus writes a compliant streaming form.
func (m *Muxer) NeedsSeek() bool { return false }

// Begin selects the codec's mapping and writes its header pages.
func (m *Muxer) Begin(tracks []container.Track) error {
	if len(tracks) != 1 {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "ogg: muxer needs a single track")
	}
	m.mapping = muxMappingFor(tracks[0].Codec)
	if m.mapping == nil {
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("ogg: cannot mux codec %q (opus, flac, vorbis)", tracks[0].Codec))
	}
	m.begun = true
	return m.mapping.writeHeaders(tracks[0].CodecConfig, m.tags, m.vendor, m.emitPage)
}

const maxTagsPageBytes = 48 << 10

// WritePacket adds a packet to the page being batched, first flushing that page when the packet would not fit (segment table, byte target, or duration cap) or when it is the stream's first audio page, which stays a single packet so audio reaches the client right after the headers.
func (m *Muxer) WritePacket(pkt container.Packet) error {
	if !m.begun {
		return waxerr.New(waxerr.CodeInternal, "ogg: WritePacket before Begin")
	}
	inc, err := m.mapping.writePacket(pkt)
	if err != nil {
		return err
	}
	segs := len(pkt.Data)/255 + 1
	if len(m.pending) > 0 &&
		(m.audioPages == 0 ||
			len(m.pendingSeg)+segs > maxPageSegEntries ||
			len(m.pending)+len(pkt.Data) > pageTargetBytes ||
			m.pendingDur+inc > maxPageGranules) {
		if err := m.flushPending(0); err != nil {
			return err
		}
	}
	m.granule += inc
	m.pending = append(m.pending, pkt.Data...)
	m.pendingSeg = appendLacing(m.pendingSeg, len(pkt.Data))
	m.pendingGranule = m.granule
	m.pendingDur += inc
	return nil
}

func (m *Muxer) flushPending(headerType byte) error {
	err := m.emitPage(m.pending, m.pendingSeg, m.pendingGranule, headerType)
	m.pending = m.pending[:0]
	m.pendingSeg = m.pendingSeg[:0]
	m.pendingDur = 0
	m.audioPages++
	return err
}

// End writes the final page with the mapping's end granule and the EOS flag.
func (m *Muxer) End(trailer codec.Trailer) error {
	if !m.begun {
		return waxerr.New(waxerr.CodeInternal, "ogg: End before Begin")
	}
	endGranule := m.mapping.endGranule(trailer)
	if len(m.pending) == 0 {
		return m.emitPage(nil, lacing(0), endGranule, flagEOS)
	}
	m.pendingGranule = endGranule
	return m.flushPending(flagEOS)
}

func (m *Muxer) emitPage(payload, seg []byte, granule int64, headerType byte) error {
	header := make([]byte, headerLen+len(seg))
	copy(header, "OggS")
	header[4] = 0
	header[5] = headerType
	binary.LittleEndian.PutUint64(header[6:], uint64(granule))
	binary.LittleEndian.PutUint32(header[14:], m.serial)
	binary.LittleEndian.PutUint32(header[18:], m.seq)
	header[26] = byte(len(seg))
	copy(header[headerLen:], seg)
	crc := crc32(0, header)
	crc = crc32(crc, payload)
	binary.LittleEndian.PutUint32(header[22:], crc)
	m.seq++
	if _, err := m.w.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := m.w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

func lacing(n int) []byte {
	return appendLacing(make([]byte, 0, n/255+1), n)
}

func appendLacing(seg []byte, n int) []byte {
	for n >= 255 {
		seg = append(seg, 255)
		n -= 255
	}
	return append(seg, byte(n))
}
