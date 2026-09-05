package mka

import (
	"fmt"
	"io"
	"sort"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

// Tracks returns the single selected audio track.
func (d *Demuxer) Tracks() []container.Track { return []container.Track{d.track} }

// Warnings returns damage tolerated during parsing.
func (d *Demuxer) Warnings() []container.Warning { return d.warnings }

func (d *Demuxer) resetReading(off int64) {
	d.curOff = off
	d.inCluster = false
	d.clusterEnd = 0
	d.clusterCursor = 0
	d.clusterUnknown = false
	d.pending = d.pending[:0]
	d.pendingIdx = 0
	d.running = 0
	d.vorbisPrevBlock = 0
	d.curBlockDiscardNS = 0
}

// ReadPacket yields the next codec packet.
func (d *Demuxer) ReadPacket(pkt *container.Packet) error {
	data, dur, sync, err := d.nextFrame()
	if err != nil {
		return err
	}
	*pkt = container.Packet{
		Track: 0,
		Packet: codec.Packet{
			Data: data,
			PTS:  d.running,
			Dur:  dur,
			Sync: sync,
		},
	}
	d.running += dur
	return nil
}

func (d *Demuxer) nextFrame() ([]byte, int64, bool, error) {
	if d.pendingIdx >= len(d.pending) {
		if err := d.advanceBlock(); err != nil {
			return nil, 0, false, err
		}
	}
	f := d.pending[d.pendingIdx]
	d.pendingIdx++
	data, err := d.frameBytes(f)
	if err != nil {
		return nil, 0, false, err
	}
	return data, d.frameSamples(data), true, nil
}

func (d *Demuxer) frameBytes(f frameLoc) ([]byte, error) {
	d.w.Trim(f.off)
	data := d.w.BytesAt(f.off, f.size)
	if len(data) != f.size {
		if e := d.w.Err(); e != nil {
			return nil, e
		}
		return nil, malformed("frame at %d truncated (want %d bytes)", f.off, f.size)
	}
	return data, nil
}

func (d *Demuxer) advanceBlock() error {
	for {
		if !d.inCluster {
			done, err := d.stepSegment()
			if err != nil {
				return err
			}
			if done {
				return io.EOF
			}
			continue
		}
		handled, done, err := d.stepCluster()
		if err != nil {
			return err
		}
		if done {
			return io.EOF
		}
		if handled {
			return nil
		}
	}
}

func (d *Demuxer) stepSegment() (done bool, err error) {
	if d.recording && d.walkLimit >= 0 && d.curOff >= d.walkLimit {
		d.walkStopped = true
		return true, nil
	}
	if d.curOff >= d.segmentEnd {
		return true, nil
	}
	e, err := d.readElement(d.curOff, d.segmentEnd)
	if err != nil {
		if werr := d.warn(d.curOff, "damaged segment element, ending stream"); werr != nil {
			return false, werr
		}
		return true, nil
	}
	if e.id == idCluster {
		d.enterCluster(e, d.curOff)
		return false, nil
	}
	if e.unknownSize {
		return true, nil
	}
	d.curOff = e.dataEnd()
	return false, nil
}

func (d *Demuxer) enterCluster(e element, startOff int64) {
	if d.recording && len(d.clusterIndex) < maxClusters {
		d.clusterIndex = append(d.clusterIndex, clusterPos{off: startOff, sample: d.walkCumulative})
	}
	d.inCluster = true
	d.clusterCursor = e.dataOff
	d.clusterUnknown = e.unknownSize
	if e.unknownSize {
		d.clusterEnd = d.segmentEnd
	} else {
		d.clusterEnd = e.dataEnd()
		d.curOff = e.dataEnd()
	}
}

func (d *Demuxer) stepCluster() (handled, done bool, err error) {
	if d.clusterCursor >= d.clusterEnd {
		d.inCluster = false
		if !d.clusterUnknown {
			d.curOff = d.clusterEnd
		} else {
			d.curOff = d.clusterCursor
		}
		return false, false, nil
	}
	e, err := d.readElement(d.clusterCursor, d.clusterEnd)
	if err != nil {
		if werr := d.warn(d.clusterCursor, "damaged cluster element, ending stream"); werr != nil {
			return false, false, werr
		}
		return false, true, nil
	}
	if d.clusterUnknown && isSegmentLevel(e.id) {
		d.inCluster = false
		d.curOff = d.clusterCursor
		return false, false, nil
	}
	switch e.id {
	case idTimestamp:
		if e.unknownSize {
			return false, true, nil
		}
		d.clusterCursor = e.dataEnd()
	case idSimpleBlock:
		if e.unknownSize {
			return false, true, nil
		}
		ok, err := d.loadBlock(e.dataOff, e.size, 0)
		if err != nil {
			return false, false, err
		}
		d.clusterCursor = e.dataEnd()
		return ok, false, nil
	case idBlockGroup:
		if e.unknownSize {
			return false, true, nil
		}
		ok, err := d.loadBlockGroup(e)
		if err != nil {
			return false, false, err
		}
		d.clusterCursor = e.dataEnd()
		return ok, false, nil
	default:
		if e.unknownSize {
			return false, true, nil
		}
		d.clusterCursor = e.dataEnd()
	}
	return false, false, nil
}

func (d *Demuxer) loadBlock(dataOff, size, discardNS int64) (bool, error) {
	bh, err := parseBlock(&d.w, dataOff, size)
	if err != nil {
		if waxerr.CodeOf(err) == waxerr.CodeSourceUnreadable {
			return false, err
		}
		if werr := d.warn(dataOff, "damaged block, skipped"); werr != nil {
			return false, werr
		}
		return false, nil
	}
	if bh.track != d.sel.number {
		return false, nil
	}
	d.pending = bh.frames
	d.pendingIdx = 0
	d.curBlockDiscardNS = discardNS
	return true, nil
}

func (d *Demuxer) loadBlockGroup(g element) (bool, error) {
	var blockOff, blockSize int64 = -1, 0
	var discardNS int64
	off := g.dataOff
	end := g.dataEnd()
	for off < end {
		e, err := d.readElement(off, end)
		if err != nil {
			if werr := d.warn(off, "damaged block group element"); werr != nil {
				return false, werr
			}
			break
		}
		switch e.id {
		case idBlock:
			blockOff, blockSize = e.dataOff, e.size
		case idDiscardPadding:
			if e.size > 8 {
				if werr := d.warn(e.dataOff, "ignoring oversized DiscardPadding"); werr != nil {
					return false, werr
				}
				break
			}
			body, err := d.readBytes(e.dataOff, e.size, 8)
			if err != nil {
				return false, err
			}
			discardNS = beInt(body)
			if discardNS < 0 {
				if !d.warnedNegativeDiscard {
					d.warnedNegativeDiscard = true
					if werr := d.warn(e.dataOff, "ignoring negative DiscardPadding"); werr != nil {
						return false, werr
					}
				}
				discardNS = 0
			}
		}
		if e.unknownSize {
			break
		}
		off = e.dataEnd()
	}
	if blockOff < 0 {
		return false, nil
	}
	return d.loadBlock(blockOff, blockSize, discardNS)
}

type clusterPos struct {
	off    int64
	sample int64
}

func (d *Demuxer) ensureWalk() error { return d.walk(-1) }

func (d *Demuxer) walk(limit int64) error {
	if d.walked {
		return nil
	}
	if limit >= 0 && !d.boundedWalkSafe() {
		limit = -1
	}
	if limit >= 0 && d.walkedTo >= limit {
		return nil
	}
	if limit < 0 {
		d.walked = true
	}
	fail := func(e error) error {
		d.recording = false
		d.walkLimit = -1
		d.walkedTo = 0
		d.resetReading(d.firstClusterOff)
		return e
	}
	if d.walkedTo > 0 {
		d.resetReading(d.walkedTo)
	} else {
		d.walkCumulative = 0
		d.walkPaddingNS = 0
		d.walkFrames = 0
		d.clusterIndex = d.clusterIndex[:0]
		d.resetReading(d.firstClusterOff)
	}
	d.recording = true
	d.walkLimit = limit
	d.walkStopped = false
	for {
		err := d.advanceBlock()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fail(err)
		}
		for d.pendingIdx < len(d.pending) {
			f := d.pending[d.pendingIdx]
			d.pendingIdx++
			if d.walkFrames++; d.walkFrames > maxFrames {
				return fail(malformed("more than %d frames", int64(maxFrames)))
			}
			dur, derr := d.frameDurAt(f)
			if derr != nil {
				return fail(derr)
			}
			d.walkCumulative += dur
		}
		if d.curBlockDiscardNS > 0 {
			d.walkPaddingNS += d.curBlockDiscardNS
		}
	}
	if d.walkStopped {
		d.walkedTo = d.curOff
	} else {
		d.rawTotal = d.walkCumulative
		d.paddingNS = d.walkPaddingNS
		d.walkedTo = d.segmentEnd
		d.walked = true
	}
	d.recording = false
	d.walkLimit = -1
	d.resetReading(d.firstClusterOff)
	return nil
}

func (d *Demuxer) boundedWalkSafe() bool {
	return d.setup.id != codec.Vorbis
}

const (
	maxSeekSeconds = 1 << 32
	maxSeekRate    = 1 << 24
)

func (d *Demuxer) cueLimit(sample int64) int64 {
	d.resolveCues()
	rate := int64(d.setup.fmt.Rate)
	if len(d.cues) == 0 || rate <= 0 {
		return -1
	}
	if sample <= 0 {
		return d.firstClusterOff
	}
	if rate > maxSeekRate || sample/rate > maxSeekSeconds {
		return -1
	}
	t := samplesToNs(sample, int(rate)) / d.timestampScale
	i := sort.Search(len(d.cues), func(i int) bool { return d.cues[i].time > t })
	if i >= len(d.cues) {
		return -1
	}
	off := d.cues[i].off
	if e, err := d.readElement(off, d.segmentEnd); err != nil || e.id != idCluster {
		return -1
	}
	return off
}

func (d *Demuxer) seekWalk(sample int64) error {
	if d.walked {
		return nil
	}
	limit := d.cueLimit(sample)
	if limit < 0 {
		return d.ensureWalk()
	}
	return d.walk(limit)
}

func (d *Demuxer) frameDurAt(f frameLoc) (int64, error) {
	if d.setup.id == codec.PCM {
		if d.setup.pcmBytesPerFrame <= 0 {
			return 0, nil
		}
		return int64(f.size / d.setup.pcmBytesPerFrame), nil
	}
	n := f.size
	if n > 128 {
		n = 128
	}
	d.w.Trim(f.off)
	data := d.w.BytesAt(f.off, n)
	if len(data) != n {
		if e := d.w.Err(); e != nil {
			return 0, e
		}
		return 0, malformed("frame prefix at %d truncated", f.off)
	}
	return d.frameSamples(data), nil
}

// SeekSample lands on the indexed cluster at or before the target in the raw decoder timeline and returns its exact sample position.
func (d *Demuxer) SeekSample(track int, sample int64) (int64, error) {
	if track != 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mka: no track %d", track))
	}
	if sample < 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, "mka: negative seek target")
	}
	if !d.haveFirstCluster {
		return 0, nil
	}
	if err := d.seekWalk(sample); err != nil {
		return 0, err
	}
	searchRaw := sample - d.seekPreRollSamples
	if searchRaw < 0 {
		searchRaw = 0
	}
	clusterOff, landed := d.firstClusterOff, int64(0)
	if len(d.clusterIndex) > 0 {
		idx := sort.Search(len(d.clusterIndex), func(i int) bool {
			return d.clusterIndex[i].sample > searchRaw
		}) - 1
		if idx < 0 {
			idx = 0
		}
		clusterOff = d.clusterIndex[idx].off
		landed = d.clusterIndex[idx].sample
	}
	d.resetReading(clusterOff)
	d.running = landed
	return landed, nil
}

func isSegmentLevel(id uint32) bool {
	switch id {
	case idCluster, idCues, idSeekHead, idInfo, idTracks,
		0x1043A770,
		0x1254C367,
		0x1941A469:
		return true
	}
	return false
}

func codecName(codecID string) string {
	if codecID == "" {
		return "unknown"
	}
	if len(codecID) > 2 && codecID[:2] == "A_" {
		return codecID[2:]
	}
	return codecID
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
