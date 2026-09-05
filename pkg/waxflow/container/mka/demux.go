package mka

import (
	"fmt"
	"sort"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/internal/srcwin"
	"novelhub/pkg/waxflow/waxerr"
)

var (
	_ container.Demuxer = (*Demuxer)(nil)
	_ container.Seeker  = (*Demuxer)(nil)
	_ container.Warner  = (*Demuxer)(nil)
)

// DemuxerOptions configures parsing.
type DemuxerOptions struct {
	Strict bool
}

// Demuxer reads one audio track from a Matroska/WebM segment.
type Demuxer struct {
	src  container.Source
	opts DemuxerOptions
	size int64

	segmentDataOff int64
	segmentEnd     int64
	timestampScale int64
	durationTicks  float64

	entries       []*trackEntry
	seekPositions map[uint32]int64

	sel   *trackEntry
	setup codecSetup
	track container.Track

	seekPreRollSamples int64

	firstClusterOff  int64
	haveFirstCluster bool

	clusterIndex []clusterPos
	walked       bool
	rawTotal     int64
	paddingNS    int64
	recording    bool

	walkCumulative int64
	walkPaddingNS  int64
	walkFrames     int
	walkLimit      int64
	walkedTo       int64
	walkStopped    bool

	cues         []cueEntry
	cuesResolved bool

	w              srcwin.Window
	curOff         int64
	inCluster      bool
	clusterEnd     int64
	clusterCursor  int64
	clusterUnknown bool

	pending           []frameLoc
	pendingIdx        int
	running           int64
	curBlockDiscardNS int64

	vorbisPrevBlock int

	warnings              []container.Warning
	warnedNegativeDiscard bool
}

// NewDemuxer parses the segment header and positions on the first cluster.
func NewDemuxer(src container.Source, opts *DemuxerOptions) (*Demuxer, error) {
	d := &Demuxer{
		src:            src,
		size:           src.Size(),
		timestampScale: defaultTimestampScale,
		seekPositions:  map[uint32]int64{},
		w:              srcwin.New(src, src.Size(), "mka: reading block data"),
		walkLimit:      -1,
	}
	if opts != nil {
		d.opts = *opts
	}
	if err := d.parse(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Demuxer) warn(off int64, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if d.opts.Strict {
		return malformed("%s (at offset %d)", msg, off)
	}
	d.warnings = append(d.warnings, container.Warning{Offset: off, Msg: msg})
	return nil
}

func (d *Demuxer) note(off int64, format string, args ...any) {
	d.warnings = append(d.warnings, container.Warning{Offset: off, Msg: fmt.Sprintf(format, args...)})
}

func (d *Demuxer) readBytes(off, n, cap int64) ([]byte, error) {
	if n < 0 || n > cap {
		return nil, malformed("element of %d bytes exceeds the %d cap", n, cap)
	}
	buf := make([]byte, n)
	if err := container.ReadFull(d.src, buf, off); err != nil {
		return nil, waxerr.Wrap(waxerr.CodeSourceUnreadable, "mka: reading element body", err)
	}
	return buf, nil
}

func (d *Demuxer) parse() error {
	head, err := d.readElement(0, d.size)
	if err != nil {
		return err
	}
	if head.id != idEBML {
		return malformed("file does not begin with an EBML header")
	}
	if head.unknownSize {
		return malformed("EBML header has unknown size")
	}
	if err := d.checkDocType(head); err != nil {
		return err
	}

	seg, err := d.readElement(head.dataEnd(), d.size)
	if err != nil {
		return err
	}
	if seg.id != idSegment {
		return malformed("no Segment after the EBML header")
	}
	d.segmentDataOff = seg.dataOff
	if seg.unknownSize {
		d.segmentEnd = d.size
	} else {
		d.segmentEnd = seg.dataEnd()
	}

	if err := d.scanSegment(); err != nil {
		return err
	}
	if err := d.resolveDeferred(); err != nil {
		return err
	}
	if err := d.selectTrack(); err != nil {
		return err
	}
	if err := d.finalizeTrack(); err != nil {
		return err
	}
	return nil
}

func (d *Demuxer) checkDocType(head element) error {
	if head.size <= 0 || head.size > 1<<16 {
		return nil
	}
	body, err := d.readBytes(head.dataOff, head.size, 1<<16)
	if err != nil {
		return err
	}
	var docType string
	_ = walkElements(body, func(id uint32, data []byte) error {
		if id == idDocType {
			docType = string(data)
		}
		return nil
	})
	if docType != "" && docType != "matroska" && docType != "webm" {
		return d.warn(head.dataOff, "unexpected EBML DocType %q", docType)
	}
	return nil
}

func (d *Demuxer) scanSegment() error {
	off := d.segmentDataOff
	for i := 0; off < d.segmentEnd; i++ {
		if i > maxTopLevelElements {
			return malformed("more than %d segment-level elements", maxTopLevelElements)
		}
		e, err := d.readElement(off, d.segmentEnd)
		if err != nil {
			if d.haveFirstCluster {
				break
			}
			return err
		}
		if e.id == idCluster {
			d.firstClusterOff = e.dataOff - headerLen(e, off)
			d.haveFirstCluster = true
			break
		}
		if e.unknownSize {
			if werr := d.warn(off, "segment element %#x has unknown size", e.id); werr != nil {
				return werr
			}
			break
		}
		if err := d.parseSegmentChild(e); err != nil {
			return err
		}
		off = e.dataEnd()
	}
	return nil
}

func headerLen(e element, off int64) int64 { return e.dataOff - off }

func (d *Demuxer) parseSegmentChild(e element) error {
	switch e.id {
	case idSeekHead:
		body, err := d.readBytes(e.dataOff, e.size, maxHeaderElement)
		if err != nil {
			return err
		}
		d.parseSeekHead(body)
	case idInfo:
		body, err := d.readBytes(e.dataOff, e.size, maxHeaderElement)
		if err != nil {
			return err
		}
		d.parseInfo(body)
	case idTracks:
		body, err := d.readBytes(e.dataOff, e.size, maxHeaderElement)
		if err != nil {
			return err
		}
		if err := d.parseTracks(body); err != nil {
			return err
		}
	}
	return nil
}

func (d *Demuxer) resolveDeferred() error {
	if len(d.entries) == 0 {
		if err := d.readViaSeek(idTracks, maxHeaderElement, d.parseTracksAt); err != nil {
			return err
		}
	}
	if d.timestampScale <= 0 {
		d.timestampScale = defaultTimestampScale
	}
	return nil
}

func (d *Demuxer) readViaSeek(id uint32, cap int64, parse func([]byte) error) error {
	off, ok := d.seekPositions[id]
	if !ok || off < d.segmentDataOff || off >= d.segmentEnd {
		return nil
	}
	e, err := d.readElement(off, d.segmentEnd)
	if err != nil || e.id != id || e.unknownSize {
		return nil
	}
	body, err := d.readBytes(e.dataOff, e.size, cap)
	if err != nil {
		return err
	}
	return parse(body)
}

func (d *Demuxer) parseTracksAt(body []byte) error { return d.parseTracks(body) }

type cueEntry struct {
	time int64
	off  int64
}

func (d *Demuxer) resolveCues() {
	if d.cuesResolved {
		return
	}
	d.cuesResolved = true
	_ = d.readViaSeek(idCues, maxCuesElement, d.parseCues)
}

func (d *Demuxer) parseCues(body []byte) error {
	var out []cueEntry
	seen, stride := 0, 1
	err := walkElements(body, func(id uint32, data []byte) error {
		if id != idCuePoint {
			return nil
		}
		e, ok := d.parseCuePoint(data)
		if !ok {
			return nil
		}
		if seen%stride == 0 {
			if len(out) == maxCuePoints {
				for i := 0; i < maxCuePoints/2; i++ {
					out[i] = out[2*i]
				}
				out = out[:maxCuePoints/2]
				stride *= 2
			}
			out = append(out, e)
		}
		seen++
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].time < out[j].time })
	d.cues = out
	return nil
}

func (d *Demuxer) parseCuePoint(body []byte) (cueEntry, bool) {
	var e cueEntry
	haveTime, havePos := false, false
	_ = walkElements(body, func(id uint32, data []byte) error {
		switch id {
		case idCueTime:
			if v := int64(beUint(data)); v >= 0 {
				e.time, haveTime = v, true
			}
		case idCueTrackPositions:
			if havePos {
				return nil
			}
			var track uint64
			pos := int64(-1)
			haveTrack := false
			_ = walkElements(data, func(cid uint32, cdata []byte) error {
				switch cid {
				case idCueTrack:
					track, haveTrack = beUint(cdata), true
				case idCueClusterPosition:
					pos = int64(beUint(cdata))
				}
				return nil
			})
			if pos >= 0 && (!haveTrack || track == d.sel.number) {
				e.off, havePos = d.segmentDataOff+pos, true
			}
		}
		return nil
	})
	if !haveTime || !havePos || e.off < d.segmentDataOff || e.off >= d.segmentEnd {
		return cueEntry{}, false
	}
	return e, true
}

func (d *Demuxer) parseSeekHead(body []byte) {
	_ = walkElements(body, func(id uint32, data []byte) error {
		if id != idSeek {
			return nil
		}
		var seekID uint32
		var pos int64
		haveID, havePos := false, false
		_ = walkElements(data, func(cid uint32, cdata []byte) error {
			switch cid {
			case idSeekID:
				seekID = uint32(beUint(cdata))
				haveID = true
			case idSeekPosition:
				pos = int64(beUint(cdata))
				havePos = true
			}
			return nil
		})
		if haveID && havePos {
			if _, seen := d.seekPositions[seekID]; !seen {
				d.seekPositions[seekID] = d.segmentDataOff + pos
			}
		}
		return nil
	})
}

func (d *Demuxer) parseInfo(body []byte) {
	_ = walkElements(body, func(id uint32, data []byte) error {
		switch id {
		case idTimestampScale:
			if v := int64(beUint(data)); v > 0 {
				d.timestampScale = v
			}
		case idDuration:
			if f, ok := beFloat(data); ok && f > 0 {
				d.durationTicks = f
			}
		}
		return nil
	})
}

func (d *Demuxer) parseTracks(body []byte) error {
	return walkElements(body, func(id uint32, data []byte) error {
		if id != idTrackEntry {
			return nil
		}
		if len(d.entries) >= maxTracks {
			return malformed("more than %d tracks", maxTracks)
		}
		t, err := d.parseTrackEntry(data)
		if err != nil {
			if werr := d.warn(-1, "skipping malformed track: %v", err); werr != nil {
				return werr
			}
			return nil
		}
		d.entries = append(d.entries, t)
		return nil
	})
}

func (d *Demuxer) parseTrackEntry(body []byte) (*trackEntry, error) {
	t := &trackEntry{}
	err := walkElements(body, func(id uint32, data []byte) error {
		switch id {
		case idTrackNumber:
			t.number = beUint(data)
		case idTrackType:
			t.trackType = beUint(data)
		case idCodecID:
			t.codecID = string(data)
		case idCodecPrivate:
			if len(data) > maxCodecPrivate {
				return malformed("CodecPrivate of %d bytes exceeds the %d cap", len(data), maxCodecPrivate)
			}
			t.codecPriv = append([]byte(nil), data...)
		case idCodecDelay:
			t.codecDelay = int64(beUint(data))
		case idSeekPreRoll:
			t.seekPreRoll = int64(beUint(data))
		case idFlagDefault:
			t.def = beUint(data) != 0
		case idAudio:
			parseAudioSettings(t, data)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func parseAudioSettings(t *trackEntry, body []byte) {
	_ = walkElements(body, func(id uint32, data []byte) error {
		switch id {
		case idSamplingFreq:
			if f, ok := beFloat(data); ok && f > 0 {
				t.rate = int(f + 0.5)
			}
		case idChannels:
			t.channels = int(beUint(data))
		case idBitDepth:
			t.bitDepth = int(beUint(data))
		}
		return nil
	})
}

func (d *Demuxer) selectTrack() error {
	var chosen, fallback *trackEntry
	var found []string
	for _, t := range d.entries {
		if t.trackType != trackTypeAudio {
			continue
		}
		if mkvCodecID(t.codecID) == "" {
			found = append(found, codecName(t.codecID))
			continue
		}
		if chosen == nil || (t.def && !chosen.def) {
			chosen = t
		}
		if fallback == nil {
			fallback = t
		}
	}
	if chosen == nil {
		chosen = fallback
	}
	if chosen == nil {
		if len(found) > 0 {
			return malformed("no decodable audio track (found: %s)", joinNames(found))
		}
		return malformed("no audio track")
	}
	setup, err := resolveCodec(chosen)
	if err != nil {
		return err
	}
	if err := setup.fmt.Valid(); err != nil {
		return waxerr.Wrap(waxerr.CodeUnsupportedFormat, "mka: unusable audio format", err)
	}
	if setup.warning != "" {
		d.note(0, "%s", setup.warning)
	}
	d.sel = chosen
	d.setup = setup
	return nil
}

func (d *Demuxer) finalizeTrack() error {
	rate := d.setup.fmt.Rate
	delay := nsToSamples(d.sel.codecDelay, rate)

	d.seekPreRollSamples = nsToSamples(d.sel.seekPreRoll, rate)
	if d.seekPreRollSamples == 0 {
		d.seekPreRollSamples = d.setup.preRoll()
	}

	samples := int64(-1)
	exact := false
	if (d.sel.codecDelay > 0 || d.needsGaplessWalk()) && d.haveFirstCluster {
		if err := d.ensureWalk(); err != nil {
			return err
		}
		padding := nsToSamples(d.paddingNS, rate)
		samples = d.rawTotal - delay - padding
		if samples < 0 {
			samples = 0
		}
		exact = true
	} else if dur := d.durationSamples(rate); dur >= 0 {
		samples = dur
	}

	d.track = container.Track{
		ID:           0,
		Codec:        d.setup.id,
		CodecConfig:  d.setup.config,
		Fmt:          d.setup.fmt,
		Samples:      samples,
		Delay:        delay,
		SamplesExact: exact,
		Default:      true,
	}
	d.resetReading(d.firstClusterOff)
	return nil
}

func (d *Demuxer) needsGaplessWalk() bool {
	return d.setup.id == codec.Vorbis
}

func (d *Demuxer) durationSamples(rate int) int64 {
	if d.durationTicks <= 0 {
		return -1
	}
	ns := int64(d.durationTicks*float64(d.timestampScale) + 0.5)
	return nsToSamples(ns, rate)
}
