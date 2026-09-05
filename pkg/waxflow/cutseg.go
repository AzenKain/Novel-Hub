package waxflow

import (
	"context"
	"fmt"

	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/mp4"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

// CutSegmentPlan describes the segmented (CMAF) form of a cut, as CutPlan describes its progressive form.
type CutSegmentPlan struct {
	RemuxSegmentPlan
	Landed        []Span
	Grid          int
	SourceSamples int64
}

// PlanCutSegments plans the segmented form of a cut: the HLS spelling of the cut rung, and the mirror of PlanRemuxSegments over a synthesized cut track.
func (e *Engine) PlanCutSegments(track container.Track, opts TranscodeOptions, spans []Span,
	grid int, segSeconds float64) (*CutSegmentPlan, error) {
	cutTrack, landed, err := CutTrack(track, spans, grid)
	if err != nil {
		if waxerr.CodeOf(err) == waxerr.CodeUnsupportedFormat {
			e.log.Debug("segmented cut declined", "codec", track.Codec, "grid", grid, "reason", err)
			return nil, nil
		}
		return nil, err
	}
	rsp, err := e.PlanRemuxSegments(cutTrack, opts, segSeconds, grid)
	if err != nil || rsp == nil {
		return nil, err
	}
	if !cutTrimsExpressible(rsp.Container, cutTrack.Delay, cutTrack.Padding) {
		e.log.Debug("segmented cut declined", "reason", "the destination cannot signal the cut's trims",
			"outContainer", rsp.Container, "delay", cutTrack.Delay, "padding", cutTrack.Padding)
		return nil, nil
	}
	rsp.Versions = []string{RemuxVersion, mp4.SegmenterVersion, CutVersion}
	return &CutSegmentPlan{
		RemuxSegmentPlan: *rsp,
		Landed:           landed,
		Grid:             grid,
		SourceSamples:    track.Samples,
	}, nil
}

// CutInitSegment builds the CMAF init header for a planned segmented cut.
func (e *Engine) CutInitSegment(plan *CutSegmentPlan) ([]byte, error) {
	return e.RemuxInitSegment(&plan.RemuxSegmentPlan)
}

// CutSegments emits numbered CMAF media segments from a span of src's own packets: the run half of the segmented cut rung, and the segmented sibling of CutStream.
func (e *Engine) CutSegments(ctx context.Context, src container.Source, hint string, opts TranscodeOptions,
	spans []Span, grid int, samples int64, segOpts SegmentedOptions, emit func(mp4.Segment) error) (*SegmentedResult, error) {
	if err := validateSegOpts(segOpts); err != nil {
		return nil, err
	}
	demux, info, err := format.OpenDemuxer(src, hint, nil)
	if err != nil {
		return nil, err
	}
	track := info.Default()
	if samples >= 0 {
		track.Samples, track.SamplesExact = samples, true
	}
	cutTrack, _, err := CutTrack(track, spans, grid)
	if err != nil {
		return nil, err
	}
	rp, err := e.PlanRemux(cutTrack, opts)
	if err != nil {
		return nil, err
	}
	if rp == nil {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: a %s cut cannot be remuxed to %s with these options; transcode it",
				track.Codec, opts.Format))
	}
	cutDemux, err := cutSeekable(demux, track, spans, grid)
	if err != nil {
		return nil, err
	}
	e.log.Debug("segmented cut started", "container", info.Container, "codec", track.Codec,
		"out", opts.Format, "segment", segOpts.StartSegment, "segSamples", segOpts.SegmentSamples)
	res, err := e.segmentWalk(ctx, cutDemux, rp.Track, cutTrack.ID, segOpts, emit)
	if err != nil {
		return nil, err
	}
	e.log.Debug("segmented cut finished", "samples", res.Samples, "segments", res.Segments)
	return res, nil
}

type cutSeekDemuxer struct {
	*cutDemuxer
	seek container.Seeker
}

func cutSeekable(demux container.Demuxer, track container.Track, spans []Span, grid int) (container.Demuxer, error) {
	sk, ok := demux.(container.Seeker)
	if !ok {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			"waxflow: this container cannot seek, so a mid-stream segmented cut cannot start")
	}
	res, err := computeCut(track, spans, grid)
	if err != nil {
		return nil, err
	}
	return &cutSeekDemuxer{
		cutDemuxer: &cutDemuxer{Demuxer: demux, cut: res.track, track: track.ID, windows: res.windows},
		seek:       sk,
	}, nil
}

// SeekSample repositions the cut to outTarget on its own contiguous output timeline, so a restarted segment worker resumes exactly where a continuous run would be.
func (c *cutSeekDemuxer) SeekSample(track int, outTarget int64) (int64, error) {
	if track != c.track {
		return 0, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: cut view has no track %d", track))
	}
	if outTarget < 0 {
		return 0, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: negative cut seek target %d", outTarget))
	}
	var outStart int64
	i := 0
	for ; i < len(c.windows); i++ {
		w := c.windows[i]
		if w.to < 0 {
			break
		}
		wlen := w.to - w.from
		if outTarget < outStart+wlen {
			break
		}
		outStart += wlen
	}
	if i == len(c.windows) {
		c.cur, c.pos, c.out = len(c.windows), 0, outTarget
		return outTarget, nil
	}
	w := c.windows[i]
	landed, err := c.seek.SeekSample(c.track, w.from+(outTarget-outStart))
	if err != nil {
		return 0, err
	}
	c.cur, c.pos = i, landed
	c.out = outStart + max(0, landed-w.from)
	return c.out, nil
}
