package waxflow

import (
	"context"
	"fmt"
	"io"
	"strings"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/mp4"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

// RemuxVersion identifies the remux rung's own sample-affecting logic for the ADR-0004 cache key.
const RemuxVersion = "remux-2"

// RemuxPlan describes what a remux would produce, computed from the source track's headers alone.
type RemuxPlan struct {
	TranscodePlan
	Track container.Track
}

// PlanRemux reports whether opts can be served by rewriting track's container around its existing packets, and how.
func (e *Engine) PlanRemux(track container.Track, opts TranscodeOptions) (*RemuxPlan, error) {
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	containerName, mediaType, err := resolveContainer(row, opts.Container)
	if err != nil {
		return nil, err
	}
	if !codecSurvives(track.Codec, row.codecID) || !remuxable(opts, track.Fmt) || !gaplessSurvives(track) {
		return nil, nil
	}
	t := track
	t.ID, t.Default = 0, true
	return &RemuxPlan{
		TranscodePlan: TranscodePlan{
			Format:         track.Fmt,
			Container:      containerName,
			MediaType:      mediaType,
			Live:           containerLive(row.live, opts.Container),
			Versions:       []string{RemuxVersion},
			Samples:        track.Samples,
			EstimatedBytes: -1,
		},
		Track: t,
	}, nil
}

func codecSurvives(src, out codec.ID) bool {
	return src == out && src != codec.PCM
}

func remuxable(opts TranscodeOptions, src audio.Format) bool {
	if opts.FromSample != 0 {
		return false
	}
	norm := opts
	if norm.Rate == src.Rate {
		norm.Rate = 0
	}
	if norm.Channels == src.Channels {
		norm.Channels = 0
	}
	if src.Type == audio.Int && norm.BitDepth == src.BitDepth {
		norm.BitDepth = 0
	}
	base := TranscodeOptions{
		Format:          opts.Format,
		Container:       opts.Container,
		ResampleProfile: opts.ResampleProfile,
		Shaping:         opts.Shaping,
	}
	return planOptsOf(norm) == planOptsOf(base)
}

func remuxTrailer(t container.Track, decoded int64) codec.Trailer {
	tr := codec.Trailer{Samples: t.Samples, Delay: t.Delay, Padding: t.Padding}
	switch {
	case t.Samples < 0:
		tr.Samples = max(0, decoded-t.Delay-t.Padding)
	case t.Delay > 0:
		tr.Padding = max(0, decoded-t.Delay-t.Samples)
	}
	return tr
}

func gaplessSurvives(t container.Track) bool {
	if t.Delay == 0 && t.Padding == 0 {
		return true
	}
	return t.Codec != codec.FLAC && t.Codec != codec.ALAC
}

// PacketGrid reports the decode duration every packet of src's default track shares: the grid a segmented remux must lay its segment boundaries on.
func (e *Engine) PacketGrid(src container.Source, hint string) (int, error) {
	demux, info, err := format.OpenDemuxer(src, hint, nil)
	if err != nil {
		return 0, err
	}
	return packetGrid(demux, info.Default().ID)
}

func packetGrid(demux container.Demuxer, track int) (int, error) {
	grid, prev := int64(0), int64(0)
	have, uniform := false, true
	var pkt container.Packet
	for {
		err := demux.ReadPacket(&pkt)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if pkt.Track != track {
			continue
		}
		if have {
			switch {
			case prev <= 0:
				uniform = false
			case grid == 0:
				grid = prev
			case grid != prev:
				uniform = false
			}
		}
		prev, have = pkt.Dur, true
	}
	if !uniform || grid <= 0 {
		return 0, nil
	}
	return int(grid), nil
}

// RemuxSegmentPlan describes the segmented (CMAF) form of a remux, as SegmentPlan describes a transcode's.
type RemuxSegmentPlan struct {
	SegmentPlan
	Track container.Track
}

// PlanRemuxSegments plans the segmented form of a remux: the rung that carries WaxTap's motivating case, since format=opus already means "Ogg-Opus progressive, fMP4-Opus segmented" and so Opus-in-WebM to Opus-in-fMP4 is this with nothing invented.
func (e *Engine) PlanRemuxSegments(track container.Track, opts TranscodeOptions, segSeconds float64, grid int) (*RemuxSegmentPlan, error) {
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	if row.hls == nil {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("waxflow: %s has no segmented (HLS) form (available: %s)", opts.Format, strings.Join(SegmentedFormats(), ", ")))
	}
	rp, err := e.PlanRemux(track, opts)
	if err != nil || rp == nil {
		return nil, err
	}
	if grid <= 0 {
		return nil, nil
	}
	if _, err := mp4.InitSegment(rp.Track); err != nil {
		return nil, nil
	}
	segSamples, err := snapSegmentSamples(segSeconds, track.Fmt.Rate, grid)
	if err != nil {
		return nil, err
	}
	sp := SegmentPlan{
		TranscodePlan:      rp.TranscodePlan,
		SegmentSamples:     segSamples,
		Delay:              track.Delay,
		Codecs:             row.hls.codecs,
		TotalDecodeSamples: -1,
		Segments:           -1,
	}
	sp.Versions = []string{RemuxVersion, mp4.SegmenterVersion}
	sp.FrameSize = grid
	if track.Samples >= 0 {
		sp.TotalDecodeSamples = totalDecodeSamples(track.Samples, track.Delay, int64(grid))
		sp.Segments = (sp.TotalDecodeSamples + int64(segSamples) - 1) / int64(segSamples)
	}
	base := sp.Format.Rate * sp.Format.Channels * max(sp.Format.BitDepth, 16)
	sp.Bandwidth = base + sp.Format.Rate/grid*64 + 2000
	return &RemuxSegmentPlan{SegmentPlan: sp, Track: rp.Track}, nil
}

// RemuxInitSegment builds the CMAF init header for a planned segmented remux: the source's own sample entry, from the codec config it already carries, so the packets the segments hold and the header that describes them come from one place.
func (e *Engine) RemuxInitSegment(plan *RemuxSegmentPlan) ([]byte, error) {
	t := plan.Track
	t.Samples, t.Delay = plan.Samples, plan.Delay
	return mp4.InitSegment(t)
}

// RemuxSegments emits numbered CMAF media segments from src's own packets: the segmented form of the middle rung, and the back end of an HLS variant that needs no encoder.
func (e *Engine) RemuxSegments(ctx context.Context, src container.Source, hint string, opts TranscodeOptions,
	segOpts SegmentedOptions, emit func(mp4.Segment) error) (*SegmentedResult, error) {
	if err := validateSegOpts(segOpts); err != nil {
		return nil, err
	}
	demux, info, err := format.OpenDemuxer(src, hint, nil)
	if err != nil {
		return nil, err
	}
	track := info.Default()
	rp, err := e.PlanRemux(track, opts)
	if err != nil {
		return nil, err
	}
	if rp == nil {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: a %s source cannot be remuxed to %s with these options; transcode it",
				track.Codec, opts.Format))
	}
	e.log.Debug("segmented remux started", "container", info.Container, "codec", track.Codec,
		"out", opts.Format, "segment", segOpts.StartSegment, "segSamples", segOpts.SegmentSamples)
	res, err := e.segmentWalk(ctx, demux, rp.Track, track.ID, segOpts, emit)
	if err != nil {
		return nil, err
	}
	e.log.Debug("segmented remux finished", "samples", res.Samples, "segments", res.Segments)
	return res, nil
}

func validateSegOpts(segOpts SegmentedOptions) error {
	switch {
	case segOpts.SegmentSamples <= 0:
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: segmenter needs a positive SegmentSamples")
	case segOpts.StartSegment < 0:
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: negative StartSegment")
	case segOpts.StartSegment > (1<<62)/int64(segOpts.SegmentSamples):
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: StartSegment overflows the sample timeline")
	}
	return nil
}

func (e *Engine) segmentWalk(ctx context.Context, demux container.Demuxer, segTrack container.Track,
	walkTrackID int, segOpts SegmentedOptions, emit func(mp4.Segment) error) (*SegmentedResult, error) {
	seg, err := mp4.NewSegmenter(segTrack, &mp4.SegmenterOptions{
		SegmentSamples: segOpts.SegmentSamples, StartSegment: segOpts.StartSegment,
	})
	if err != nil {
		return nil, err
	}
	p0 := segOpts.StartSegment * int64(segOpts.SegmentSamples)
	pos, err := seekPackets(demux, walkTrackID, p0)
	if err != nil {
		return nil, err
	}

	res := &SegmentedResult{}
	emitSeg := func(s mp4.Segment) error {
		res.Segments++
		return emit(s)
	}

	grid := int64(0)
	var held codec.Packet
	var have bool
	var scratch []byte
	release := func(checked bool) error {
		if !have {
			return nil
		}
		if checked {
			switch {
			case grid == 0:
				grid = held.Dur
			case held.Dur != grid:
				return waxerr.New(waxerr.CodeUnsupportedFormat,
					fmt.Sprintf("waxflow: source packet of %d samples breaks the %d-sample grid; this stream cannot be segmented without re-encoding",
						held.Dur, grid))
			}
		}
		have = false
		return seg.WritePacket(held, emitSeg)
	}
	decoded, err := copyPackets(ctx, demux, walkTrackID, func(pkt container.Packet) error {
		if pos < p0 {
			pos += pkt.Dur
			return nil
		}
		if err := release(true); err != nil {
			return err
		}
		scratch = append(scratch[:0], pkt.Data...)
		held, have = pkt.Packet, true
		held.Data = scratch
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := release(false); err != nil {
		return nil, err
	}
	if err := seg.End(emitSeg); err != nil {
		return nil, err
	}
	res.Samples = decoded
	return res, nil
}

func seekPackets(demux container.Demuxer, track int, target int64) (int64, error) {
	if target == 0 {
		return 0, nil
	}
	sk, ok := demux.(container.Seeker)
	if !ok {
		return 0, waxerr.New(waxerr.CodeUnsupportedFormat,
			"waxflow: this container cannot seek, so a mid-stream segmented remux cannot start")
	}
	landed, err := sk.SeekSample(track, target)
	if err != nil {
		return 0, err
	}
	if landed > target {
		return 0, waxerr.New(waxerr.CodeInternal,
			fmt.Sprintf("waxflow: seek landed at %d, past the segment start %d", landed, target))
	}
	return landed, nil
}

// Remux rewrites src's container around its existing packets: the ladder's middle rung, and the one that makes "direct play, transmux, transcode" true.
func (e *Engine) Remux(ctx context.Context, src container.Source, hint string, dst io.Writer, opts TranscodeOptions) (*TranscodeResult, error) {
	demux, info, err := format.OpenDemuxer(src, hint, nil)
	if err != nil {
		return nil, err
	}
	return e.RemuxDemuxer(ctx, demux, info.Default(), dst, opts)
}

// RemuxDemuxer remuxes an already-opened demuxer to dst, the same packet copy as Remux without the source-open step.
func (e *Engine) RemuxDemuxer(ctx context.Context, demux container.Demuxer, track container.Track,
	dst io.Writer, opts TranscodeOptions) (*TranscodeResult, error) {
	plan, err := e.PlanRemux(track, opts)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: a %s source cannot be remuxed to %s with these options; transcode it",
				track.Codec, opts.Format))
	}
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	mux, err := row.mux(plan.Track, opts, nil, dst)
	if err != nil {
		return nil, err
	}
	if err := checkSeekable(mux, dst, opts.Format); err != nil {
		return nil, err
	}
	if err := mux.Begin([]container.Track{plan.Track}); err != nil {
		return nil, err
	}
	e.log.Debug("remux started", "codec", track.Codec,
		"out", opts.Format, "outContainer", plan.Container, "samples", plan.Track.Samples)

	var done int64
	decoded, err := copyPackets(ctx, demux, track.ID, func(pkt container.Packet) error {
		if err := mux.WritePacket(container.Packet{Track: 0, Packet: pkt.Packet}); err != nil {
			return err
		}
		if opts.Progress != nil {
			done += pkt.Dur
			opts.Progress(done, plan.Track.Samples)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	trailer := remuxTrailer(plan.Track, decoded)
	if err := mux.End(trailer); err != nil {
		return nil, err
	}
	e.log.Debug("remux finished", "samples", trailer.Samples)
	return &TranscodeResult{Samples: trailer.Samples, Format: track.Fmt, Container: plan.Container}, nil
}

func copyPackets(ctx context.Context, demux container.Demuxer, track int, write func(container.Packet) error) (int64, error) {
	var pkt container.Packet
	var samples int64
	for {
		if err := ctx.Err(); err != nil {
			return 0, waxerr.Wrap(waxerr.CodeCanceled, "remux canceled", err)
		}
		err := demux.ReadPacket(&pkt)
		if err == io.EOF {
			return samples, nil
		}
		if err != nil {
			return 0, err
		}
		if pkt.Track != track {
			continue
		}
		samples += pkt.Dur
		if err := write(pkt); err != nil {
			return 0, err
		}
	}
}
