package mp4

import (
	"math"
	"math/bits"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"novelhub/pkg/waxflow/container"
)

func (d *Demuxer) gapless(t *track) (delay, padding, samples int64) {
	totalRaw := t.st.totalDur

	if d.smpbOK {
		delay = clamp(d.smpbDelay, 0, totalRaw)
		samples = d.smpbTotal
		if samples < 0 || samples > totalRaw-delay {
			samples = totalRaw - delay
		}
		padding = totalRaw - delay - samples
		return delay, padding, samples
	}

	if t.hasEdit && t.editMedia > 0 {
		delay, seg, haveSeg := editListTrims(t, d.movieTimescale)
		delay = clamp(delay, 0, totalRaw)
		if haveSeg {
			samples = seg
		} else {
			samples = totalRaw - delay
		}
		if samples < 0 || samples > totalRaw-delay {
			samples = totalRaw - delay
		}
		padding = totalRaw - delay - samples
		return delay, padding, samples
	}

	return 0, 0, totalRaw
}

func editListTrims(t *track, movieTimescale int64) (delay, segSamples int64, haveSeg bool) {
	rate := int64(t.fmt.Rate)
	delay = t.editMedia
	if t.timescale > 0 && rate > 0 && t.timescale != rate {
		delay = mulDivSat(t.editMedia, rate, t.timescale)
	}
	if delay < 0 {
		delay = 0
	}
	if t.editSegDur > 0 && movieTimescale > 0 && rate > 0 {
		return delay, mulDivSat(t.editSegDur, rate, movieTimescale), true
	}
	return delay, 0, false
}

func clamp(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mulDivSat(a, b, c int64) int64 {
	if a <= 0 || b <= 0 || c <= 0 {
		return 0
	}
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	if hi >= uint64(c) {
		return math.MaxInt64
	}
	q, _ := bits.Div64(hi, lo, uint64(c))
	if q > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(q)
}

func mulDivRound(a, b, c int64) int64 {
	if a <= 0 || b <= 0 || c <= 0 {
		return 0
	}
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	lo, carry := bits.Add64(lo, uint64(c)/2, 0)
	hi += carry
	if hi >= uint64(c) {
		return math.MaxInt64
	}
	q, _ := bits.Div64(hi, lo, uint64(c))
	if q > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(q)
}

func (d *Demuxer) parseUdta(body []byte, depth int) error {
	if depth > maxDepth {
		return malformed("box nesting deeper than %d", maxDepth)
	}
	return walkBoxes(body, func(typ string, payload []byte) error {
		switch typ {
		case "chpl":
			d.parseChpl(payload)
		case "meta":
			d.parseMeta(payload, depth+1)
		}
		return nil
	})
}

func (d *Demuxer) parseMeta(payload []byte, depth int) {
	if depth > maxDepth {
		return
	}
	body := payload
	switch {
	case len(payload) >= 12 && string(payload[8:12]) == "hdlr":
		body = payload[4:]
	case len(payload) >= 8 && string(payload[4:8]) == "hdlr":
		body = payload
	case len(payload) >= 4:
		body = payload[4:]
	}
	_ = walkBoxes(body, func(typ string, p []byte) error {
		if typ == "ilst" {
			d.parseILST(p)
		}
		return nil
	})
}

var ilstTextKeys = map[string]string{
	"\xa9nam": "TITLE",
	"\xa9ART": "ARTIST",
	"\xa9alb": "ALBUM",
	"aART":    "ALBUMARTIST",
	"\xa9wrt": "COMPOSER",
	"\xa9gen": "GENRE",
	"\xa9day": "RECORDINGDATE",
	"\xa9cmt": "COMMENT",
	"\xa9lyr": "LYRICS",
}

var ilstFreeformKeys = map[string]bool{
	"REPLAYGAIN_TRACK_GAIN": true,
	"REPLAYGAIN_TRACK_PEAK": true,
	"REPLAYGAIN_ALBUM_GAIN": true,
	"REPLAYGAIN_ALBUM_PEAK": true,
}

func (d *Demuxer) parseILST(body []byte) {
	_ = walkBoxes(body, func(typ string, payload []byte) error {
		switch {
		case typ == "----":
			d.parseFreeform(payload)
		case typ == "trkn":
			d.addNumberPair(payload, "TRACKNUMBER", "TRACKTOTAL")
		case typ == "disk":
			d.addNumberPair(payload, "DISCNUMBER", "DISCTOTAL")
		default:
			if key, ok := ilstTextKeys[typ]; ok {
				if v, ok := itemText(payload); ok {
					d.addTag(key, v)
				}
			}
		}
		return nil
	})
}

func (d *Demuxer) addTag(key, value string) {
	if value == "" {
		return
	}
	if d.tags == nil {
		d.tags = make(map[string][]string)
	}
	d.tags[key] = append(d.tags[key], value)
}

func itemText(body []byte) (string, bool) {
	var out string
	var ok bool
	_ = walkBoxes(body, func(typ string, p []byte) error {
		if typ == "data" && !ok && len(p) >= 8 && be32(p) == 1 {
			if s := strings.TrimRight(string(p[8:]), "\x00"); utf8.ValidString(s) {
				out, ok = s, true
			}
		}
		return nil
	})
	return out, ok
}

func (d *Demuxer) addNumberPair(body []byte, numKey, totalKey string) {
	_ = walkBoxes(body, func(typ string, p []byte) error {
		if typ != "data" || len(p) < 14 || be32(p) != 0 {
			return nil
		}
		payload := p[8:]
		if n := be16(payload[2:]); n != 0 {
			d.addTag(numKey, strconv.Itoa(int(n)))
		}
		if t := be16(payload[4:]); t != 0 {
			d.addTag(totalKey, strconv.Itoa(int(t)))
		}
		return nil
	})
}

func (d *Demuxer) parseFreeform(body []byte) {
	var name string
	var data []byte
	_ = walkBoxes(body, func(typ string, p []byte) error {
		switch typ {
		case "name":
			if _, _, rest, ok := fullBox(p); ok {
				name = string(rest)
			}
		case "data":
			if len(p) >= 8 {
				data = p[8:]
			}
		}
		return nil
	})
	if data == nil {
		return
	}
	if name == "iTunSMPB" {
		d.parseSMPB(string(data))
		return
	}
	if ilstFreeformKeys[name] {
		if s := strings.TrimRight(string(data), "\x00"); utf8.ValidString(s) {
			d.addTag(name, s)
		}
	}
}

func (d *Demuxer) parseSMPB(s string) {
	fields := strings.Fields(s)
	if len(fields) < 4 {
		return
	}
	delay, e1 := strconv.ParseInt(fields[1], 16, 64)
	padding, e2 := strconv.ParseInt(fields[2], 16, 64)
	total, e3 := strconv.ParseInt(fields[3], 16, 64)
	if e1 != nil || e2 != nil || e3 != nil || delay < 0 || padding < 0 || total < 0 {
		return
	}
	d.smpbDelay, d.smpbTotal, d.smpbOK = delay, total, true
}

func (d *Demuxer) parseChpl(payload []byte) {
	version, _, rest, ok := fullBox(payload)
	if !ok {
		return
	}
	if version == 1 && len(rest) >= 4 {
		rest = rest[4:]
	}
	if len(rest) < 1 {
		return
	}
	count := int(rest[0])
	rest = rest[1:]
	for i := 0; i < count && i < maxChapters; i++ {
		if len(rest) < 9 {
			return
		}
		start := int64(be64(rest))
		titleLen := int(rest[8])
		rest = rest[9:]
		if titleLen > len(rest) {
			titleLen = len(rest)
		}
		d.chplChapters = append(d.chplChapters, Chapter{
			Start: time.Duration(start) * 100,
			Title: sanitizeTitle(rest[:titleLen]),
		})
		rest = rest[titleLen:]
	}
}

func (d *Demuxer) resolveChapters(tracks []*track, audio *track) {
	if ct := d.chapterTrack(tracks, audio); ct != nil {
		if chapters := d.readTextChapters(ct); len(chapters) > 0 {
			d.chapters = chapters
			return
		}
	}
	d.chapters = d.chplChapters
}

func (d *Demuxer) chapterTrack(tracks []*track, audio *track) *track {
	byID := func(id int) *track {
		for _, t := range tracks {
			if t.id == id && (t.handler == "text" || t.handler == "sbtl") {
				return t
			}
		}
		return nil
	}
	for _, id := range audio.chapRefs {
		if t := byID(id); t != nil && t.st.total > 0 {
			return t
		}
	}
	for _, t := range tracks {
		if (t.handler == "text" || t.handler == "sbtl") && t.st.total > 0 {
			return t
		}
	}
	return nil
}

func (d *Demuxer) readTextChapters(ct *track) []Chapter {
	st := &ct.st
	n := st.total
	if n > maxChapters {
		n = maxChapters
	}
	shift := mulDivRound(ct.emptyEdit, ct.timescale, d.movieTimescale)
	var out []Chapter
	for i := int64(0); i < n; i++ {
		size := int(st.sizes[i])
		if size < 2 || size > 1<<16 {
			continue
		}
		buf := make([]byte, size)
		if container.ReadFull(d.src, buf, st.offsets[i]) != nil {
			break
		}
		textLen := int(be16(buf))
		if 2+textLen > len(buf) {
			textLen = len(buf) - 2
		}
		pts, dur := st.timeOf(i)
		start := time.Duration(mulDivSat(pts+shift, int64(time.Second), ct.timescale))
		end := time.Duration(0)
		if dur > 0 {
			end = time.Duration(mulDivSat(pts+shift+dur, int64(time.Second), ct.timescale))
		}
		out = append(out, Chapter{Start: start, End: end, Title: sanitizeTitle(buf[2 : 2+textLen])})
	}
	return out
}

func sanitizeTitle(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		return decodeUTF16BE(b[2:])
	}
	s := strings.TrimRight(string(b), "\x00")
	return strings.TrimSpace(s)
}

func decodeUTF16BE(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, be16(b[i:]))
	}
	s := strings.TrimRight(string(utf16.Decode(u)), "\x00")
	return strings.TrimSpace(s)
}
