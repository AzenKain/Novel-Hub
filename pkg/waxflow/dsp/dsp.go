// Package dsp assembles the transcode pipeline's PCM processing chain: format.Media -> [convert] -> [resample] -> [mix] -> [gain] -> [dynamics] -> [limiter] -> [dither] -> framer Nodes are inserted only when needed, in that fixed order.
package dsp

import (
	"fmt"
	"time"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/dsp/dither"
	"novelhub/pkg/waxflow/dsp/gain"
	"novelhub/pkg/waxflow/dsp/mix"
	"novelhub/pkg/waxflow/dsp/resample"
	"novelhub/pkg/waxflow/waxerr"
)

const (
	convertVersion = "convert-1"
	widenVersion   = "widen-1"
)

const maxGainDB = 120

// Stage is one pull-based pipeline node.
type Stage interface {
	Format() audio.Format
	ReadChunk(dst *audio.Buffer) error
}

// Reader is the upstream side of a chain.
type Reader interface {
	ReadChunk(dst *audio.Buffer) error
}

// Settler is an optional capability, asserted by pumpStage on its kernel.
type Settler interface {
	Horizon() time.Duration
}

type releaser interface{ release() }

// NewSource adapts a Reader (typically format.Media) into the chain's head Stage.
func NewSource(r Reader, f audio.Format) Stage {
	return &sourceStage{r: r, fmt: f}
}

type sourceStage struct {
	r   Reader
	fmt audio.Format
}

func (s *sourceStage) Format() audio.Format { return s.fmt }

// ReadChunk pulls from the reader and holds it to the one part of the Stage contract the chain cannot survive being wrong about: io.EOF is the only empty answer.
func (s *sourceStage) ReadChunk(dst *audio.Buffer) error {
	err := s.r.ReadChunk(dst)
	if err == nil && dst.N == 0 {
		return waxerr.New(waxerr.CodeInternal,
			"dsp: the chain's source returned no frames and no error; io.EOF is the only empty answer")
	}
	return err
}

// ChainSpec declares the output a chain must produce.
type ChainSpec struct {
	Rate       int
	Channels   int
	BitDepth   int
	Float      bool
	GainDB     float64
	Dynamics   gain.Preset
	Shaping    dither.Shaping
	DitherSeed uint64
	Profile    resample.Profile
	FrameSize  int
}

// Chain is an assembled processing pipeline.
type Chain struct {
	out      Stage
	stages   []Stage
	versions []string
	l, m     int
	horizon  time.Duration
}

// NewChain builds the processing chain from src's format to the spec, inserting only the nodes the conversion needs.
func NewChain(src Stage, spec ChainSpec) (*Chain, error) {
	in := src.Format()
	if err := in.Valid(); err != nil {
		return nil, err
	}
	if spec.Rate < 0 || spec.Channels < 0 || spec.FrameSize < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "dsp: negative chain spec field")
	}
	if spec.BitDepth != 0 && (spec.BitDepth < 2 || spec.BitDepth > 32) {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dsp: output depth %d outside 2..32", spec.BitDepth))
	}
	if spec.Float && spec.BitDepth != 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			"dsp: Float and BitDepth are mutually exclusive output domains")
	}
	if !(spec.GainDB >= -maxGainDB && spec.GainDB <= maxGainDB) {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dsp: gain %g dB outside +-%d dB", spec.GainDB, maxGainDB))
	}

	rate := in.Rate
	if spec.Rate != 0 {
		rate = spec.Rate
	}
	channels := in.Channels
	if spec.Channels != 0 {
		channels = spec.Channels
	}
	needRate := rate != in.Rate
	needMix := channels != in.Channels
	needGain := spec.GainDB != 0
	needDyn := spec.Dynamics != gain.PresetOff

	outInt := in.Type == audio.Int
	outDepth := in.BitDepth
	if spec.BitDepth != 0 {
		outInt = true
		outDepth = spec.BitDepth
	} else if spec.Float {
		outInt = false
	}
	narrowing := in.Type == audio.Int && outInt && outDepth < in.BitDepth
	forceFloat := spec.Float && in.Type == audio.Int
	floatWork := needRate || needMix || needGain || needDyn || narrowing || forceFloat

	c := &Chain{out: src, l: 1, m: 1}
	cur := in

	if floatWork && in.Type == audio.Int {
		cur = withType(cur, audio.Float, 32)
		c.push(&convertStage{up: c.out, fmt: cur, scale: float32(1 / scaleFor(in.BitDepth))}, convertVersion)
	}

	if needRate {
		r, err := resample.New(cur.Rate, rate, cur.Channels, profileOr(spec.Profile))
		if err != nil {
			return nil, err
		}
		c.l, c.m = r.Ratio()
		cur = withRate(cur, rate)
		c.push(newPump(c.out, cur, resampleOps{r}), profileOr(spec.Profile).Version())
	}

	var matrix *mix.Matrix
	if needMix {
		srcLayout := in.Layout
		if srcLayout == 0 {
			srcLayout = audio.DefaultLayout(in.Channels)
		}
		dstLayout := audio.DefaultLayout(channels)
		if srcLayout == 0 || dstLayout == 0 {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
				fmt.Sprintf("dsp: no layout convention for %d -> %d channels", in.Channels, channels))
		}
		m, err := mix.For(srcLayout, dstLayout)
		if err != nil {
			return nil, err
		}
		matrix = m
		cur = withLayout(cur, channels, dstLayout)
		c.push(&mixStage{up: c.out, fmt: cur, matrix: m}, mix.Version)
	}

	if needGain {
		c.push(&gainStage{up: c.out, fmt: cur, g: float32(gain.FromDB(spec.GainDB))}, gain.Version)
	}

	if needDyn {
		comp, err := gain.NewCompressor(cur.Rate, cur.Channels, spec.Dynamics)
		if err != nil {
			return nil, err
		}
		c.push(newPump(c.out, cur, compressorOps{comp}), gain.CompressorVersion)
	}

	if spec.GainDB > 0 || needDyn || (matrix != nil && matrix.MaxGain() > 1) {
		lim, err := gain.NewLimiter(cur.Rate, cur.Channels, gain.DefaultCeilingDB)
		if err != nil {
			return nil, err
		}
		c.push(newPump(c.out, cur, limiterOps{lim}), gain.LimiterVersion)
	}

	switch {
	case outInt && cur.Type == audio.Float:
		shaping := spec.Shaping
		if shaping == dither.Shaped && !dither.SupportsShaping(rate) {
			shaping = dither.TPDF
		}
		seed := spec.DitherSeed
		if seed == 0 {
			seed = dither.DefaultSeed
		}
		q, err := dither.NewQuantizer(outDepth, cur.Channels, shaping, seed)
		if err != nil {
			return nil, err
		}
		cur = withType(cur, audio.Int, outDepth)
		c.push(&quantizeStage{up: c.out, fmt: cur, q: q}, dither.Version)
	case outInt && outDepth > cur.BitDepth:
		shift := outDepth - cur.BitDepth
		cur = withType(cur, audio.Int, outDepth)
		c.push(&widenStage{up: c.out, fmt: cur, shift: uint(shift)}, widenVersion)
	}

	if spec.FrameSize > 0 {
		c.push(&framerStage{up: c.out, fmt: cur, size: spec.FrameSize}, "")
	}
	return c, nil
}

func (c *Chain) push(s Stage, version string) {
	c.out = s
	c.stages = append(c.stages, s)
	if version != "" {
		c.versions = append(c.versions, version)
	}
	if p, ok := s.(*pumpStage); ok {
		c.horizon += p.horizon()
	}
}

// Horizon reports how much pre-roll a mid-stream start must feed this chain before its first kept sample for the output to be bit-identical to a continuous run's: 0 when no node decays, which is every chain with no gain and no dynamics.
func (c *Chain) Horizon() time.Duration { return c.horizon }

// Format returns the chain's output format.
func (c *Chain) Format() audio.Format { return c.out.Format() }

// ReadChunk pulls the next processed chunk (Stage contract).
func (c *Chain) ReadChunk(dst *audio.Buffer) error {
	if dst.Fmt != c.out.Format() {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("dsp: chunk buffer is %v, chain output is %v", dst.Fmt, c.out.Format()))
	}
	if dst.Cap() == 0 {
		return waxerr.New(waxerr.CodeInvalidRequest, "dsp: zero-capacity chunk buffer")
	}
	return c.out.ReadChunk(dst)
}

// Ratio returns the chain's output/input rate ratio in lowest terms, 1/1 when the rate is unchanged.
func (c *Chain) Ratio() (l, m int) { return c.l, c.m }

// OutputSamples maps a source-stream length onto the chain's output length: rate conversion rescales (ceil, matching the resampler's drain guarantee), everything else is one to one.
func (c *Chain) OutputSamples(in int64) int64 {
	if in < 0 {
		return -1
	}
	return (in*int64(c.l) + int64(c.m) - 1) / int64(c.m)
}

// Versions lists the algorithm revisions of every sample-affecting node in chain order, for the cache key (ADR-0004).
func (c *Chain) Versions() []string { return c.versions }

// Release returns all stage scratch buffers to the pool.
func (c *Chain) Release() {
	for _, s := range c.stages {
		if r, ok := s.(releaser); ok {
			r.release()
		}
	}
}

func profileOr(p resample.Profile) resample.Profile {
	if p == "" {
		return resample.HQ
	}
	return p
}

func scaleFor(bits int) float64 {
	return float64(int64(1) << (bits - 1))
}

func withType(f audio.Format, t audio.SampleType, depth int) audio.Format {
	f.Type = t
	f.BitDepth = depth
	return f
}

func withRate(f audio.Format, rate int) audio.Format {
	f.Rate = rate
	return f
}

func withLayout(f audio.Format, channels int, layout audio.ChannelMask) audio.Format {
	f.Channels = channels
	f.Layout = layout
	return f
}
