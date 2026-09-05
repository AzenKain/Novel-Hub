package flacn

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/internal/srcwin"
	"novelhub/pkg/waxflow/waxerr"
)

var (
	_ container.Demuxer = (*Demuxer)(nil)
	_ container.Seeker  = (*Demuxer)(nil)
	_ container.Warner  = (*Demuxer)(nil)
)

const (
	maxMetaBlocks = 4096
	maxSeekPoints = 1 << 16
	maxFrameLen   = 1 << 24
	maxResync     = 1 << 20
	seekWindow    = 128 << 10
	maxTrailers   = 8
)

// Metadata block types (RFC 9639 section 8).
const (
	blockStreamInfo = 0
	blockSeekTable  = 3
	blockInvalid    = 127
)

// DemuxerOptions configures parsing.
type DemuxerOptions struct {
	Strict bool
}

type seekPoint struct {
	sample int64
	off    int64
}

// Demuxer reads one FLAC track from a native FLAC source.
type Demuxer struct {
	src  container.Source
	opts DemuxerOptions

	si        flac.StreamInfo
	track     container.Track
	seekTable []seekPoint
	warnings  []container.Warning

	firstFrame int64

	num    flac.Numbering
	varBit bool

	off   int64
	cur   flac.FrameInfo
	valid bool
	empty bool

	w srcwin.Window
}

// NewDemuxer parses the headers of a native FLAC source and positions on the first frame.
func NewDemuxer(src container.Source, opts *DemuxerOptions) (*Demuxer, error) {
	d := &Demuxer{src: src, w: srcwin.New(src, src.Size(), "flacn: reading frame data")}
	if opts != nil {
		d.opts = *opts
	}
	if err := d.parse(); err != nil {
		return nil, err
	}
	return d, nil
}

func malformed(format string, args ...any) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, "flacn: "+fmt.Sprintf(format, args...))
}

func (d *Demuxer) warn(off int64, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if d.opts.Strict {
		return malformed("%s (at offset %d)", msg, off)
	}
	d.warnings = append(d.warnings, container.Warning{Offset: off, Msg: msg})
	return nil
}

func (d *Demuxer) parse() error {
	size := d.src.Size()
	var head [4]byte
	if err := container.ReadFull(d.src, head[:], 0); err != nil {
		return waxerr.Wrap(waxerr.CodeUnsupportedFormat, "flacn: reading marker", err)
	}
	if !Match(head[:]) {
		return malformed("not a FLAC file")
	}

	var (
		siRaw   []byte
		haveSI  bool
		off     = int64(4)
		last    = false
		blockNo = 0
	)
	for !last {
		if blockNo++; blockNo > maxMetaBlocks {
			return malformed("more than %d metadata blocks", maxMetaBlocks)
		}
		var hdr [4]byte
		if err := container.ReadFull(d.src, hdr[:], off); err != nil {
			return waxerr.Wrap(waxerr.CodeUnsupportedFormat, "flacn: reading metadata block header", err)
		}
		last = hdr[0]&0x80 != 0
		typ := int(hdr[0] & 0x7F)
		length := int64(hdr[1])<<16 | int64(hdr[2])<<8 | int64(hdr[3])
		if off+4+length > size {
			return malformed("metadata block extends past end of file")
		}
		switch typ {
		case blockStreamInfo:
			if haveSI {
				if err := d.warn(off, "duplicate STREAMINFO ignored"); err != nil {
					return err
				}
				break
			}
			if blockNo != 1 {
				if err := d.warn(off, "STREAMINFO is not the first metadata block"); err != nil {
					return err
				}
			}
			if length != flac.StreamInfoLen {
				return malformed("STREAMINFO of %d bytes", length)
			}
			siRaw = make([]byte, flac.StreamInfoLen)
			if err := container.ReadFull(d.src, siRaw, off+4); err != nil {
				return waxerr.Wrap(waxerr.CodeSourceUnreadable, "flacn: reading STREAMINFO", err)
			}
			var err error
			if d.si, err = flac.ParseStreamInfo(siRaw); err != nil {
				return err
			}
			haveSI = true
		case blockSeekTable:
			if err := d.parseSeekTable(off+4, length); err != nil {
				return err
			}
		case blockInvalid:
			return malformed("forbidden metadata block type 127")
		}
		off += 4 + length
	}
	if !haveSI {
		return malformed("no STREAMINFO metadata block")
	}

	d.firstFrame = off

	f := d.si.PCMFormat()
	if err := f.Valid(); err != nil {
		return waxerr.Wrap(waxerr.CodeUnsupportedFormat, "flacn: unusable format", err)
	}
	samples := d.si.Samples
	if samples == 0 {
		samples = -1
	}
	d.track = container.Track{
		Codec:       codec.FLAC,
		CodecConfig: siRaw,
		Fmt:         f,
		Samples:     samples,
		Default:     true,
	}

	fi, err := flac.ParseFrameHeader(d.w.BytesAt(off, flac.MaxFrameHeaderLen))
	if err != nil || !d.consistent(fi) {
		if d.firstFrame >= d.w.DataEnd() {
			if d.si.Samples != 0 {
				if werr := d.warn(off, "STREAMINFO declares %d samples but the stream has no frames", d.si.Samples); werr != nil {
					return werr
				}
			}
			d.track.Samples = 0
			d.empty = true
			return d.w.Err()
		}
		fOff, ffi, ok := d.nextCandidate(off, off+maxResync)
		if !ok {
			if d.w.Err() != nil {
				return d.w.Err()
			}
			return malformed("no audio frame after the metadata blocks")
		}
		if werr := d.warn(off, "%d unparsable bytes before the first frame", fOff-off); werr != nil {
			return werr
		}
		d.firstFrame, fi = fOff, ffi
	}
	if d.w.Err() != nil {
		return d.w.Err()
	}
	d.varBit = fi.Variable
	d.num = d.si.Numbering(fi)
	d.off, d.cur, d.valid = d.firstFrame, fi, true
	return nil
}

func (d *Demuxer) parseSeekTable(off, length int64) error {
	if length%18 != 0 {
		if err := d.warn(off, "SEEKTABLE length %d is not whole seek points", length); err != nil {
			return err
		}
	}
	points := length / 18
	if points > maxSeekPoints {
		points = maxSeekPoints
	}
	raw := make([]byte, points*18)
	if err := container.ReadFull(d.src, raw, off); err != nil {
		return waxerr.Wrap(waxerr.CodeSourceUnreadable, "flacn: reading SEEKTABLE", err)
	}
	d.seekTable = make([]seekPoint, 0, points)
	for i := int64(0); i < points; i++ {
		b := raw[i*18:]
		sample := binary.BigEndian.Uint64(b)
		if sample == 1<<64-1 {
			continue
		}
		p := seekPoint{sample: int64(sample), off: int64(binary.BigEndian.Uint64(b[8:]))}
		if p.sample < 0 || p.off < 0 ||
			(len(d.seekTable) > 0 && p.sample <= d.seekTable[len(d.seekTable)-1].sample) {
			if err := d.warn(off+i*18, "invalid seek point dropped"); err != nil {
				return err
			}
			continue
		}
		d.seekTable = append(d.seekTable, p)
	}
	return nil
}

// Tracks returns the single FLAC track.
func (d *Demuxer) Tracks() []container.Track { return []container.Track{d.track} }

// Warnings returns damage tolerated during parsing.
func (d *Demuxer) Warnings() []container.Warning { return d.warnings }

func (d *Demuxer) consistent(fi flac.FrameInfo) bool {
	rate, bits := fi.Rate, fi.Bits
	if rate == 0 {
		rate = d.si.Rate
	}
	if bits == 0 {
		bits = d.si.Bits
	}
	return rate == d.si.Rate && bits == d.si.Bits && fi.Channels == d.si.Channels
}

func (d *Demuxer) nextCandidate(from, limit int64) (int64, flac.FrameInfo, bool) {
	limit = min(limit, d.w.DataEnd())
	for off := from; off < limit; {
		buf := d.w.BytesAt(off, srcwin.Chunk)
		if len(buf) < 2 {
			return 0, flac.FrameInfo{}, false
		}
		i := bytes.IndexByte(buf, 0xFF)
		if i < 0 {
			off += int64(len(buf))
			continue
		}
		cand := off + int64(i)
		if cand >= limit {
			return 0, flac.FrameInfo{}, false
		}
		hdr := d.w.BytesAt(cand, flac.MaxFrameHeaderLen)
		if flac.SyncOK(hdr) {
			if fi, err := flac.ParseFrameHeader(hdr); err == nil && d.consistent(fi) {
				return cand, fi, true
			}
		}
		off = cand + 1
	}
	return 0, flac.FrameInfo{}, false
}

func (d *Demuxer) findEnd() (end int64, next flac.FrameInfo, nextOK bool, err error) {
	start := d.off
	crc := uint16(0)
	crcPos := start

	scan := start + 4
	for {
		if scan-start > maxFrameLen {
			return 0, flac.FrameInfo{}, false, malformed("frame at offset %d exceeds %d bytes", start, int64(maxFrameLen))
		}
		buf := d.w.BytesAt(scan, srcwin.Chunk)
		if len(buf) == 0 {
			break
		}
		rel := 0
		for {
			i := bytes.IndexByte(buf[rel:], 0xFF)
			if i < 0 {
				break
			}
			cand := scan + int64(rel+i)
			rel += i + 1
			if cand-2 < crcPos || cand+2 > d.w.DataEnd() {
				continue
			}
			hdr := d.w.BytesAt(cand, flac.MaxFrameHeaderLen)
			if !flac.SyncOK(hdr) {
				continue
			}
			fi, perr := flac.ParseFrameHeader(hdr)
			if perr != nil || fi.Variable != d.varBit || fi.Coded != d.num.Next(d.cur) {
				continue
			}
			crc = flac.UpdateCRC16(crc, d.w.BytesAt(crcPos, int(cand-2-crcPos)))
			crcPos = cand - 2
			tail := d.w.BytesAt(crcPos, 2)
			if len(tail) == 2 && crc == uint16(tail[0])<<8|uint16(tail[1]) {
				if !d.consistent(fi) {
					return 0, flac.FrameInfo{}, false,
						malformed("mid-stream format change at offset %d", cand)
				}
				return cand, fi, true, nil
			}
			crc = flac.UpdateCRC16(crc, tail)
			crcPos += int64(len(tail))
		}
		scan += int64(len(buf))
	}
	if d.w.Err() != nil {
		return 0, flac.FrameInfo{}, false, d.w.Err()
	}

	end = d.w.DataEnd()
	for range maxTrailers {
		if d.tailChecks(crc, crcPos, end) {
			if end != d.w.DataEnd() {
				if werr := d.warn(end, "%d trailing tag or padding bytes ignored", d.w.DataEnd()-end); werr != nil {
					return 0, flac.FrameInfo{}, false, werr
				}
				d.w.SetDataEnd(end)
			}
			return end, flac.FrameInfo{}, false, nil
		}
		stripped, ok := d.stripTrailer(start, end)
		if !ok {
			break
		}
		end = stripped
	}
	if d.w.Err() != nil {
		return 0, flac.FrameInfo{}, false, d.w.Err()
	}
	if werr := d.warn(start, "%d trailing bytes are not a valid frame, dropped", d.w.DataEnd()-start); werr != nil {
		return 0, flac.FrameInfo{}, false, werr
	}
	return -1, flac.FrameInfo{}, false, nil
}

func (d *Demuxer) stripTrailer(start, end int64) (int64, bool) {
	if e := end - 128; e >= start+2 {
		if string(d.w.BytesAt(e, 3)) == "TAG" {
			return e, true
		}
	}
	if e := end - 32; e >= start+2 {
		if f := d.w.BytesAt(e, 32); len(f) == 32 && string(f[:8]) == "APETAGEX" {
			total := int64(binary.LittleEndian.Uint32(f[12:16]))
			if binary.LittleEndian.Uint32(f[20:24])&(1<<31) != 0 {
				total += 32
			}
			if total >= 32 && end-total >= start+2 {
				return end - total, true
			}
		}
	}
	if e := end - 10; e >= start+2 {
		if f := d.w.BytesAt(e, 10); len(f) == 10 && string(f[:3]) == "3DI" &&
			(f[6]|f[7]|f[8]|f[9])&0x80 == 0 {
			size := int64(f[6])<<21 | int64(f[7])<<14 | int64(f[8])<<7 | int64(f[9])
			if total := size + 20; end-total >= start+2 {
				return end - total, true
			}
		}
	}
	if n := min(int64(64<<10), end-(start+2)); n > 0 {
		if tail := d.w.BytesAt(end-n, int(n)); int64(len(tail)) == n {
			i := int64(len(tail))
			for i > 0 && tail[i-1] == 0 {
				i--
			}
			if i < n {
				return end - (n - i), true
			}
		}
	}
	return end, false
}

func (d *Demuxer) tailChecks(crc uint16, crcPos, end int64) bool {
	if end-crcPos < 2 || end > d.w.DataEnd() {
		return false
	}
	crc = flac.UpdateCRC16(crc, d.w.BytesAt(crcPos, int(end-2-crcPos)))
	tail := d.w.BytesAt(end-2, 2)
	if len(tail) < 2 {
		return false
	}
	return crc == uint16(tail[0])<<8|uint16(tail[1])
}

// ReadPacket yields one whole frame, checksum included.
func (d *Demuxer) ReadPacket(pkt *container.Packet) error {
	if !d.valid {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return io.EOF
	}
	d.w.Trim(d.off)
	end, next, nextOK, err := d.findEnd()
	if err != nil {
		return err
	}
	if end < 0 {
		d.valid = false
		return io.EOF
	}
	data := d.w.BytesAt(d.off, int(end-d.off))
	if int64(len(data)) != end-d.off {
		if d.w.Err() != nil {
			return d.w.Err()
		}
		return waxerr.New(waxerr.CodeSourceUnreadable, "flacn: reading frame data")
	}
	*pkt = container.Packet{
		Track: 0,
		Packet: codec.Packet{
			Data: data,
			PTS:  d.num.Start(d.cur),
			Dur:  int64(d.cur.BlockSize),
			Sync: true,
		},
	}
	d.off, d.cur, d.valid = end, next, nextOK
	return nil
}

// SeekSample repositions to the frame containing the target sample and returns that frame's first sample; format.Media pre-rolls the remainder.
func (d *Demuxer) SeekSample(track int, sample int64) (int64, error) {
	if track != 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf("flacn: no track %d", track))
	}
	if sample < 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, "flacn: negative seek target")
	}
	if d.empty {
		return 0, nil
	}

	off, fi, ok := d.seekTableHint(sample)
	if ok {
		d.off, d.cur, d.valid = off, fi, true
		saved := len(d.warnings)
		end, _, _, err := d.findEnd()
		if err != nil || end < 0 {
			if d.w.Err() != nil {
				return 0, d.w.Err()
			}
			d.warnings = d.warnings[:saved]
			ok = false
		}
	}
	if !ok {
		off, fi, ok = d.bisect(sample)
	}
	if !ok {
		if d.w.Err() != nil {
			return 0, d.w.Err()
		}
		return 0, malformed("cannot relocate any frame for seeking")
	}

	d.off, d.cur, d.valid = off, fi, true
	for d.num.Start(d.cur)+int64(d.cur.BlockSize) <= sample {
		d.w.Trim(d.off)
		end, next, nextOK, err := d.findEnd()
		if err != nil {
			return 0, err
		}
		if end < 0 || !nextOK {
			break
		}
		d.off, d.cur = end, next
	}
	return d.num.Start(d.cur), nil
}

func (d *Demuxer) seekTableHint(sample int64) (int64, flac.FrameInfo, bool) {
	lo, hi := 0, len(d.seekTable)
	for lo < hi {
		mid := (lo + hi) / 2
		if d.seekTable[mid].sample <= sample {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return 0, flac.FrameInfo{}, false
	}
	pt := d.seekTable[lo-1]
	off := d.firstFrame + pt.off
	if off < d.firstFrame || off >= d.w.DataEnd() {
		return 0, flac.FrameInfo{}, false
	}
	hdr := d.w.BytesAt(off, flac.MaxFrameHeaderLen)
	fi, err := flac.ParseFrameHeader(hdr)
	if err != nil || !d.consistent(fi) || d.num.Start(fi) > sample {
		return 0, flac.FrameInfo{}, false
	}
	return off, fi, true
}

func (d *Demuxer) bisect(sample int64) (int64, flac.FrameInfo, bool) {
	lo := d.firstFrame
	loFi, err := flac.ParseFrameHeader(d.w.BytesAt(lo, flac.MaxFrameHeaderLen))
	if err != nil || !d.consistent(loFi) {
		var ok bool
		lo, loFi, ok = d.nextCandidate(lo, lo+maxResync)
		if !ok {
			return 0, flac.FrameInfo{}, false
		}
	}
	if d.num.Start(loFi) > sample {
		return lo, loFi, true
	}
	hi := d.w.DataEnd()
	for hi-lo > seekWindow {
		mid := lo + (hi-lo)/2
		off, fi, ok := d.nextCandidate(mid, hi)
		if !ok || d.num.Start(fi) > sample {
			hi = mid
			continue
		}
		lo, loFi = off, fi
	}
	return lo, loFi, true
}
