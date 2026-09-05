package ogg

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/codec/vorbis"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

type muxMapping interface {
	codecID() codec.ID
	writeHeaders(cfg []byte, tags []container.Tag, vendor string,
		emit func(payload, seg []byte, granule int64, headerType byte) error) error
	writePacket(pkt container.Packet) (granuleIncrement int64, err error)
	endGranule(trailer codec.Trailer) int64
}

func muxMappingFor(id codec.ID) muxMapping {
	switch id {
	case codec.Opus:
		return opusMuxMapping{}
	case codec.FLAC:
		return flacMuxMapping{}
	case codec.Vorbis:
		return &vorbisMuxMapping{}
	}
	return nil
}

// opusMuxMapping writes the Ogg-Opus mapping (RFC 7845): an OpusHead BOS page and an OpusTags comment page, then audio pages whose granule is the running 48 kHz sample count including pre-skip.
type opusMuxMapping struct{}

func (opusMuxMapping) codecID() codec.ID { return codec.Opus }

func (opusMuxMapping) writeHeaders(cfg []byte, tags []container.Tag, vendor string,
	emit func(payload, seg []byte, granule int64, headerType byte) error) error {
	if len(cfg) < 19 || string(cfg[:8]) != "OpusHead" {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "ogg: track CodecConfig is not an OpusHead")
	}
	if err := emit(cfg, lacing(len(cfg)), 0, flagBOS); err != nil {
		return err
	}
	comment := buildComment("OpusTags", vendor, tags)
	return emit(comment, lacing(len(comment)), 0, 0)
}

func (opusMuxMapping) writePacket(pkt container.Packet) (int64, error) { return pkt.Dur, nil }

// endGranule is the pre-skip plus the true length: the final page granule the decoder clamps its output to (RFC 7845 section 4.4).
func (opusMuxMapping) endGranule(trailer codec.Trailer) int64 {
	return trailer.Delay + trailer.Samples
}

type flacMuxMapping struct{}

func (flacMuxMapping) codecID() codec.ID { return codec.FLAC }

func (flacMuxMapping) writeHeaders(cfg []byte, tags []container.Tag, vendor string,
	emit func(payload, seg []byte, granule int64, headerType byte) error) error {
	if len(cfg) != flac.StreamInfoLen {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "ogg: FLAC CodecConfig is not a STREAMINFO block")
	}
	id := []byte{0x7F, 'F', 'L', 'A', 'C', 1, 0, 0, 1, 'f', 'L', 'a', 'C',
		0x00, 0x00, 0x00, byte(flac.StreamInfoLen)}
	id = append(id, cfg...)
	if err := emit(id, lacing(len(id)), 0, flagBOS); err != nil {
		return err
	}
	body := buildComment("", vendor, tags)
	comment := append([]byte{0x84, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}, body...)
	return emit(comment, lacing(len(comment)), 0, 0)
}

func (flacMuxMapping) writePacket(pkt container.Packet) (int64, error) { return pkt.Dur, nil }

func (flacMuxMapping) endGranule(trailer codec.Trailer) int64 { return trailer.Samples }

type vorbisMuxMapping struct {
	cfg       vorbis.Config
	modeBits  int
	haveCfg   bool
	prevBlock int
}

func (m *vorbisMuxMapping) codecID() codec.ID { return codec.Vorbis }

func (m *vorbisMuxMapping) writeHeaders(cfg []byte, tags []container.Tag, vendor string,
	emit func(payload, seg []byte, granule int64, headerType byte) error) error {
	id, _, setup, err := vorbis.SplitConfig(cfg)
	if err != nil {
		return err
	}
	if len(id) < 7 || id[0] != 0x01 || string(id[1:7]) != "vorbis" {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "ogg: track CodecConfig is not a Vorbis identification header")
	}
	c, err := vorbis.ParseConfig(cfg)
	if err != nil {
		return err
	}
	m.cfg = c
	m.modeBits = vorbis.ModeBits(c)
	m.haveCfg = true

	if err := emit(id, lacing(len(id)), 0, flagBOS); err != nil {
		return err
	}
	comment := append([]byte{0x03, 'v', 'o', 'r', 'b', 'i', 's'}, buildComment("", vendor, tags)...)
	comment = append(comment, 0x01)
	return emitHeaderPages([][]byte{comment, setup}, emit)
}

func (m *vorbisMuxMapping) writePacket(pkt container.Packet) (int64, error) {
	if !m.haveCfg {
		return 0, waxerr.New(waxerr.CodeInternal, "ogg: vorbis writePacket before writeHeaders")
	}
	block, ok := vorbis.PacketBlockSize(m.cfg, m.modeBits, pkt.Data)
	if !ok {
		return 0, waxerr.New(waxerr.CodeUnsupportedFormat, "ogg: not a valid Vorbis audio packet")
	}
	var inc int64
	if m.prevBlock != 0 {
		inc = int64(m.prevBlock+block) / 4
	}
	m.prevBlock = block
	return inc, nil
}

func (m *vorbisMuxMapping) endGranule(trailer codec.Trailer) int64 {
	return trailer.Samples
}

func emitHeaderPages(pkts [][]byte,
	emit func(payload, seg []byte, granule int64, headerType byte) error) error {
	var body, laces []byte
	for _, pkt := range pkts {
		body = append(body, pkt...)
		laces = appendLacing(laces, len(pkt))
	}
	byteOff := 0
	for i := 0; i < len(laces); {
		end := min(i+maxPageSegEntries, len(laces))
		seg := laces[i:end]
		n := 0
		for _, l := range seg {
			n += int(l)
		}
		flags := byte(0)
		if i > 0 && laces[i-1] == 255 {
			flags = flagContinued
		}
		if err := emit(body[byteOff:byteOff+n], seg, 0, flags); err != nil {
			return err
		}
		byteOff += n
		i = end
	}
	return nil
}

func buildComment(magic, vendor string, tags []container.Tag) []byte {
	out := make([]byte, 0, len(magic)+4+len(vendor)+4)
	out = append(out, magic...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(vendor)))
	out = append(out, vendor...)
	countAt := len(out)
	out = binary.LittleEndian.AppendUint32(out, 0)
	count := uint32(0)
	for _, t := range tags {
		if !container.ValidTagKey(t.Key) {
			continue
		}
		c := t.Key + "=" + t.Value
		if len(out)+4+len(c) > maxTagsPageBytes {
			continue
		}
		out = binary.LittleEndian.AppendUint32(out, uint32(len(c)))
		out = append(out, c...)
		count++
	}
	binary.LittleEndian.PutUint32(out[countAt:], count)
	return out
}
