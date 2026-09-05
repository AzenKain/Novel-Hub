package waxflow

import (
	"context"
	"fmt"
	"io"

	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/codec/opus"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

// CutVersion identifies the cut rung's own sample-affecting logic for the ADR-0004 cache key.
const CutVersion = "cut-1"

// Span is a kept sample range [From, To) of a source's own track timeline.
type Span struct{ From, To int64 }

// CutPlan describes what a cut would produce, computed from the source track's headers alone.
type CutPlan struct {
	RemuxPlan
	Landed []Span
}

type cutCodec struct {
	preroll int64
	reprime func(cfg []byte, delay int64) ([]byte, error)
}

var cutCodecs = map[codec.ID]cutCodec{
	codec.Opus: {
		preroll: opus.SeekPreroll,
		reprime: func(cfg []byte, delay int64) ([]byte, error) {
			return opus.SetPreSkip(cfg, int(delay))
		},
	},
	codec.AACLC: {preroll: aac.EncoderDelay},
}

// Cuttable reports whether track's codec is one whose packets survive being moved within a stream, which is the premise the cut rung rests on.
func Cuttable(track container.Track) bool {
	_, ok := cutCodecs[track.Codec]
	return ok
}

// CutFormats lists the output formats the cut rung serves without re-encoding, in table order.
func CutFormats() []string {
	var names []string
	for _, o := range outputs {
		if _, ok := cutCodecs[o.codecID]; ok && o.live && o.hls != nil {
			names = append(names, o.name)
		}
	}
	return names
}

type cutWindow struct{ from, to int64 }

type cutResult struct {
	track   container.Track
	landed  []Span
	windows []cutWindow
}

const maxCutSample = 1 << 61

func snapGridDown(x, g int64) int64 {
	if x <= 0 {
		return 0
	}
	return x / g * g
}

func snapGridUp(x, g int64) int64 {
	if x <= 0 {
		return 0
	}
	return (x + g - 1) / g * g
}

// CutTrack synthesizes the track a cut of track to spans would produce: its trims, its length, its rewritten codec config, and where the spans landed.
func CutTrack(track container.Track, spans []Span, grid int) (container.Track, []Span, error) {
	res, err := computeCut(track, spans, grid)
	if err != nil {
		return container.Track{}, nil, err
	}
	return res.track, res.landed, nil
}

func validateCutSpans(track container.Track, spans []Span) error {
	if len(spans) == 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, "waxflow: a cut needs at least one span")
	}
	for i, s := range spans {
		last := i == len(spans)-1
		switch {
		case s.From < 0:
			return waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: negative span start %d", s.From))
		case s.To < ToEnd:
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span end %d: want a sample offset or %d for the end of the source", s.To, ToEnd))
		case s.To == ToEnd && !last:
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span %d runs to the end of the source but %d more follow it", i, len(spans)-1-i))
		case s.To >= 0 && s.To < s.From:
			return waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: span [%d, %d) ends before it starts", s.From, s.To))
		case s.From >= maxCutSample || s.To >= maxCutSample:
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span [%d, %d) overflows the sample timeline", s.From, s.To))
		case s.To == s.From:
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span [%d, %d) keeps no samples; a cut cannot express an empty span", s.From, s.To))
		case s.To == ToEnd && track.Samples >= 0 && s.From == track.Samples:
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span [%d, ToEnd) starts at the end of the source's %d samples and so keeps none; a cut cannot express an empty span",
				s.From, track.Samples))
		}
		if track.Samples >= 0 {
			if s.From > track.Samples {
				return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
					"waxflow: span starts at sample %d, past the source's %d samples", s.From, track.Samples))
			}
			if s.To > track.Samples {
				return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
					"waxflow: span ends at sample %d, past the source's %d samples", s.To, track.Samples))
			}
		}
		if i > 0 && s.From < spans[i-1].To {
			return waxerr.New(waxerr.CodeInvalidRequest, fmt.Sprintf(
				"waxflow: span [%d, %d) starts before span %d ended at %d; spans must be in order and disjoint",
				s.From, s.To, i-1, spans[i-1].To))
		}
	}
	return nil
}

func computeCut(track container.Track, spans []Span, grid int) (*cutResult, error) {
	if err := validateCutSpans(track, spans); err != nil {
		return nil, err
	}
	cc, ok := cutCodecs[track.Codec]
	if !ok {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
			"waxflow: %s packets do not survive being moved within the stream, so this source cannot be cut without re-encoding",
			track.Codec))
	}
	if grid <= 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			"waxflow: this source's packet durations vary, so there is no grid to cut on")
	}
	if track.Delay < 0 || track.Delay >= maxCutSample || int64(grid) >= maxCutSample {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
			"waxflow: this source declares a %d-sample delay on a %d-sample grid, which is outside the timeline this rung can compute in",
			track.Delay, grid))
	}
	g := int64(grid)
	n := len(spans)

	decodedEnd := int64(-1)
	if track.Samples >= 0 {
		decodedEnd = track.Delay + track.Samples + track.Padding
	}
	lastToEnd := spans[n-1].To == ToEnd

	df := make([]int64, n)
	dt := make([]int64, n)
	sd := make([]int64, n)
	su := make([]int64, n)
	for i, s := range spans {
		df[i] = s.From + track.Delay
		sd[i] = snapGridDown(df[i]-cc.preroll, g)
		switch {
		case s.To != ToEnd:
			dt[i] = s.To + track.Delay
			su[i] = snapGridUp(dt[i], g)
		case decodedEnd >= 0:
			dt[i] = track.Delay + track.Samples
			su[i] = decodedEnd
		default:
			dt[i], su[i] = -1, -1
		}
	}

	for i := 0; i < n-1; i++ {
		if su[i] > sd[i+1] {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
				"waxflow: the gap between spans %d and %d is smaller than the %d-sample packet grid can express, so this cut cannot be made without re-encoding",
				i, i+1, grid))
		}
	}
	tailClamped := false
	if !lastToEnd && decodedEnd >= 0 && su[n-1] > decodedEnd {
		su[n-1] = decodedEnd
		tailClamped = true
	}

	out := track
	out.Delay = df[0] - sd[0]

	if tailClamped && out.Delay == 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
			"waxflow: span [%d, %d) ends inside the source's final packet, whose length the header does not state, and this cut has no delay for the trailer to resolve it against; re-encode it",
			spans[n-1].From, spans[n-1].To))
	}

	switch {
	case lastToEnd && track.Samples < 0:
		out.Padding = track.Padding
		out.Samples = -1
	default:
		out.Padding = su[n-1] - dt[n-1]
		var delivered int64
		for i := range spans {
			delivered += su[i] - sd[i]
		}
		out.Samples = delivered - out.Delay - out.Padding
	}
	if !lastToEnd {
		out.SamplesExact = true
	}
	if cc.reprime != nil {
		cfg, err := cc.reprime(track.CodecConfig, out.Delay)
		if err != nil {
			return nil, err
		}
		out.CodecConfig = cfg
	}

	landed := make([]Span, n)
	windows := make([]cutWindow, n)
	for i := range spans {
		landed[i] = Span{From: sd[i] - track.Delay, To: su[i] - track.Delay}
		windows[i] = cutWindow{from: sd[i], to: su[i]}
	}
	landed[0].From = spans[0].From
	switch {
	case !lastToEnd:
		landed[n-1].To = spans[n-1].To
	case track.Samples >= 0:
		landed[n-1].To = track.Samples
	default:
		landed[n-1].To = ToEnd
	}
	if lastToEnd || tailClamped {
		windows[n-1].to = -1
	}
	return &cutResult{track: out, landed: landed, windows: windows}, nil
}

// Cut returns a view of demux holding only track's packets that fall in spans, retimed to be contiguous: the packet-domain sibling of Slice, and the input side of a cut.
func Cut(demux container.Demuxer, track container.Track, spans []Span, grid int) (container.Demuxer, error) {
	res, err := computeCut(track, spans, grid)
	if err != nil {
		return nil, err
	}
	return &cutDemuxer{Demuxer: demux, cut: res.track, track: track.ID, windows: res.windows}, nil
}

type cutDemuxer struct {
	container.Demuxer
	cut     container.Track
	track   int
	windows []cutWindow
	cur     int
	pos     int64
	out     int64
}

// Tracks reports the cut's own track, which is the one the packets coming out of ReadPacket belong to.
func (c *cutDemuxer) Tracks() []container.Track { return []container.Track{c.cut} }

func (c *cutDemuxer) ReadPacket(pkt *container.Packet) error {
	for {
		if c.cur == len(c.windows) {
			return io.EOF
		}
		if err := c.Demuxer.ReadPacket(pkt); err != nil {
			return err
		}
		if pkt.Track != c.track {
			continue
		}
		start, end := c.pos, c.pos+pkt.Dur
		c.pos = end
		for c.cur < len(c.windows) && c.windows[c.cur].to >= 0 && start >= c.windows[c.cur].to {
			c.cur++
		}
		if c.cur == len(c.windows) {
			continue
		}
		w := c.windows[c.cur]
		if end <= w.from {
			continue
		}
		if start < w.from {
			return cutStraddle(start, end, w.from)
		}
		if w.to >= 0 && end > w.to {
			return cutStraddle(start, end, w.to)
		}
		pkt.PTS = c.out
		c.out += pkt.Dur
		return nil
	}
}

func cutStraddle(start, end, at int64) error {
	return waxerr.New(waxerr.CodeUnsupportedFormat, fmt.Sprintf(
		"waxflow: source packet [%d, %d) straddles the cut boundary at %d; this stream cannot be cut without re-encoding",
		start, end, at))
}

// PlanCut reports whether opts can be served by cutting track's existing packets to spans and rewriting the container around them, and how.
func (e *Engine) PlanCut(track container.Track, opts TranscodeOptions, spans []Span, grid int) (*CutPlan, error) {
	cut, landed, err := CutTrack(track, spans, grid)
	if err != nil {
		if waxerr.CodeOf(err) == waxerr.CodeUnsupportedFormat {
			e.log.Debug("cut declined", "codec", track.Codec, "grid", grid, "reason", err)
			return nil, nil
		}
		return nil, err
	}
	rp, err := e.PlanRemux(cut, opts)
	if err != nil || rp == nil {
		return nil, err
	}
	if !cutTrimsExpressible(rp.Container, cut.Delay, cut.Padding) {
		e.log.Debug("cut declined", "reason", "the destination cannot signal the cut's trims",
			"outContainer", rp.Container, "delay", cut.Delay, "padding", cut.Padding)
		return nil, nil
	}
	rp.Versions = []string{RemuxVersion, CutVersion}
	return &CutPlan{RemuxPlan: *rp, Landed: landed}, nil
}

func cutTrimsExpressible(containerName string, delay, padding int64) bool {
	if delay == 0 && padding == 0 {
		return true
	}
	switch containerName {
	case "opus", "mka", "webm":
		return true
	case ContainerProgressive:
		return true
	case "aac":
		return delay > 0
	}
	return false
}

// CutStream cuts src's existing packets to spans and rewrites the container around them, opening the source itself: the run half of the cut rung, and the packet-move sibling of Remux.
func (e *Engine) CutStream(ctx context.Context, src container.Source, hint string, dst io.Writer,
	opts TranscodeOptions, spans []Span, grid int, samples int64) (*TranscodeResult, error) {
	demux, info, err := format.OpenDemuxer(src, hint, nil)
	if err != nil {
		return nil, err
	}
	track := info.Default()
	if samples >= 0 {
		track.Samples, track.SamplesExact = samples, true
	}
	cut, _, err := CutTrack(track, spans, grid)
	if err != nil {
		return nil, err
	}
	cutDemux, err := Cut(demux, track, spans, grid)
	if err != nil {
		return nil, err
	}
	return e.RemuxDemuxer(ctx, cutDemux, cut, dst, opts)
}
