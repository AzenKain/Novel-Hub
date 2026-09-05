package waxflow

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/mp4"
	"novelhub/pkg/waxflow/dsp"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

// DefaultSegmentSeconds is the HLS target segment duration when a request names none, per the Apple authoring guidance for low-latency-enough audio streaming without playlist bloat.
const DefaultSegmentSeconds = 4.0

const maxSegmentSeconds = 60.0

const primeSeconds = 0.1

func snapSegmentSamples(segSeconds float64, rate, grid int) (int, error) {
	switch {
	case segSeconds == 0:
		segSeconds = DefaultSegmentSeconds
	case segSeconds < 0 || segSeconds > maxSegmentSeconds:
		return 0, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: segment duration %g outside 0..%g seconds", segSeconds, maxSegmentSeconds))
	}
	if grid <= 0 {
		return 0, waxerr.New(waxerr.CodeInternal, "waxflow: segment grid must be positive")
	}
	return max(int(segSeconds*float64(rate)/float64(grid)+0.5), 1) * grid, nil
}

// SegmentPlan describes the segmented CMAF (HLS) form of a transcode, computed from headers alone like TranscodePlan (which it embeds: the embedded Versions already carry the segmenter revision, so an HLS cache key derives from it directly).
type SegmentPlan struct {
	TranscodePlan
	SegmentSamples int
	Delay          int64
	// Codecs is the RFC 6381 CODECS attribute for master playlists.
	Codecs             string
	Bandwidth          int
	TotalDecodeSamples int64
	Segments           int64
}

// SegmentDuration returns segment n's decode duration in samples, or -1 when n is out of range or the total is unknown.
func (p *SegmentPlan) SegmentDuration(n int64) int64 {
	if p.Segments < 0 || n < 0 || n >= p.Segments {
		return -1
	}
	if n < p.Segments-1 {
		return int64(p.SegmentSamples)
	}
	return p.TotalDecodeSamples - (p.Segments-1)*int64(p.SegmentSamples)
}

// PresentationDuration returns segment n's playable duration in samples: its decode span intersected with the presentation window the init segment's edit list declares, [Delay, Delay+Samples).
func (p *SegmentPlan) PresentationDuration(n int64) int64 {
	d := p.SegmentDuration(n)
	if d < 0 {
		return -1
	}
	start := n * int64(p.SegmentSamples)
	return max(0, min(start+d, p.Delay+p.Samples)-max(start, p.Delay))
}

// PlanSegments plans the segmented form of a transcode of track.
func (e *Engine) PlanSegments(track container.Track, opts TranscodeOptions, segSeconds float64) (*SegmentPlan, error) {
	if opts.FromSample != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: segmented transcodes have no FromSample; segments address time")
	}
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	if row.hls == nil {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("waxflow: %s has no segmented (HLS) form (available: %s)", opts.Format, strings.Join(SegmentedFormats(), ", ")))
	}
	plan, err := e.PlanTranscode(track, opts)
	if err != nil {
		return nil, err
	}
	if plan.FrameSize <= 0 {
		return nil, waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("waxflow: %s registered an HLS form without a native frame size", opts.Format))
	}
	segSamples, err := snapSegmentSamples(segSeconds, plan.Format.Rate, plan.FrameSize)
	if err != nil {
		return nil, err
	}
	if row.hls.delay > 0 && int64(segSamples) <= row.hls.delay {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: segment duration %g s is %d samples, no longer than %s's %d-sample encoder priming; "+
				"the first segment would present nothing",
				segSeconds, segSamples, opts.Format, row.hls.delay))
	}
	sp := &SegmentPlan{
		TranscodePlan:      *plan,
		SegmentSamples:     segSamples,
		Delay:              row.hls.delay,
		Codecs:             row.hls.codecs,
		TotalDecodeSamples: -1,
		Segments:           -1,
	}
	sp.Versions = append(append([]string{}, plan.Versions...), mp4.SegmenterVersion)

	if plan.Samples >= 0 {
		sp.TotalDecodeSamples = totalDecodeSamples(plan.Samples, row.hls.delay, int64(plan.FrameSize))
		sp.Segments = (sp.TotalDecodeSamples + int64(segSamples) - 1) / int64(segSamples)
	}

	base := plan.BitRate
	if base == 0 {
		base = plan.Format.Rate * plan.Format.Channels * max(plan.Format.BitDepth, 16)
	}
	sp.Bandwidth = base + plan.Format.Rate/plan.FrameSize*64 + 2000
	return sp, nil
}

// PlanSegmentsTimeline plans the segmented form of a concatenated timeline from its members' tracks alone (no decode, no open), exactly as PlanSegments does for one track: it is the same plan, over the synthetic track ConcatTrack computes, with the versions the synthetic track cannot name prepended.
func (e *Engine) PlanSegmentsTimeline(tracks []container.Track, copts ConcatOptions,
	opts TranscodeOptions, segSeconds float64) (*SegmentPlan, error) {
	env, err := ConcatTrack(tracks, copts)
	if err != nil {
		return nil, err
	}
	plan, err := e.PlanSegments(env, opts, segSeconds)
	if err != nil {
		return nil, err
	}
	extra, err := timelineVersions(tracks, env.Fmt, copts)
	if err != nil {
		return nil, err
	}
	plan.Versions = append(extra, plan.Versions...)
	return plan, nil
}

func timelineVersions(tracks []container.Track, env audio.Format, copts ConcatOptions) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if copts.Crossfade != 0 {
		add(crossfadeVersion(copts.Crossfade))
	}
	normalized := map[audio.Format]bool{}
	for _, t := range tracks {
		add(decodeVersion(t.Codec))
		if t.Fmt == env || normalized[t.Fmt] {
			continue
		}
		normalized[t.Fmt] = true
		chain, err := dsp.NewChain(dsp.NewSource(eofReader{}, t.Fmt), concatSpec(env, copts))
		if err != nil {
			return nil, err
		}
		for _, v := range chain.Versions() {
			add(v)
		}
		chain.Release()
	}
	return out, nil
}

func crossfadeVersion(x int64) string { return fmt.Sprintf("xfade-%d-1", x) }

func primeStarts(p0 int64, rate, frame int, horizon time.Duration, floor int64) (pChain, pEnc int64) {
	roundUp := func(n int64) int64 { return (n + int64(frame) - 1) / int64(frame) * int64(frame) }
	encPrime := roundUp(int64(float64(rate) * primeSeconds))
	chainPrime := encPrime + roundUp(int64(horizon.Seconds()*float64(rate)))
	floor = floor / int64(frame) * int64(frame)
	return max(floor, p0-chainPrime), max(0, p0-encPrime)
}

func headroomFloor(med format.Media, chain *dsp.Chain) int64 {
	h, ok := med.(Headroomer)
	if !ok {
		return 0
	}
	room := h.Headroom() - headroomMargin
	if room <= 0 {
		return 0
	}
	l, m := chain.Ratio()
	return -(room * int64(l) / int64(m))
}

const headroomMargin = 2

func floorDiv(a, b int64) int64 {
	q, r := a/b, a%b
	if r < 0 {
		q--
	}
	return q
}

func totalDecodeSamples(samples, delay, frame int64) int64 {
	if delay == 0 {
		return samples
	}
	return (samples + delay + frame - 1) / frame * frame
}

// SegmentedFormats lists the output formats with a segmented (HLS) form, in table order.
func SegmentedFormats() []string {
	var names []string
	for _, o := range outputs {
		if o.hls != nil {
			names = append(names, o.name)
		}
	}
	return names
}

// InitSegment builds the CMAF init header for a planned segmented transcode: the ftyp+moov all the plan's media segments share.
func (e *Engine) InitSegment(plan *SegmentPlan, opts TranscodeOptions) ([]byte, error) {
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	if row.hls == nil {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("waxflow: %s has no segmented (HLS) form", opts.Format))
	}
	enc, err := row.hls.encode(plan.Format, opts, 0)
	if err != nil {
		return nil, err
	}
	return mp4.InitSegment(container.Track{
		Codec:       row.codecID,
		CodecConfig: enc.CodecConfig(),
		Fmt:         plan.Format,
		Samples:     plan.Samples,
		Delay:       plan.Delay,
		Default:     true,
	})
}

// SegmentedOptions selects which slice of the segment sequence a TranscodeSegments run produces.
type SegmentedOptions struct {
	SegmentSamples int
	StartSegment   int64
}

// SegmentedResult reports what a TranscodeSegments run produced.
type SegmentedResult struct {
	Samples  int64
	Segments int64
}

// TranscodeSegments decodes src and emits numbered CMAF media segments: the variant-worker back end of HLS delivery.
func (e *Engine) TranscodeSegments(ctx context.Context, src container.Source, hint string, opts TranscodeOptions,
	segOpts SegmentedOptions, emit func(mp4.Segment) error) (*SegmentedResult, error) {
	if _, err := segmentOutputRow(opts); err != nil {
		return nil, err
	}
	med, err := e.OpenStream(src, hint)
	if err != nil {
		return nil, err
	}
	defer med.Close()
	return e.TranscodeSegmentsMedia(ctx, med, opts, segOpts, emit)
}

func segmentOutputRow(opts TranscodeOptions) (*output, error) {
	if opts.FromSample != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: segmented transcodes have no FromSample")
	}
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	if row.hls == nil {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("waxflow: %s has no segmented (HLS) form", opts.Format))
	}
	return row, nil
}

// TranscodeSegmentsMedia emits numbered CMAF media segments from an already-opened Media: the seam TranscodeSegments is built on, for inputs that are not a single sniffable Source (a concatenated album timeline).
func (e *Engine) TranscodeSegmentsMedia(ctx context.Context, med format.Media, opts TranscodeOptions,
	segOpts SegmentedOptions, emit func(mp4.Segment) error) (*SegmentedResult, error) {
	row, err := segmentOutputRow(opts)
	if err != nil {
		return nil, err
	}
	srcTrack := med.Info().Default()

	spec := specFor(opts)
	if row.adjust != nil {
		row.adjust(&spec, srcTrack.Fmt, opts)
	}
	frame := spec.FrameSize
	if frame <= 0 {
		return nil, waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("waxflow: %s registered an HLS form without a native frame size", opts.Format))
	}
	spec.FrameSize = 0
	switch {
	case segOpts.SegmentSamples <= 0 || segOpts.SegmentSamples%frame != 0:
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: segment length %d is not a positive multiple of the %d-sample frame", segOpts.SegmentSamples, frame))
	case segOpts.StartSegment < 0:
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: negative StartSegment")
	case segOpts.StartSegment > (1<<62)/int64(segOpts.SegmentSamples):
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: StartSegment overflows the sample timeline")
	}

	chain, err := dsp.NewChain(dsp.NewSource(med, srcTrack.Fmt), spec)
	if err != nil {
		return nil, err
	}
	defer chain.Release()
	f := chain.Format()
	e.logImplicitDownmix(opts, srcTrack.Fmt, f)

	p0 := segOpts.StartSegment * int64(segOpts.SegmentSamples)
	floor := headroomFloor(med, chain)
	pChain, pEnc := primeStarts(p0, f.Rate, frame, chain.Horizon(), floor)

	if pChain != 0 {
		srcPos := pChain
		if l, m := chain.Ratio(); l != m {
			srcPos = floorDiv(pChain*int64(m), int64(l)) - 1
		}
		if _, err := med.SeekSample(srcPos); err != nil {
			return nil, err
		}
	}

	enc, err := row.hls.encode(f, opts, pEnc)
	if err != nil {
		return nil, err
	}
	seg, err := mp4.NewSegmenter(container.Track{
		Codec:       row.codecID,
		CodecConfig: enc.CodecConfig(),
		Fmt:         f,
		Samples:     -1,
		Delay:       row.hls.delay,
		Default:     true,
	}, &mp4.SegmenterOptions{SegmentSamples: segOpts.SegmentSamples, StartSegment: segOpts.StartSegment})
	if err != nil {
		return nil, err
	}

	res := &SegmentedResult{}
	emitSeg := func(s mp4.Segment) error {
		res.Segments++
		return emit(s)
	}
	discard := (p0 - pEnc) / int64(frame)
	pkts := int64(0)
	emitPkt := func(p codec.Packet) error {
		pkts++
		if pkts <= discard {
			return nil
		}
		return seg.WritePacket(p, emitSeg)
	}

	e.log.Debug("segmented transcode started",
		"container", med.Info().Container, "source", srcTrack.Fmt.String(), "format", f.String(),
		"out", opts.Format, "segment", segOpts.StartSegment, "segSamples", segOpts.SegmentSamples,
		"dsp", strings.Join(chain.Versions(), ","))

	buf := audio.Get(f, audio.StandardChunk)
	defer audio.Put(buf)
	stage := audio.Get(f, frame)
	defer audio.Put(stage)

	skip := int64(-1)
	for {
		if err := ctx.Err(); err != nil {
			return nil, waxerr.Wrap(waxerr.CodeCanceled, "segmented transcode canceled", err)
		}
		err := chain.ReadChunk(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if skip < 0 {
			skip = pEnc - buf.Pos
			if skip < 0 {
				return nil, waxerr.New(waxerr.CodeInternal,
					fmt.Sprintf("waxflow: chain landed at %d, past the priming start %d", buf.Pos, pEnc))
			}
		}
		from := 0
		if skip > 0 {
			n := min(skip, int64(buf.N))
			skip -= n
			from = int(n)
		}
		if err := stageFrames(stage, buf, from, frame, enc, emitPkt); err != nil {
			return nil, err
		}
	}
	if stage.N > 0 {
		if err := enc.Encode(stage, emitPkt); err != nil {
			return nil, err
		}
		stage.N = 0
	}
	trailer, err := enc.Finish(emitPkt)
	if err != nil {
		return nil, err
	}
	if err := seg.End(emitSeg); err != nil {
		return nil, err
	}
	res.Samples = pEnc + trailer.Samples
	e.log.Debug("segmented transcode finished", "samples", res.Samples, "segments", res.Segments)
	return res, nil
}

func stageFrames(stage, src *audio.Buffer, from, frame int, enc codec.Encoder, emit func(codec.Packet) error) error {
	for from < src.N {
		n := min(frame-stage.N, src.N-from)
		for c := 0; c < stage.Fmt.Channels; c++ {
			if stage.I != nil {
				copy(stage.I[c*stage.Stride+stage.N:c*stage.Stride+stage.N+n], src.I[c*src.Stride+from:c*src.Stride+from+n])
			} else {
				copy(stage.F[c*stage.Stride+stage.N:c*stage.Stride+stage.N+n], src.F[c*src.Stride+from:c*src.Stride+from+n])
			}
		}
		if stage.N == 0 {
			stage.Pos = src.Pos + int64(from)
		}
		stage.N += n
		from += n
		if stage.N == frame {
			if err := enc.Encode(stage, emit); err != nil {
				return err
			}
			stage.N = 0
		}
	}
	return nil
}
