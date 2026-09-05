package format

import (
	"fmt"
	"io"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

type media struct {
	info    *Info
	demux   container.Demuxer
	seeker  container.Seeker
	track   container.Track
	decoder codec.Decoder

	stashFn  func(*audio.Buffer) error
	sink     *audio.Buffer
	carry    *audio.Buffer
	carryOff int
	pos      int64
	discont  bool
	eof      bool
	closed   bool

	delay  int64
	skip   int64
	rawEnd int64
}

func newMedia(info *Info, demux container.Demuxer) (Media, error) {
	if len(info.Tracks) == 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "format: no audio tracks")
	}
	track := info.Default()
	dec, err := newDecoder(track)
	if err != nil {
		return nil, err
	}
	m := &media{info: info, demux: demux, track: track, decoder: dec,
		delay: track.Delay, skip: track.Delay, rawEnd: -1}
	if (track.SamplesExact || track.Delay > 0 || track.Padding > 0) && track.Samples >= 0 {
		m.rawEnd = track.Delay + track.Samples
	}
	m.stashFn = m.stash
	if s, ok := demux.(container.Seeker); ok {
		m.seeker = s
	}
	if ix, ok := demux.(container.Indexer); ok {
		return &indexableMedia{media: m, ix: ix}, nil
	}
	return m, nil
}

type indexableMedia struct {
	*media
	ix container.Indexer
}

func (m *indexableMedia) IndexSnapshot() []byte         { return m.ix.IndexSnapshot() }
func (m *indexableMedia) RestoreIndex(blob []byte) bool { return m.ix.RestoreIndex(blob) }

func (m *media) Info() *Info { return m.info }

func (m *media) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	audio.Put(m.carry)
	m.carry = nil
	if r, ok := m.decoder.(codec.Releaser); ok {
		r.Release()
	}
	if c, ok := m.demux.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// ReadChunk fills dst to capacity from the decoded stream.
func (m *media) ReadChunk(dst *audio.Buffer) error {
	if m.closed {
		return waxerr.New(waxerr.CodeInternal, "format: ReadChunk on closed media")
	}
	if dst.Fmt != m.track.Fmt {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("format: chunk buffer is %v, track is %v", dst.Fmt, m.track.Fmt))
	}
	if dst.Cap() == 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, "format: zero-capacity chunk buffer")
	}
	if m.skip > 0 && !m.eof {
		dropped, err := m.discard(m.skip)
		if err != nil {
			return err
		}
		m.skip -= dropped
	}
	dst.N = 0
	m.copyOut(dst)
	for dst.N < dst.Cap() && !m.eof {
		if err := m.fill(dst); err != nil && err != io.EOF {
			return err
		}
	}
	if m.rawEnd >= 0 {
		if allowed := m.rawEnd - m.delay - m.pos; int64(dst.N) >= allowed {
			dst.N = int(max(allowed, 0))
			m.eof = true
		}
	}
	if dst.N == 0 {
		return io.EOF
	}
	dst.Pos = m.pos
	dst.Discont = m.discont
	m.discont = false
	m.pos += int64(dst.N)
	return nil
}

// SeekSample repositions to target.
func (m *media) SeekSample(target int64) (int64, error) {
	if m.closed {
		return 0, waxerr.New(waxerr.CodeInternal, "format: SeekSample on closed media")
	}
	if m.seeker == nil {
		return 0, waxerr.New(waxerr.CodeUnsupportedFormat, "format: source is not seekable")
	}
	if target < 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest, "format: negative seek target")
	}
	rawTarget := target + m.delay
	if m.rawEnd >= 0 {
		rawTarget = min(rawTarget, m.rawEnd)
	}
	landed, err := m.seeker.SeekSample(m.track.ID, rawTarget)
	if err != nil {
		return 0, err
	}
	m.decoder.Reset()
	m.carryOff = 0
	if m.carry != nil {
		m.carry.N = 0
	}
	m.eof = false
	m.skip = 0

	pos := landed
	if rawTarget > landed {
		dropped, err := m.discard(rawTarget - landed)
		if err != nil {
			return 0, err
		}
		pos += dropped
	}
	m.pos = max(pos-m.delay, 0)
	m.discont = true
	return m.pos, nil
}

func (m *media) discard(n int64) (int64, error) {
	var dropped int64
	for dropped < n {
		if m.carryLen() == 0 {
			err := m.fill(nil)
			if err == io.EOF {
				break
			}
			if err != nil {
				return dropped, err
			}
		}
		drop := int(min(int64(m.carryLen()), n-dropped))
		m.carryOff += drop
		dropped += int64(drop)
	}
	return dropped, nil
}

func (m *media) carryLen() int {
	if m.carry == nil {
		return 0
	}
	return m.carry.N - m.carryOff
}

func (m *media) copyOut(dst *audio.Buffer) {
	n := min(dst.Cap()-dst.N, m.carryLen())
	if n == 0 {
		return
	}
	audio.CopyFrames(dst, dst.N, m.carry, m.carryOff, n)
	dst.N += n
	m.carryOff += n
}

func (m *media) fill(dst *audio.Buffer) error {
	if m.eof {
		return io.EOF
	}
	m.sink = dst
	defer func() { m.sink = nil }()
	var pkt container.Packet
	for {
		err := m.demux.ReadPacket(&pkt)
		if err == io.EOF {
			m.eof = true
			return m.decoder.Drain(m.stashFn)
		}
		if err != nil {
			return err
		}
		if pkt.Track != m.track.ID {
			continue
		}
		return m.decoder.Decode(pkt.Data, m.stashFn)
	}
}

func (m *media) stash(b *audio.Buffer) error {
	if b.N == 0 {
		return nil
	}
	if b.Fmt != m.track.Fmt {
		return waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("format: decoder emitted %v for a %v track", b.Fmt, m.track.Fmt))
	}
	off := 0
	if m.sink != nil {
		take := min(m.sink.Cap()-m.sink.N, b.N)
		if take > 0 {
			audio.CopyFrames(m.sink, m.sink.N, b, 0, take)
			m.sink.N += take
			off = take
		}
	}
	rest := b.N - off
	if rest == 0 {
		return nil
	}
	m.compact()
	if m.carry == nil || m.carry.Cap()-m.carry.N < rest {
		grown := audio.Get(m.track.Fmt, max(m.carryLen()+rest, audio.StandardChunk))
		if old := m.carry; old != nil {
			grown.N = m.carryLen()
			audio.CopyFrames(grown, 0, old, m.carryOff, grown.N)
			audio.Put(old)
		}
		m.carry = grown
		m.carryOff = 0
	}
	audio.CopyFrames(m.carry, m.carry.N, b, off, rest)
	m.carry.N += rest
	return nil
}

func (m *media) compact() {
	if m.carry == nil || m.carryOff == 0 {
		return
	}
	n := m.carryLen()
	audio.CopyFrames(m.carry, 0, m.carry, m.carryOff, n)
	m.carry.N = n
	m.carryOff = 0
}
