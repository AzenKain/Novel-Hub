package mp4

import (
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
)

type track struct {
	id        int
	handler   string
	timescale int64
	duration  int64

	codec       codec.ID
	codecConfig []byte
	fmt         audio.Format

	note string

	stsdErr error

	st sampleTable

	hasEdit    bool
	editMedia  int64
	editSegDur int64

	emptyEdit int64

	chapRefs []int
}

func (d *Demuxer) parseMoov(moov []byte) ([]*track, error) {
	_ = walkBoxes(moov, func(typ string, payload []byte) error {
		if typ == "mvex" {
			d.parseMvex(payload)
		}
		return nil
	})
	var tracks []*track
	err := walkBoxes(moov, func(typ string, payload []byte) error {
		switch typ {
		case "mvhd":
			d.movieTimescale = mvhdTimescale(payload)
		case "trak":
			if len(tracks) >= maxTracks {
				return malformed("more than %d tracks", maxTracks)
			}
			t := &track{editMedia: -1}
			if err := d.parseTrak(t, payload, 1); err != nil {
				return err
			}
			tracks = append(tracks, t)
		case "udta":
			return d.parseUdta(payload, 1)
		}
		return nil
	})
	return tracks, err
}

func mvhdTimescale(payload []byte) int64 {
	version, _, rest, ok := fullBox(payload)
	if !ok {
		return 0
	}
	off := 8
	if version == 1 {
		off = 16
	}
	if len(rest) < off+4 {
		return 0
	}
	return int64(be32(rest[off:]))
}

func (d *Demuxer) parseTrak(t *track, body []byte, depth int) error {
	if depth > maxDepth {
		return malformed("box nesting deeper than %d", maxDepth)
	}
	return walkBoxes(body, func(typ string, payload []byte) error {
		switch typ {
		case "tkhd":
			t.id = tkhdTrackID(payload)
		case "edts":
			return walkBoxes(payload, func(t2 string, p2 []byte) error {
				if t2 == "elst" {
					parseElst(t, p2)
				}
				return nil
			})
		case "tref":
			parseTref(t, payload)
		case "mdia":
			return d.parseMdia(t, payload, depth+1)
		}
		return nil
	})
}

func tkhdTrackID(payload []byte) int {
	version, _, rest, ok := fullBox(payload)
	if !ok {
		return 0
	}
	off := 8
	if version == 1 {
		off = 16
	}
	if len(rest) < off+4 {
		return 0
	}
	return int(be32(rest[off:]))
}

func (d *Demuxer) parseMdia(t *track, body []byte, depth int) error {
	if depth > maxDepth {
		return malformed("box nesting deeper than %d", maxDepth)
	}
	return walkBoxes(body, func(typ string, payload []byte) error {
		switch typ {
		case "mdhd":
			t.timescale, t.duration = mdhdTime(payload)
		case "hdlr":
			t.handler = hdlrType(payload)
		case "minf":
			return d.parseMinf(t, payload, depth+1)
		}
		return nil
	})
}

func mdhdTime(payload []byte) (timescale, duration int64) {
	version, _, rest, ok := fullBox(payload)
	if !ok {
		return 0, 0
	}
	if version == 1 {
		if len(rest) < 28 {
			return 0, 0
		}
		return int64(be32(rest[16:])), int64(be64(rest[20:]))
	}
	if len(rest) < 16 {
		return 0, 0
	}
	return int64(be32(rest[8:])), int64(be32(rest[12:]))
}

func hdlrType(payload []byte) string {
	_, _, rest, ok := fullBox(payload)
	if !ok || len(rest) < 8 {
		return ""
	}
	return trimBrand(rest[4:8])
}

func (d *Demuxer) parseMinf(t *track, body []byte, depth int) error {
	if depth > maxDepth {
		return malformed("box nesting deeper than %d", maxDepth)
	}
	return walkBoxes(body, func(typ string, payload []byte) error {
		if typ == "stbl" {
			return d.parseStbl(t, payload, depth+1)
		}
		return nil
	})
}

func parseElst(t *track, payload []byte) {
	version, _, rest, ok := fullBox(payload)
	if !ok || len(rest) < 4 {
		return
	}
	count := int64(be32(rest))
	rest = rest[4:]
	entrySize := int64(12)
	if version == 1 {
		entrySize = 20
	}
	if count > int64(len(rest))/entrySize {
		count = int64(len(rest)) / entrySize
	}
	var totalSeg, empty int64
	mediaTime := int64(-1)
	for i := int64(0); i < count; i++ {
		e := rest[i*entrySize:]
		var segDur, mt int64
		if version == 1 {
			segDur = int64(be64(e[0:]))
			mt = int64(be64(e[8:]))
		} else {
			segDur = int64(be32(e[0:]))
			mt = int64(int32(be32(e[4:])))
		}
		if mt < 0 {
			if mediaTime < 0 && segDur > 0 {
				empty += segDur
			}
			continue
		}
		if mediaTime < 0 {
			mediaTime = mt
		}
		totalSeg += segDur
	}
	t.emptyEdit = empty
	if mediaTime < 0 {
		return
	}
	t.hasEdit = true
	t.editMedia = mediaTime
	t.editSegDur = totalSeg
}

func parseTref(t *track, body []byte) {
	_ = walkBoxes(body, func(typ string, payload []byte) error {
		if typ == "chap" {
			for i := 0; i+4 <= len(payload); i += 4 {
				t.chapRefs = append(t.chapRefs, int(be32(payload[i:])))
			}
		}
		return nil
	})
}
