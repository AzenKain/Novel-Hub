package mp4

import (
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/internal/srcwin"
)

const (
	maxMoofBytes          = 8 << 20
	maxSamplesPerFragment = 1 << 20
)

type trexDefaults struct {
	trackID      uint32
	defaultDur   uint32
	defaultSize  uint32
	defaultFlags uint32
	have         bool
}

type fragSample struct {
	off  int64
	size uint32
	dur  uint32
	sync bool
}

func (d *Demuxer) parseMvex(payload []byte) {
	d.fragmented = true
	_ = walkBoxes(payload, func(typ string, body []byte) error {
		if typ != "trex" {
			return nil
		}
		if _, _, rest, ok := fullBox(body); ok && len(rest) >= 20 {
			d.trex = trexDefaults{
				trackID:      be32(rest[0:]),
				defaultDur:   be32(rest[8:]),
				defaultSize:  be32(rest[12:]),
				defaultFlags: be32(rest[16:]),
				have:         true,
			}
		}
		return nil
	})
}

func (d *Demuxer) fragmentedGapless(t *track) (delay, samples int64, exact bool) {
	if !t.hasEdit {
		return 0, -1, false
	}
	delay, seg, haveSeg := editListTrims(t, d.movieTimescale)
	if haveSeg {
		return delay, seg, true
	}
	return delay, -1, false
}

// NewFragmentedDemuxer reads a bare CMAF/HLS media segment (moof+mdat with no ftyp/moov) using an out-of-band init segment for the codec config, sample entry, mvex defaults, and edit list.
func NewFragmentedDemuxer(init []byte, media container.Source) (*Demuxer, error) {
	d := &Demuxer{
		src:  media,
		size: media.Size(),
		w:    srcwin.New(media, media.Size(), "mp4: reading sample data"),
	}
	moov, err := findInitMoov(init)
	if err != nil {
		return nil, err
	}
	tracks, err := d.parseMoov(moov)
	if err != nil {
		return nil, err
	}
	d.fragmented = true
	if err := d.selectAudio(tracks); err != nil {
		return nil, err
	}
	d.fragOff = 0
	return d, nil
}

func findInitMoov(init []byte) ([]byte, error) {
	var moov []byte
	err := walkBoxes(init, func(typ string, payload []byte) error {
		if typ == "moov" && moov == nil {
			moov = payload
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if moov == nil {
		return nil, malformed("init segment has no moov box")
	}
	return moov, nil
}

func (d *Demuxer) readFragmentedPacket(pkt *container.Packet) error {
	for d.fragIdx >= len(d.fragQueue) {
		if err := d.nextFragment(); err != nil {
			return err
		}
	}
	s := d.fragQueue[d.fragIdx]
	d.fragIdx++
	d.w.Trim(s.off)
	data := d.w.BytesAt(s.off, int(s.size))
	if len(data) != int(s.size) {
		if err := d.w.Err(); err != nil {
			return err
		}
		return malformed("fragment sample at %d truncated (want %d bytes)", s.off, s.size)
	}
	pts := d.fragDecode
	d.fragDecode += int64(s.dur)
	*pkt = container.Packet{
		Track: 0,
		Packet: codec.Packet{
			Data: data,
			PTS:  pts,
			Dur:  int64(s.dur),
			Sync: s.sync,
		},
	}
	return nil
}

func (d *Demuxer) nextFragment() error {
	for d.fragOff < d.size {
		b, err := readBox(d.src, d.fragOff, d.size)
		if err != nil {
			return err
		}
		if b.typ == "moof" {
			return d.loadFragment(b)
		}
		if b.toEnd {
			break
		}
		d.fragOff = b.off + b.size
	}
	return io.EOF
}

func (d *Demuxer) loadFragment(moof box) error {
	if moof.payloadLen() > maxMoofBytes {
		return malformed("moof of %d bytes exceeds the %d cap", moof.payloadLen(), int64(maxMoofBytes))
	}
	buf := make([]byte, moof.payloadLen())
	if err := container.ReadFull(d.src, buf, moof.payloadOff()); err != nil {
		return err
	}
	fi, err := parseFragment(buf, d.trex, d.sel.id)
	if err != nil {
		return err
	}
	base := moof.off
	if fi.haveBaseOffset {
		base = fi.baseDataOffset
	}
	off := base + int64(fi.dataOffset)

	rate := int64(d.sel.fmt.Rate)
	rescale := d.sel.timescale > 0 && rate > 0 && d.sel.timescale != rate

	d.fragQueue = d.fragQueue[:0]
	for _, s := range fi.samples {
		if off < 0 || off > d.size-int64(s.size) {
			return malformed("fragment sample runs past end of source")
		}
		dur := int64(s.dur)
		if rescale {
			dur = rescaleTicks(dur, rate, d.sel.timescale)
		}
		if dur < 1 {
			dur = 1
		}
		d.fragQueue = append(d.fragQueue, fragSample{off: off, size: s.size, dur: uint32(dur), sync: s.sync})
		off += int64(s.size)
	}
	d.fragIdx = 0
	d.fragDecode = fi.baseDecodeTime
	if rescale {
		d.fragDecode = mulDivSat(fi.baseDecodeTime, rate, d.sel.timescale)
	}

	d.fragOff = moof.off + moof.size
	if mdat, err := readBox(d.src, d.fragOff, d.size); err == nil && mdat.typ == "mdat" {
		d.fragOff = mdat.off + mdat.size
	}
	return nil
}

type fragInfo struct {
	baseDecodeTime int64
	dataOffset     int32
	baseDataOffset int64
	haveBaseOffset bool
	samples        []fragSampleInfo
}

type fragSampleInfo struct {
	dur, size uint32
	sync      bool
}

func parseFragment(buf []byte, trex trexDefaults, selID int) (fragInfo, error) {
	var fi fragInfo
	matched := false
	var perr error
	_ = walkBoxes(buf, func(typ string, body []byte) error {
		if typ != "traf" || matched {
			return nil
		}
		if id, ok := trafTrackID(body); selID > 0 && ok && int(id) != selID {
			return nil
		}
		matched = true
		perr = parseTraf(&fi, body, trex)
		return nil
	})
	return fi, perr
}

func trafTrackID(body []byte) (uint32, bool) {
	var id uint32
	found := false
	_ = walkBoxes(body, func(typ string, p []byte) error {
		if typ == "tfhd" && !found {
			if _, _, rest, ok := fullBox(p); ok && len(rest) >= 4 {
				id, found = be32(rest), true
			}
		}
		return nil
	})
	return id, found
}

func parseTraf(fi *fragInfo, body []byte, trex trexDefaults) error {
	defaultDur, defaultSize, defaultFlags := trex.defaultDur, trex.defaultSize, trex.defaultFlags
	var trun []byte
	haveTrun := false
	err := walkBoxes(body, func(typ string, p []byte) error {
		switch typ {
		case "tfhd":
			parseTfhd(fi, p, &defaultDur, &defaultSize, &defaultFlags)
		case "tfdt":
			parseTfdt(fi, p)
		case "trun":
			if !haveTrun {
				trun, haveTrun = p, true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !haveTrun {
		return malformed("traf has no trun")
	}
	return parseTrun(fi, trun, defaultDur, defaultSize, defaultFlags)
}

func parseTfhd(fi *fragInfo, payload []byte, defaultDur, defaultSize, defaultFlags *uint32) {
	_, flags, rest, ok := fullBox(payload)
	if !ok || len(rest) < 4 {
		return
	}
	q := rest[4:]
	take := func(n int) []byte {
		if len(q) < n {
			q = nil
			return nil
		}
		b := q[:n]
		q = q[n:]
		return b
	}
	if flags&0x000001 != 0 {
		if b := take(8); b != nil {
			fi.baseDataOffset = int64(be64(b))
			fi.haveBaseOffset = true
		}
	}
	if flags&0x000002 != 0 {
		take(4)
	}
	if flags&0x000008 != 0 {
		if b := take(4); b != nil {
			*defaultDur = be32(b)
		}
	}
	if flags&0x000010 != 0 {
		if b := take(4); b != nil {
			*defaultSize = be32(b)
		}
	}
	if flags&0x000020 != 0 {
		if b := take(4); b != nil {
			*defaultFlags = be32(b)
		}
	}
}

func parseTfdt(fi *fragInfo, payload []byte) {
	version, _, rest, ok := fullBox(payload)
	if !ok {
		return
	}
	if version == 1 {
		if len(rest) >= 8 {
			fi.baseDecodeTime = int64(be64(rest))
		}
		return
	}
	if len(rest) >= 4 {
		fi.baseDecodeTime = int64(be32(rest))
	}
}

func parseTrun(fi *fragInfo, trun []byte, defaultDur, defaultSize, defaultFlags uint32) error {
	_, flags, rest, ok := fullBox(trun)
	if !ok || len(rest) < 4 {
		return malformed("trun truncated")
	}
	count := be32(rest)
	rest = rest[4:]
	if count > maxSamplesPerFragment {
		return malformed("trun declares %d samples", count)
	}
	if flags&0x000001 != 0 {
		if len(rest) < 4 {
			return malformed("trun data offset truncated")
		}
		fi.dataOffset = int32(be32(rest))
		rest = rest[4:]
	}
	var firstFlags uint32
	haveFirstFlags := false
	if flags&0x000004 != 0 {
		if len(rest) < 4 {
			return malformed("trun first-sample flags truncated")
		}
		firstFlags = be32(rest)
		rest = rest[4:]
		haveFirstFlags = true
	}
	perSample := 0
	for _, f := range []uint32{0x000100, 0x000200, 0x000400, 0x000800} {
		if flags&f != 0 {
			perSample += 4
		}
	}
	if int64(count)*int64(perSample) > int64(len(rest)) {
		return malformed("trun declares %d samples for %d bytes", count, len(rest))
	}
	fi.samples = make([]fragSampleInfo, count)
	for i := uint32(0); i < count; i++ {
		dur, size, sflags := defaultDur, defaultSize, defaultFlags
		if flags&0x000100 != 0 {
			dur = be32(rest)
			rest = rest[4:]
		}
		if flags&0x000200 != 0 {
			size = be32(rest)
			rest = rest[4:]
		}
		if flags&0x000400 != 0 {
			sflags = be32(rest)
			rest = rest[4:]
		}
		if flags&0x000800 != 0 {
			rest = rest[4:]
		}
		if i == 0 && haveFirstFlags {
			sflags = firstFlags
		}
		fi.samples[i] = fragSampleInfo{dur: dur, size: size, sync: sflags&0x00010000 == 0}
	}
	return nil
}

func (d *Demuxer) seekFragmented(sample int64) (int64, error) {
	off := d.fragStart
	landedOff, landed := off, int64(0)
	rate := int64(d.sel.fmt.Rate)
	rescale := d.sel.timescale > 0 && rate > 0 && d.sel.timescale != rate
	var scratch []byte
	for off < d.size {
		b, err := readBox(d.src, off, d.size)
		if err != nil {
			return 0, err
		}
		if b.typ == "moof" {
			if b.payloadLen() > maxMoofBytes {
				return 0, malformed("moof exceeds cap during seek")
			}
			if int64(cap(scratch)) < b.payloadLen() {
				scratch = make([]byte, b.payloadLen())
			}
			buf := scratch[:b.payloadLen()]
			if err := container.ReadFull(d.src, buf, b.payloadOff()); err != nil {
				return 0, err
			}
			base := moofBaseTime(buf)
			if rescale {
				base = mulDivSat(base, rate, d.sel.timescale)
			}
			if base > sample {
				break
			}
			landedOff, landed = b.off, base
		}
		if b.toEnd {
			break
		}
		off = b.off + b.size
	}
	d.fragOff = landedOff
	d.fragQueue = d.fragQueue[:0]
	d.fragIdx = 0
	d.fragDecode = landed
	return landed, nil
}

func moofBaseTime(buf []byte) int64 {
	var fi fragInfo
	_ = walkBoxes(buf, func(typ string, body []byte) error {
		if typ != "traf" {
			return nil
		}
		return walkBoxes(body, func(inner string, p []byte) error {
			if inner == "tfdt" {
				parseTfdt(&fi, p)
			}
			return nil
		})
	})
	return fi.baseDecodeTime
}
