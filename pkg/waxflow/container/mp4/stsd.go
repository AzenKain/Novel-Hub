package mp4

import (
	"encoding/binary"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/codec/alac"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/codec/opus"
)

func (d *Demuxer) parseStsd(t *track, payload []byte, depth int) error {
	if depth > maxDepth {
		return malformed("box nesting deeper than %d", maxDepth)
	}
	_, _, rest, ok := fullBox(payload)
	if !ok || len(rest) < 4 {
		return malformed("stsd truncated")
	}
	entries := be32(rest)
	rest = rest[4:]
	if entries == 0 {
		return malformed("stsd has no sample entries")
	}
	done := false
	return walkBoxes(rest, func(format string, body []byte) error {
		if done {
			return nil
		}
		done = true
		return d.parseAudioSampleEntry(t, format, body, depth+1)
	})
}

func (d *Demuxer) parseAudioSampleEntry(t *track, format string, body []byte, depth int) error {
	if len(body) < 28 {
		return malformed("audio sample entry %q truncated", format)
	}
	version := be16(body[8:10])
	channels := int(be16(body[16:18]))
	bitsPerSample := int(be16(body[18:20]))
	sampleRate := int(be32(body[24:28]) >> 16)

	childOff := 28
	switch version {
	case 1:
		childOff = 28 + 16
	case 2:
		childOff = 28 + 36
	}
	if childOff > len(body) {
		childOff = len(body)
	}
	children := body[childOff:]

	if wave := findChild(children, "wave"); wave != nil {
		children = wave
	}

	switch format {
	case "alac":
		return d.setALAC(t, format, children, sampleRate, channels, bitsPerSample)
	case "mp4a":
		return d.setMP4A(t, children, sampleRate, channels)
	case "Opus":
		return d.setOpus(t, children)
	case "fLaC":
		return d.setFLAC(t, children)
	default:
		t.codec = codec.ID(format)
		return nil
	}
}

func (d *Demuxer) setOpus(t *track, children []byte) error {
	dops := findChild(children, "dOps")
	if dops == nil {
		return malformed("Opus sample entry has no dOps box")
	}
	if len(dops) < 11 {
		return malformed("dOps box truncated (%d bytes)", len(dops))
	}
	if family := dops[10]; family != 0 {
		return malformed("Opus channel mapping family %d unsupported", family)
	}
	head := make([]byte, 19)
	copy(head, "OpusHead")
	head[8] = 1
	head[9] = dops[1]
	binary.LittleEndian.PutUint16(head[10:], be16(dops[2:]))
	binary.LittleEndian.PutUint32(head[12:], be32(dops[4:]))
	binary.LittleEndian.PutUint16(head[16:], be16(dops[8:]))
	head[18] = 0
	cfg, err := opus.ParseOpusHead(head)
	if err != nil {
		return err
	}
	t.codec = codec.Opus
	t.codecConfig = head
	t.fmt = cfg.Format()
	return nil
}

func (d *Demuxer) setFLAC(t *track, children []byte) error {
	dfla := findChild(children, "dfLa")
	if dfla == nil {
		return malformed("FLAC sample entry has no dfLa box")
	}
	_, _, rest, ok := fullBox(dfla)
	if !ok || len(rest) < 4+flac.StreamInfoLen {
		return malformed("dfLa box truncated")
	}
	if typ := rest[0] & 0x7F; typ != 0 {
		return malformed("dfLa first metadata block is type %d, want STREAMINFO", typ)
	}
	si := append([]byte(nil), rest[4:4+flac.StreamInfoLen]...)
	parsed, err := flac.ParseStreamInfo(si)
	if err != nil {
		return err
	}
	t.codec = codec.FLAC
	t.codecConfig = si
	t.fmt = parsed.PCMFormat()
	return nil
}

func (d *Demuxer) setALAC(t *track, format string, children []byte, rate, channels, bits int) error {
	ext := findChild(children, "alac")
	if ext == nil {
		return malformed("ALAC sample entry has no magic cookie")
	}
	cookie := ext
	if len(ext) >= 28 {
		cookie = ext[4:]
	}
	cfg, err := alac.ParseMagicCookie(cookie)
	if err != nil {
		return err
	}
	t.codec = codec.ALAC
	t.codecConfig = append([]byte(nil), cfg.Cookie...)
	t.fmt = cfg.Format()
	return nil
}

func (d *Demuxer) setMP4A(t *track, children []byte, rate, channels int) error {
	esds := findChild(children, "esds")
	if esds == nil {
		return malformed("mp4a sample entry has no esds")
	}
	asc, objType, err := parseESDS(esds)
	if err != nil {
		return err
	}
	if !isAACObjectType(objType) {
		t.codec = codec.ID(objectTypeName(objType))
		return nil
	}
	if len(asc) == 0 {
		return malformed("mp4a esds carries no AudioSpecificConfig")
	}
	cfg, err := aac.ParseASC(asc)
	if err != nil {
		return err
	}
	if cfg.Channels == 0 {
		cfg.Channels = channels
	}
	f, err := cfg.Format()
	if err != nil {
		return err
	}
	t.note = cfg.SBRWarning()
	t.codec = codec.AACLC
	t.codecConfig = append([]byte(nil), asc...)
	t.fmt = f
	return nil
}

func findChild(body []byte, typ string) []byte {
	var found []byte
	_ = walkBoxes(body, func(t string, payload []byte) error {
		if found == nil && t == typ {
			found = payload
		}
		return nil
	})
	return found
}

func parseESDS(payload []byte) (asc []byte, objType byte, err error) {
	_, _, rest, ok := fullBox(payload)
	if !ok {
		return nil, 0, malformed("esds truncated")
	}
	tag, body, _, ok := readDescriptor(rest)
	if !ok || tag != tagES {
		return nil, 0, malformed("esds has no ES_Descriptor")
	}
	if len(body) < 3 {
		return nil, 0, malformed("ES_Descriptor truncated")
	}
	flags := body[2]
	p := body[3:]
	if flags&0x80 != 0 {
		if len(p) < 2 {
			return nil, 0, malformed("ES_Descriptor truncated")
		}
		p = p[2:]
	}
	if flags&0x40 != 0 {
		if len(p) < 1 || len(p) < 1+int(p[0]) {
			return nil, 0, malformed("ES_Descriptor URL truncated")
		}
		p = p[1+int(p[0]):]
	}
	if flags&0x20 != 0 {
		if len(p) < 2 {
			return nil, 0, malformed("ES_Descriptor truncated")
		}
		p = p[2:]
	}
	for len(p) >= 2 {
		dt, dbody, drest, ok := readDescriptor(p)
		if !ok {
			break
		}
		if dt == tagDecoderConfig {
			if len(dbody) < 13 {
				return nil, 0, malformed("DecoderConfigDescriptor truncated")
			}
			objType = dbody[0]
			q := dbody[13:]
			for len(q) >= 2 {
				st, sbody, srest, ok := readDescriptor(q)
				if !ok {
					break
				}
				if st == tagDecoderSpecific {
					asc = sbody
				}
				q = srest
			}
		}
		p = drest
	}
	return asc, objType, nil
}

const (
	tagES              = 0x03
	tagDecoderConfig   = 0x04
	tagDecoderSpecific = 0x05
)

func readDescriptor(b []byte) (tag byte, body, rest []byte, ok bool) {
	if len(b) < 2 {
		return 0, nil, nil, false
	}
	tag = b[0]
	i := 1
	length := 0
	for n := 0; n < 4; n++ {
		if i >= len(b) {
			return 0, nil, nil, false
		}
		c := b[i]
		i++
		length = length<<7 | int(c&0x7F)
		if c&0x80 == 0 {
			break
		}
	}
	if length > maxDescriptorLen {
		length = maxDescriptorLen
	}
	if i+length > len(b) {
		length = len(b) - i
	}
	return tag, b[i : i+length], b[i+length:], true
}

func isAACObjectType(ot byte) bool {
	switch ot {
	case 0x40, 0x66, 0x67, 0x68:
		return true
	}
	return false
}

func objectTypeName(ot byte) string {
	switch ot {
	case 0x69, 0x6B:
		return "mp3"
	case 0xA9:
		return "dts"
	case 0xA5, 0xA6:
		return "ac3"
	default:
		return "unknown"
	}
}
