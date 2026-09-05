package mpa

import (
	"bytes"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/mp3"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/internal/id3"
	"novelhub/pkg/waxflow/container/internal/srcwin"
	"novelhub/pkg/waxflow/waxerr"
)

var (
	_ container.Demuxer = (*Demuxer)(nil)
	_ container.Seeker  = (*Demuxer)(nil)
	_ container.Warner  = (*Demuxer)(nil)
)

const (
	maxResync      = 1 << 20
	minFrameLen    = 24
	maxID3Tags     = 8
	trailerScan    = 64 << 10
	reservoirCover = 511 + 64
	stateFrames    = 3
)

// DemuxerOptions configures parsing.
type DemuxerOptions struct {
	Strict bool
}

// Demuxer reads one MP3 track from an elementary stream source.
type Demuxer struct {
	src  container.Source
	opts DemuxerOptions

	hdr      mp3.Header
	spf      int64
	track    container.Track
	warnings []container.Warning

	firstFrame int64

	idx  []int64
	done bool
	grew bool

	cur int64

	w srcwin.Window
}

// NewDemuxer parses the stream head (tags, the VBR metadata frame) and positions on the first audio frame.
func NewDemuxer(src container.Source, opts *DemuxerOptions) (*Demuxer, error) {
	d := &Demuxer{src: src, w: srcwin.New(src, src.Size(), "mpa: reading frame data")}
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
		return waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("mpa: %s (at offset %d)", msg, off))
	}
	d.warnings = append(d.warnings, container.Warning{Offset: off, Msg: msg})
	return nil
}

func (d *Demuxer) parse() error {
	off := int64(0)
	for range maxID3Tags {
		head := d.w.BytesAt(off, 10)
		n := id3.Size(head)
		if n == 0 || off+n > d.w.DataEnd() {
			break
		}
		off += n
	}

	if fh, err := mp3.ParseHeader(d.w.BytesAt(off, mp3.HeaderLen)); err == nil && fh.Size() == 0 {
		return waxerr.New(waxerr.CodeUnsupportedFormat, "mpa: free-format stream")
	}

	first, h, ok := d.nextCandidate(off, off+maxResync)
	if !ok {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return waxerr.New(waxerr.CodeUnsupportedFormat, "mpa: no Layer III frames found")
	}
	if first != off {
		if err := d.warn(off, "%d unparsable bytes before the first frame", first-off); err != nil {
			return err
		}
	}

	tag, hasTag := vbrTag{delay: -1, padding: -1}, false
	if frame := d.w.BytesAt(first, h.Size()); len(frame) == h.Size() {
		if t, ok := parseVBRTag(h, frame); ok {
			tag, hasTag = t, true
			first += int64(h.Size())
			fh, err := mp3.ParseHeader(d.w.BytesAt(first, mp3.HeaderLen))
			if err != nil || !h.Kin(fh) {
				var ok bool
				first, fh, ok = d.nextCandidate(first, first+maxResync)
				if !ok {
					if d.w.Err() != nil {
						return d.w.Err()
					}
					return waxerr.New(waxerr.CodeUnsupportedFormat, "mpa: no audio frames after the VBR tag")
				}
			}
			h = fh
		}
	}
	if d.w.Err() != nil {
		return d.w.Err()
	}

	d.hdr = h
	d.spf = int64(h.SamplesPerFrame())
	d.firstFrame = first
	if first+int64(h.Size()) <= d.w.DataEnd() {
		d.idx = append(d.idx, first)
	} else if err := d.warn(first, "the only frame is truncated, dropped"); err != nil {
		return err
	}

	f := h.PCMFormat()
	if err := f.Valid(); err != nil {
		return waxerr.Wrap(waxerr.CodeUnsupportedFormat, "mpa: unusable format", err)
	}

	samples, delay, padding := int64(-1), int64(0), int64(0)
	if hasTag && tag.delay >= 0 && tag.frames > 0 {
		delay = tag.delay + decoderDelay
		padding = max(tag.padding-decoderDelay, 0)
		samples = max(tag.frames*d.spf-delay-padding, 0)
	} else if hasTag && tag.frames > 0 {
		samples = tag.frames * d.spf
	}
	d.track = container.Track{
		Codec:   codec.MP3,
		Fmt:     f,
		Samples: samples,
		Delay:   delay,
		Padding: padding,
		Default: true,
	}
	return nil
}

// Tracks returns the single MP3 track.
func (d *Demuxer) Tracks() []container.Track { return []container.Track{d.track} }

// Warnings returns damage tolerated during parsing.
func (d *Demuxer) Warnings() []container.Warning { return d.warnings }

func (d *Demuxer) nextCandidate(from, limit int64) (int64, mp3.Header, bool) {
	limit = min(limit, d.w.DataEnd())
	ref := d.hdr
	haveRef := d.spf != 0
	for off := from; off < limit; {
		buf := d.w.BytesAt(off, srcwin.Chunk)
		if len(buf) == 0 {
			return 0, mp3.Header{}, false
		}
		i := bytes.IndexByte(buf, 0xFF)
		if i < 0 {
			off += int64(len(buf))
			continue
		}
		cand := off + int64(i)
		if cand >= limit {
			return 0, mp3.Header{}, false
		}
		h, err := mp3.ParseHeader(d.w.BytesAt(cand, mp3.HeaderLen))
		if err == nil && h.Size() != 0 && (!haveRef || ref.Kin(h)) {
			next := cand + int64(h.Size())
			if next >= d.w.DataEnd() {
				return cand, h, true
			}
			nh, nerr := mp3.ParseHeader(d.w.BytesAt(next, mp3.HeaderLen))
			if nerr == nil && h.Kin(nh) {
				return cand, h, true
			}
		}
		off = cand + 1
	}
	return 0, mp3.Header{}, false
}

func (d *Demuxer) frameAt(off int64) (mp3.Header, bool) {
	h, err := mp3.ParseHeader(d.w.BytesAt(off, mp3.HeaderLen))
	if err != nil || !d.hdr.Kin(h) || h.Size() == 0 {
		return mp3.Header{}, false
	}
	return h, true
}

func (d *Demuxer) extend() (bool, error) {
	if d.done {
		return false, nil
	}
	if d.w.Err() != nil {
		return false, d.w.Err()
	}
	if len(d.idx) == 0 {
		d.done = true
		return false, nil
	}
	last := d.idx[len(d.idx)-1]
	h, ok := d.frameAt(last)
	if !ok {
		d.done = true
		return false, d.w.Err()
	}
	next := last + int64(h.Size())
	if next >= d.w.DataEnd() {
		d.done = true
		return false, nil
	}
	cand := next
	nh, ok := d.frameAt(next)
	if !ok {
		cand, nh, ok = d.nextCandidate(next, next+maxResync)
		if !ok {
			if d.w.Err() != nil {
				return false, d.w.Err()
			}
			d.done = true
			if tail := d.w.DataEnd() - next; tail > 0 && !d.recognizedTrailer(next) {
				return false, d.warn(next, "%d trailing bytes are not frames, dropped", tail)
			}
			return false, nil
		}
		if err := d.warn(next, "%d unparsable bytes skipped", cand-next); err != nil {
			return false, err
		}
	}
	if cand+int64(nh.Size()) > d.w.DataEnd() {
		d.done = true
		return false, d.warn(cand, "truncated final frame dropped")
	}
	d.idx = append(d.idx, cand)
	d.grew = true
	return true, nil
}

func (d *Demuxer) recognizedTrailer(off int64) bool {
	if d.w.DataEnd()-off > trailerScan {
		return false
	}
	b := d.w.BytesAt(off, int(d.w.DataEnd()-off))
	for len(b) > 0 {
		switch {
		case len(b) >= 3 && string(b[:3]) == "TAG":
			b = b[min(128, len(b)):]
		case len(b) >= 8 && string(b[:8]) == "APETAGEX":
			return true
		case id3.Size(b) > 0:
			n := id3.Size(b)
			if n > int64(len(b)) {
				return true
			}
			b = b[n:]
		case len(b) >= 11 && string(b[:11]) == "LYRICSBEGIN":
			return true
		case b[0] == 0:
			i := 0
			for i < len(b) && b[i] == 0 {
				i++
			}
			b = b[i:]
		default:
			return false
		}
	}
	return true
}

func (d *Demuxer) frameNo(n int64) (int64, error) {
	for int64(len(d.idx)) <= n {
		grew, err := d.extend()
		if err != nil {
			return 0, err
		}
		if !grew {
			break
		}
	}
	return int64(len(d.idx)) - 1, nil
}

// ReadPacket yields one whole frame.
func (d *Demuxer) ReadPacket(pkt *container.Packet) error {
	lastNo, err := d.frameNo(d.cur)
	if err != nil {
		return err
	}
	if d.cur > lastNo || lastNo < 0 {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return io.EOF
	}
	off := d.idx[d.cur]
	h, ok := d.frameAt(off)
	if !ok {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return waxerr.New(waxerr.CodeSourceUnreadable, "mpa: indexed frame vanished")
	}
	d.w.Trim(off)
	data := d.w.BytesAt(off, h.Size())
	if len(data) != h.Size() {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return waxerr.New(waxerr.CodeSourceUnreadable, "mpa: reading frame data")
	}
	*pkt = container.Packet{
		Track: 0,
		Packet: codec.Packet{
			Data: data,
			PTS:  d.cur * d.spf,
			Dur:  d.spf,
			Sync: syncFrame(h, data),
		},
	}
	d.cur++
	return nil
}

func syncFrame(h mp3.Header, frame []byte) bool {
	off := mp3.HeaderLen
	if h.Protected {
		off += 2
	}
	if len(frame) <= off+1 {
		return false
	}
	if h.Version == mp3.MPEG1 {
		return frame[off] == 0 && frame[off+1]&0x80 == 0
	}
	return frame[off] == 0
}

// SeekSample repositions so the reader is far enough before the target that decoder state converges: landings precede the target frame by stateFrames plus however many frames the bit reservoir's reach needs.
func (d *Demuxer) SeekSample(track int, sample int64) (int64, error) {
	if track != 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("mpa: no track %d", track))
	}
	if sample < 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, "mpa: negative seek target")
	}
	target := sample / d.spf
	lastNo, err := d.frameNo(target)
	if err != nil {
		return 0, err
	}
	if lastNo < 0 {
		return 0, nil
	}
	target = min(target, lastNo)

	overhead := int64(mp3.HeaderLen + d.hdr.SideInfoLen())
	if d.hdr.Protected {
		overhead += 2
	}
	land := max(target-stateFrames, 0)
	cover := int64(0)
	for land > 0 && cover < reservoirCover {
		land--
		cover += max(d.frameSize(land)-overhead, 0)
	}
	d.cur = land
	return land * d.spf, nil
}

func (d *Demuxer) frameSize(n int64) int64 {
	if n+1 < int64(len(d.idx)) {
		return d.idx[n+1] - d.idx[n]
	}
	if h, ok := d.frameAt(d.idx[n]); ok {
		return int64(h.Size())
	}
	return 0
}
