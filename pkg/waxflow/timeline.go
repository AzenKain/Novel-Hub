package waxflow

import (
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/dsp"
	"novelhub/pkg/waxflow/dsp/dither"
	"novelhub/pkg/waxflow/dsp/resample"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

const concatContainer = "timeline"

// ToEnd is Slice's open-ended upper bound: the span runs to the end of the source.
const ToEnd = -1

// Slice bounds med to the sample range [from, to) of its own timeline, as a Media whose sample 0 is med's sample from and whose length is to-from.
func Slice(med format.Media, from, to int64) (format.Media, error) {
	if med == nil {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: Slice of a nil Media")
	}
	track := med.Info().Default()
	spanned, err := SpanTrack(track, from, to)
	if err != nil {
		return nil, err
	}
	s := &slice{med: med, from: from, limit: ToEnd, fmt: track.Fmt}
	if to >= 0 {
		s.limit = to - from
	}
	in := med.Info()
	s.info = &format.Info{
		Container: in.Container,
		Tracks:    []container.Track{spanned},
		Chapters:  spanChapters(in.Chapters, from, s.limit, track.Fmt.Rate),
		Warnings:  in.Warnings,
	}
	return s, nil
}

func spanChapters(chapters []container.Chapter, from, limit int64, rate int) []container.Chapter {
	if len(chapters) == 0 || rate <= 0 {
		return nil
	}
	start := SampleTime(from, rate)
	end := time.Duration(-1)
	if limit >= 0 {
		end = SampleTime(from+limit, rate)
	}
	var out []container.Chapter
	for i, ch := range chapters {
		if end >= 0 && ch.Start >= end {
			continue
		}
		if e := chapterEnd(chapters, i); e >= 0 && e <= start {
			continue
		}
		ch.Start = max(ch.Start-start, 0)
		if ch.End > 0 {
			if end >= 0 {
				ch.End = min(ch.End, end)
			}
			ch.End -= start
		}
		out = append(out, ch)
	}
	return out
}

func chapterEnd(chapters []container.Chapter, i int) time.Duration {
	if e := chapters[i].End; e > 0 {
		return e
	}
	if i+1 < len(chapters) {
		return chapters[i+1].Start
	}
	return -1
}

// SampleTime is sample n's position on a stream's clock at rate.
func SampleTime(n int64, rate int) time.Duration {
	sec, rem := n/int64(rate), n%int64(rate)
	return time.Duration(sec)*time.Second + time.Duration(rem)*time.Second/time.Duration(rate)
}

// SpanTrack computes the track a Slice of track to [from, to) presents: the same format, the window's length, and no gapless trims.
func SpanTrack(track container.Track, from, to int64) (container.Track, error) {
	switch {
	case from < 0:
		return container.Track{}, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: negative span start %d", from))
	case to < ToEnd:
		return container.Track{}, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: span end %d: want a sample offset or %d for the end of the source", to, ToEnd))
	case to >= 0 && to < from:
		return container.Track{}, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: span [%d, %d) ends before it starts", from, to))
	}
	total := track.Samples

	if total >= 0 {
		if from > total {
			return container.Track{}, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span starts at sample %d, past the source's %d samples", from, total))
		}
		if to > total {
			return container.Track{}, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span ends at sample %d, past the source's %d samples", to, total))
		}
	}

	out := track
	switch {
	case to >= 0:
		out.Samples, out.SamplesExact = to-from, true
	case total >= 0:
		out.Samples = total - from
	}
	out.Delay, out.Padding = 0, 0
	return out, nil
}

// Headroomer is implemented by a Media that has real audio before its own sample 0, as a window onto a longer stream does.
type Headroomer interface {
	Headroom() int64
}

type slice struct {
	med   format.Media
	info  *format.Info
	fmt   audio.Format
	from  int64
	limit int64

	pos     int64
	started bool
	discont bool
	closed  bool
}

func (s *slice) Info() *format.Info { return s.info }

// Headroom is the audio ahead of the window: the samples between the inner media's start and this span's, plus whatever the inner media can itself reach back to.
func (s *slice) Headroom() int64 {
	if h, ok := s.med.(Headroomer); ok {
		return s.from + h.Headroom()
	}
	return s.from
}

func (s *slice) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.med.Close()
}

func (s *slice) ensureStart() error {
	if s.started {
		return nil
	}
	if s.from == 0 {
		s.started = true
		return nil
	}
	landed, err := s.med.SeekSample(s.from)
	if err != nil {
		return err
	}
	s.pos = max(landed-s.from, 0)
	s.started = true
	return nil
}

// ReadChunk fills dst from the window.
func (s *slice) ReadChunk(dst *audio.Buffer) error {
	switch {
	case s.closed:
		return waxerr.New(waxerr.CodeInternal, "waxflow: ReadChunk on a closed span")
	case dst.Fmt != s.fmt:
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: chunk buffer is %v, span is %v", dst.Fmt, s.fmt))
	case dst.Cap() == 0:
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: zero-capacity chunk buffer")
	}
	if err := s.ensureStart(); err != nil {
		return err
	}
	if s.limit >= 0 && s.pos >= s.limit {
		return io.EOF
	}
	dst.N = 0
	err := s.med.ReadChunk(dst)
	if err == io.EOF {
		return s.endOfSource()
	}
	if err != nil {
		return err
	}
	if dst.N == 0 {
		return waxerr.New(waxerr.CodeInternal,
			"waxflow: a span's source returned no frames and no error; io.EOF is the only empty answer")
	}
	if s.limit >= 0 {
		if allowed := s.limit - s.pos; int64(dst.N) >= allowed {
			dst.N = int(max(allowed, 0))
		}
	}
	dst.Pos = s.pos
	dst.Discont = s.discont
	s.discont = false
	s.pos += int64(dst.N)
	return nil
}

func (s *slice) endOfSource() error {
	if s.limit >= 0 && s.pos < s.limit {
		return waxerr.New(waxerr.CodeSourceUnreadable, fmt.Sprintf(
			"waxflow: the source ended %d samples into a span that declared %d; its cut points do not describe this file",
			s.pos, s.limit))
	}
	return io.EOF
}

// SeekSample repositions to target on the window's own timeline.
func (s *slice) SeekSample(target int64) (int64, error) {
	room := s.Headroom()
	switch {
	case s.closed:
		return 0, waxerr.New(waxerr.CodeInternal, "waxflow: SeekSample on a closed span")
	case target < -room:
		return 0, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
			"waxflow: seek to %d is %d samples before the source's start; the span has %d samples of headroom",
			target, -target-room, room))
	}
	if s.limit >= 0 {
		target = min(target, s.limit)
	}
	landed, err := s.med.SeekSample(s.from + target)
	if err != nil {
		return 0, err
	}
	s.started = true
	s.pos = landed - s.from
	if s.limit >= 0 {
		s.pos = min(s.pos, s.limit)
	}
	s.discont = true
	return s.pos, nil
}

// ConcatSource is one member of a timeline: its track, as Probe reported it, so a timeline can be planned without opening anything, and a function that opens it on demand.
type ConcatSource struct {
	Track container.Track
	Open  func() (format.Media, error)
}

// ConcatOptions configures a Concat.
type ConcatOptions struct {
	Profile resample.Profile

	Crossfade int64
}

const maxCrossfadeBytes = 16 << 20

func concatLayout(tracks []container.Track, opts ConcatOptions) (env audio.Format, lens, starts []int64, total int64, err error) {
	if len(tracks) == 0 {
		return audio.Format{}, nil, nil, 0, waxerr.New(waxerr.CodeInvalidRequest,
			"waxflow: a timeline needs at least one member")
	}
	env = audio.Format{Type: audio.Int}
	for i, t := range tracks {
		if err := t.Fmt.Valid(); err != nil {
			return audio.Format{}, nil, nil, 0, waxerr.Wrap(waxerr.CodeUnsupportedFormat,
				fmt.Sprintf("waxflow: timeline member %d", i), err)
		}
		if t.Samples < 0 {
			return audio.Format{}, nil, nil, 0, waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: timeline member %d has no declared length; measure it before planning a timeline", i))
		}
		env.Rate = max(env.Rate, t.Fmt.Rate)
		env.Channels = max(env.Channels, t.Fmt.Channels)
		env.BitDepth = max(env.BitDepth, t.Fmt.BitDepth)
		if t.Fmt.Type == audio.Float {
			env.Type = audio.Float
		}
	}
	if env.Type == audio.Float {
		env.BitDepth = 32
	}
	env.Layout = audio.DefaultLayout(env.Channels)

	for i, t := range tracks {
		if t.Fmt.Channels == env.Channels && t.Fmt.Layout != env.Layout {
			return audio.Format{}, nil, nil, 0, waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
				"waxflow: timeline member %d lays its %d channels out as %v, not the conventional %v; "+
					"a timeline normalizes channel counts, not speaker assignments",
				i, t.Fmt.Channels, t.Fmt.Layout, env.Layout))
		}
	}

	lens = make([]int64, len(tracks))
	starts = make([]int64, len(tracks)+1)
	for i, t := range tracks {
		lens[i] = concatMemberSamples(t, env)
		total += lens[i]
		tail := int64(0)
		if i < len(tracks)-1 {
			tail = opts.Crossfade
		}
		starts[i+1] = starts[i] + lens[i] - tail
	}
	if err := checkCrossfade(lens, env, opts.Crossfade); err != nil {
		return audio.Format{}, nil, nil, 0, err
	}
	total -= int64(len(tracks)-1) * opts.Crossfade
	return env, lens, starts, total, nil
}

// ConcatTrack computes the synthetic track a Concat of these members presents: the common (envelope) format, the summed normalized length, and no gapless trims.
func ConcatTrack(tracks []container.Track, opts ConcatOptions) (container.Track, error) {
	env, _, _, total, err := concatLayout(tracks, opts)
	if err != nil {
		return container.Track{}, err
	}
	return concatSynthetic(env, total), nil
}

func concatSynthetic(env audio.Format, total int64) container.Track {
	return container.Track{
		Codec:        codec.PCM,
		Fmt:          env,
		Samples:      total,
		SamplesExact: true,
		Default:      true,
	}
}

// MemberBoundary is one member's place on a concatenated timeline.
type MemberBoundary struct {
	OffsetSamples   int64 `json:"offsetSamples"`
	DurationSamples int64 `json:"durationSamples"`
}

// ConcatBoundaries reports where each member lands on the concatenated timeline and how long it is, from the members' headers alone (no decode, no open), plus the envelope format the offsets are measured on.
func ConcatBoundaries(tracks []container.Track, opts ConcatOptions) ([]MemberBoundary, audio.Format, error) {
	env, lens, starts, _, err := concatLayout(tracks, opts)
	if err != nil {
		return nil, audio.Format{}, err
	}
	bounds := make([]MemberBoundary, len(tracks))
	for i := range tracks {
		bounds[i] = MemberBoundary{OffsetSamples: starts[i], DurationSamples: lens[i]}
	}
	return bounds, env, nil
}

// CrossfadeSamples converts a crossfade expressed in seconds into the envelope samples ConcatOptions.Crossfade carries.
func CrossfadeSamples(tracks []container.Track, seconds float64) (int64, error) {
	if !(seconds > 0) {
		return 0, nil
	}
	env, _, _, _, err := concatLayout(tracks, ConcatOptions{})
	if err != nil {
		return 0, err
	}
	x := math.Round(seconds * float64(env.Rate))
	if x >= float64(math.MaxInt64) {
		return math.MaxInt64, nil
	}
	return int64(x), nil
}

func checkCrossfade(lens []int64, env audio.Format, x int64) error {
	if x == 0 {
		return nil
	}
	if x < 0 {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: negative crossfade %d", x))
	}
	if perFrame := int64(env.Channels) * 4; x > maxCrossfadeBytes/perFrame {
		limit := maxCrossfadeBytes / perFrame
		return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
			"waxflow: a crossfade of %d samples is more than this timeline can blend; the most it can is "+
				"%d samples (%.1f s at %d Hz, %d channels), which is the largest buffer the sample pool holds",
			x, limit, float64(limit)/float64(env.Rate), env.Rate, env.Channels))
	}
	for i, l := range lens {
		var need int64
		if i > 0 {
			need += x
		}
		if i < len(lens)-1 {
			need += x
		}
		if need > l {
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: timeline member %d is %d samples, too short for the %d samples of crossfade it carries "+
					"(a crossfade of %d, on %d of its seams)",
				i, l, need, x, need/x))
		}
	}
	return nil
}

func concatMemberSamples(t container.Track, env audio.Format) int64 {
	return resample.OutputLen(t.Samples, t.Fmt.Rate, env.Rate)
}

func concatSpec(env audio.Format, opts ConcatOptions) dsp.ChainSpec {
	spec := dsp.ChainSpec{
		Rate:     env.Rate,
		Channels: env.Channels,
		Profile:  opts.Profile,
		Shaping:  dither.TPDF,
	}
	if env.Type == audio.Float {
		spec.Float = true
	} else {
		spec.BitDepth = env.BitDepth
	}
	return spec
}

// Concat sequences members into one gapless format.Media: a single continuous timeline whose sample len(a) is b's sample 0, exactly, unless opts.Crossfade asks for a blend.
func Concat(members []ConcatSource, opts ConcatOptions) (format.Media, error) {
	tracks := make([]container.Track, len(members))
	for i := range members {
		if members[i].Open == nil {
			return nil, waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: timeline member %d has no Open function", i))
		}
		tracks[i] = members[i].Track
	}
	env, lens, starts, total, err := concatLayout(tracks, opts)
	if err != nil {
		return nil, err
	}
	return &concat{
		members: members,
		tracks:  tracks,
		opts:    opts,
		fmt:     env,
		starts:  starts,
		lens:    lens,
		info:    &format.Info{Container: concatContainer, Tracks: []container.Track{concatSynthetic(env, total)}},
	}, nil
}

type concat struct {
	members []ConcatSource
	tracks  []container.Track
	opts    ConcatOptions
	info    *format.Info
	fmt     audio.Format
	starts  []int64
	lens    []int64

	cur   int
	med   format.Media
	chain *dsp.Chain

	local        int64
	pos          int64
	discont      bool
	closed       bool
	blend        *audio.Buffer
	blendOff     int
	unpositioned bool
}

func (c *concat) Info() *format.Info { return c.info }

// Members reports the members' tracks (format.Composite).
func (c *concat) Members() []container.Track {
	return append([]container.Track(nil), c.tracks...)
}

func (c *concat) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	c.releaseBlend()
	return c.closeMember()
}

// ReadChunk fills dst from the timeline, crossing member boundaries.
func (c *concat) ReadChunk(dst *audio.Buffer) error {
	switch {
	case c.closed:
		return waxerr.New(waxerr.CodeInternal, "waxflow: ReadChunk on a closed timeline")
	case c.unpositioned:
		return waxerr.New(waxerr.CodeInternal,
			"waxflow: reading a timeline whose seek failed; its position is unknown until a seek succeeds")
	case dst.Fmt != c.fmt:
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: chunk buffer is %v, timeline is %v", dst.Fmt, c.fmt))
	case dst.Cap() == 0:
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: zero-capacity chunk buffer")
	}
	if err := c.fill(dst); err != nil {
		return err
	}
	dst.Pos = c.pos
	dst.Discont = c.discont
	c.discont = false
	c.pos += int64(dst.N)
	return nil
}

func (c *concat) fill(dst *audio.Buffer) error {
	for c.cur < len(c.members) {
		if c.med == nil {
			if err := c.open(c.cur); err != nil {
				return err
			}
		}
		n := c.bound()
		if n == 0 {
			if err := c.captureTail(); err != nil {
				return err
			}
			continue
		}
		dst.N = 0
		err := c.readBounded(dst, n)
		if err == io.EOF {
			if err := c.advance(); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err := c.count(dst.N); err != nil {
			return err
		}
		if c.blend != nil {
			c.mixBlend(dst)
		}
		return nil
	}
	return io.EOF
}

// SeekSample repositions to target on the timeline: the member holding it is opened and positioned, and everything after it follows in order.
func (c *concat) SeekSample(target int64) (int64, error) {
	switch {
	case c.closed:
		return 0, waxerr.New(waxerr.CodeInternal, "waxflow: SeekSample on a closed timeline")
	case target < 0:
		return 0, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: negative seek target")
	}
	pos, err := c.seekTo(target)
	if err != nil {
		c.closeMember()
		c.releaseBlend()
		c.unpositioned = true
		return 0, err
	}
	c.unpositioned = false
	return pos, nil
}

func (c *concat) seekTo(target int64) (int64, error) {
	c.releaseBlend()
	total := c.starts[len(c.members)]
	if target >= total {
		if err := c.closeMember(); err != nil {
			return 0, err
		}
		c.cur = len(c.members)
		c.local, c.pos, c.discont = 0, total, true
		return total, nil
	}
	i := c.memberAt(target)
	off := target - c.starts[i]
	if i > 0 && off < c.opts.Crossfade {
		return c.seekIntoBlend(i, off)
	}
	if c.med == nil || c.cur != i {
		if err := c.closeMember(); err != nil {
			return 0, err
		}
		if err := c.open(i); err != nil {
			return 0, err
		}
	} else if err := c.buildChain(); err != nil {
		return 0, err
	}
	landed, err := c.seekBody(off)
	if err != nil {
		return 0, err
	}
	c.local = landed
	c.pos = c.starts[i] + landed
	c.discont = true
	return c.pos, nil
}

func (c *concat) seekBody(local int64) (int64, error) {
	landed, err := c.seekMember(local)
	if err != nil {
		return 0, err
	}
	if body := c.lens[c.cur] - c.tailOf(c.cur); landed > body {
		return 0, waxerr.New(waxerr.CodeSourceUnreadable, fmt.Sprintf(
			"waxflow: timeline member %d could not be positioned before its crossfade zone: "+
				"a seek to %d landed at %d, past the %d samples that are the member's own",
			c.cur, local, landed, body))
	}
	return landed, nil
}

func (c *concat) seekIntoBlend(i int, off int64) (int64, error) {
	if err := c.closeMember(); err != nil {
		return 0, err
	}
	if err := c.open(i - 1); err != nil {
		return 0, err
	}
	landed, err := c.seekBody(c.lens[i-1] - c.opts.Crossfade)
	if err != nil {
		return 0, err
	}
	c.local = landed
	if err := c.captureTail(); err != nil {
		return 0, err
	}
	if err := c.preRoll(off); err != nil {
		return 0, err
	}
	c.pos = c.starts[i] + off
	c.discont = true
	return c.pos, nil
}

func (c *concat) preRoll(n int64) error {
	for n > 0 {
		buf := audio.Get(c.fmt, int(min(n, int64(audio.StandardChunk))))
		err := c.fill(buf)
		got := int64(buf.N)
		audio.Put(buf)
		switch {
		case err == io.EOF:
			return waxerr.New(waxerr.CodeInternal,
				fmt.Sprintf("waxflow: timeline member %d ended inside a crossfade seek pre-roll", c.cur))
		case err != nil:
			return err
		}
		n -= got
	}
	return nil
}

func (c *concat) memberAt(target int64) int {
	return sort.Search(len(c.members), func(i int) bool { return target < c.starts[i+1] })
}

func (c *concat) open(i int) error {
	med, err := c.members[i].Open()
	if err != nil {
		return err
	}
	if got := med.Info().Default().Fmt; got != c.tracks[i].Fmt {
		med.Close()
		return waxerr.New(waxerr.CodeSourceChanged, fmt.Sprintf(
			"waxflow: timeline member %d opened as %v, its headers declared %v", i, got, c.tracks[i].Fmt))
	}
	c.med, c.cur, c.local = med, i, 0
	if err := c.buildChain(); err != nil {
		c.closeMember()
		return err
	}
	return nil
}

func (c *concat) buildChain() error {
	if c.chain != nil {
		c.chain.Release()
		c.chain = nil
	}
	in := c.tracks[c.cur].Fmt
	if in == c.fmt {
		return nil
	}
	chain, err := dsp.NewChain(dsp.NewSource(c.med, in), concatSpec(c.fmt, c.opts))
	if err != nil {
		return err
	}
	c.chain = chain
	return nil
}

func (c *concat) readMember(dst *audio.Buffer) error {
	if c.chain != nil {
		return c.chain.ReadChunk(dst)
	}
	err := c.med.ReadChunk(dst)
	if err == nil && dst.N == 0 {
		return waxerr.New(waxerr.CodeInternal, fmt.Sprintf(
			"waxflow: timeline member %d returned no frames and no error; io.EOF is the only empty answer", c.cur))
	}
	return err
}

func (c *concat) tailOf(i int) int64 {
	if i >= len(c.members)-1 {
		return 0
	}
	return c.opts.Crossfade
}

func (c *concat) bound() int64 {
	if c.blend != nil {
		return int64(c.blend.N - c.blendOff)
	}
	tail := c.tailOf(c.cur)
	if tail == 0 {
		return -1
	}
	return c.lens[c.cur] - tail - c.local
}

func (c *concat) readBounded(dst *audio.Buffer, n int64) error {
	if n < 0 || n >= int64(dst.Cap()) {
		return c.readMember(dst)
	}
	buf := audio.Get(c.fmt, int(n))
	err := c.readMember(buf)
	if err == nil {
		audio.CopyFrames(dst, 0, buf, 0, buf.N)
		dst.N = buf.N
	}
	audio.Put(buf)
	return err
}

func (c *concat) captureTail() error {
	x := c.tailOf(c.cur)
	if want := c.lens[c.cur] - x; c.local != want {
		return waxerr.New(waxerr.CodeSourceUnreadable, fmt.Sprintf(
			"waxflow: timeline member %d is at sample %d, not the %d where its crossfade zone begins; "+
				"its seek landed somewhere the member's own headers say it should not have", c.cur, c.local, want))
	}
	blend := audio.Get(c.fmt, int(x))
	buf := audio.Get(c.fmt, audio.StandardChunk)
	defer audio.Put(buf)
	for {
		buf.N = 0
		err := c.readMember(buf)
		if err == io.EOF {
			break
		}
		if err == nil {
			err = c.count(buf.N)
		}
		if err != nil {
			audio.Put(blend)
			return err
		}
		audio.CopyFrames(blend, blend.N, buf, 0, buf.N)
		blend.N += buf.N
	}
	if err := c.advance(); err != nil {
		audio.Put(blend)
		return err
	}
	c.blend, c.blendOff = blend, 0
	return nil
}

func (c *concat) mixBlend(dst *audio.Buffer) {
	blendFrames(dst, c.blend, c.blendOff, int(c.opts.Crossfade))
	c.blendOff += dst.N
	if c.blendOff >= c.blend.N {
		c.releaseBlend()
	}
}

func (c *concat) releaseBlend() {
	if c.blend != nil {
		audio.Put(c.blend)
		c.blend = nil
	}
	c.blendOff = 0
}

func blendFrames(dst, out *audio.Buffer, off, x int) {
	n, chans := dst.N, dst.Fmt.Channels
	if dst.Fmt.Type == audio.Float {
		for k := 0; k < n; k++ {
			sin, cos := math.Sincos(float64(off+k) / float64(x) * (math.Pi / 2))
			gi, go_ := float32(sin), float32(cos)
			for ch := 0; ch < chans; ch++ {
				d, o := &dst.F[ch*dst.Stride+k], out.F[ch*out.Stride+off+k]
				*d = o*go_ + *d*gi
			}
		}
		return
	}
	scale := math.Ldexp(1, dst.Fmt.BitDepth-1)
	lo, hi := -scale, scale-1
	for k := 0; k < n; k++ {
		sin, cos := math.Sincos(float64(off+k) / float64(x) * (math.Pi / 2))
		for ch := 0; ch < chans; ch++ {
			d := &dst.I[ch*dst.Stride+k]
			v := math.Floor(float64(out.I[ch*out.Stride+off+k])*cos + float64(*d)*sin + 0.5)
			if v < lo {
				v = lo
			} else if v > hi {
				v = hi
			}
			*d = int32(v)
		}
	}
}

func (c *concat) seekMember(local int64) (int64, error) {
	if c.chain == nil {
		return c.med.SeekSample(local)
	}
	src := local
	if l, m := c.chain.Ratio(); l != m {
		src = local * int64(m) / int64(l)
		if src > 0 {
			src--
		}
	}
	landed, err := c.med.SeekSample(src)
	if err != nil {
		return 0, err
	}
	out := c.chain.OutputSamples(landed)
	if out >= local {
		return out, nil
	}
	if err := c.dropOutput(local - out); err != nil {
		return 0, err
	}
	return local, nil
}

func (c *concat) dropOutput(n int64) error {
	for n > 0 {
		buf := audio.Get(c.fmt, int(min(n, int64(audio.StandardChunk))))
		err := c.chain.ReadChunk(buf)
		got := int64(buf.N)
		audio.Put(buf)
		switch {
		case err == io.EOF:
			return waxerr.New(waxerr.CodeInternal,
				fmt.Sprintf("waxflow: timeline member %d ended inside a seek pre-roll", c.cur))
		case err != nil:
			return err
		}
		n -= got
	}
	return nil
}

func (c *concat) count(n int) error {
	c.local += int64(n)
	if want := c.lens[c.cur]; c.local > want {
		return waxerr.New(waxerr.CodeSourceUnreadable, fmt.Sprintf(
			"waxflow: timeline member %d holds more audio than the %d samples its headers declared", c.cur, want))
	}
	return nil
}

func (c *concat) advance() error {
	if want := c.lens[c.cur]; c.local != want {
		return waxerr.New(waxerr.CodeSourceUnreadable, fmt.Sprintf(
			"waxflow: timeline member %d delivered %d samples, its headers declared %d", c.cur, c.local, want))
	}
	if err := c.closeMember(); err != nil {
		return err
	}
	c.cur++
	c.local = 0
	return nil
}

func (c *concat) closeMember() error {
	if c.chain != nil {
		c.chain.Release()
		c.chain = nil
	}
	if c.med == nil {
		return nil
	}
	med := c.med
	c.med = nil
	return med.Close()
}
