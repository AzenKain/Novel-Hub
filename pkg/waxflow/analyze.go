package waxflow

import (
	"context"
	"fmt"
	"io"
	"time"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/dsp"
	"novelhub/pkg/waxflow/dsp/loudness"
	"novelhub/pkg/waxflow/dsp/mix"
	"novelhub/pkg/waxflow/dsp/silence"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

const (
	DefaultSilenceThresholdDB = -50.0
	DefaultSilenceMinDuration = 500 * time.Millisecond
)

// AnalyzeOptions configures Engine.Analyze.
type AnalyzeOptions struct {
	Channels int
	Progress func(done, total int64)
	Silence  *SilenceOptions
	Tap      func(chans [][]float32) error
}

// SilenceOptions configures the silence map.
type SilenceOptions struct {
	ThresholdDB float64
	MinDuration time.Duration
}

func (o SilenceOptions) resolve() (thresholdDB float64, minDur time.Duration) {
	thresholdDB, minDur = o.ThresholdDB, o.MinDuration
	if thresholdDB == 0 {
		thresholdDB = DefaultSilenceThresholdDB
	}
	if minDur == 0 {
		minDur = DefaultSilenceMinDuration
	}
	return thresholdDB, minDur
}

// SilenceSpan is one silent span of the analyzed source, in frames on its own timeline (ADR-0006).
type SilenceSpan struct {
	From int64
	To   int64
}

// SilenceResult is the silence map: the spans plus the parameters they were found with, so a caller that stores the map can tell what it means.
type SilenceResult struct {
	Version        string
	ThresholdDB    float64
	MinDuration    time.Duration
	Spans          []SilenceSpan
	Dropped        int
	DroppedSamples int64
	TotalSamples   int64
}

// AnalyzeResult is a full-stream loudness measurement of the decoded audio per ITU-R BS.1770-4 and EBU R128.
type AnalyzeResult struct {
	Format         audio.Format
	Samples        int64
	IntegratedLUFS float64
	LoudnessRange  float64
	TruePeakDB     float64
	SamplePeakDB   float64
	Silence        *SilenceResult
}

// Analyze decodes src end to end and measures its loudness: integrated LUFS, loudness range, true peak, and sample peak.
func (e *Engine) Analyze(ctx context.Context, src container.Source, hint string, opts AnalyzeOptions) (*AnalyzeResult, error) {
	med, err := e.OpenStream(src, hint)
	if err != nil {
		return nil, err
	}
	defer med.Close()
	return e.AnalyzeMedia(ctx, med, opts)
}

// AnalyzeMedia analyzes an already-opened Media, the same measurement as Analyze without the source-open step.
func (e *Engine) AnalyzeMedia(ctx context.Context, med format.Media, opts AnalyzeOptions) (*AnalyzeResult, error) {
	if opts.Channels < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("analyze: negative channel count %d", opts.Channels))
	}
	track := med.Info().Default()
	chain, err := dsp.NewChain(dsp.NewSource(med, track.Fmt), dsp.ChainSpec{Float: true})
	if err != nil {
		return nil, err
	}
	defer chain.Release()

	f := chain.Format()

	meterFmt := f
	var matrix *mix.Matrix
	var scratch *audio.Buffer
	var dstV [][]float32
	if opts.Channels != 0 && opts.Channels != f.Channels {
		srcLayout := f.Layout
		if srcLayout == 0 {
			srcLayout = audio.DefaultLayout(f.Channels)
		}
		dstLayout := audio.DefaultLayout(opts.Channels)
		if srcLayout == 0 || dstLayout == 0 {
			return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
				fmt.Sprintf("analyze: no layout convention for %d -> %d channels", f.Channels, opts.Channels))
		}
		matrix, err = mix.For(srcLayout, dstLayout)
		if err != nil {
			return nil, err
		}
		meterFmt.Channels = opts.Channels
		meterFmt.Layout = dstLayout
		scratch = audio.Get(meterFmt, audio.StandardChunk)
		defer audio.Put(scratch)
		dstV = make([][]float32, opts.Channels)
	}

	meter, err := loudness.NewMeter(meterFmt.Rate, meterFmt.Channels, meterFmt.Layout)
	if err != nil {
		return nil, err
	}
	var det *silence.Detector
	var silThreshold float64
	var silMinDur time.Duration
	if opts.Silence != nil {
		silThreshold, silMinDur = opts.Silence.resolve()
		if det, err = silence.New(f.Rate, f.Channels, silThreshold, silMinDur); err != nil {
			return nil, err
		}
	}
	buf := audio.Get(f, audio.StandardChunk)
	defer audio.Put(buf)
	chans := make([][]float32, f.Channels)
	var done int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, waxerr.Wrap(waxerr.CodeCanceled, "analyze canceled", err)
		}
		err := chain.ReadChunk(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for c := range chans {
			chans[c] = buf.ChanF(c)
		}
		if matrix != nil {
			scratch.N = buf.N
			for c := range dstV {
				dstV[c] = scratch.ChanF(c)
			}
			matrix.Apply(dstV, chans, buf.N)
			if err := meter.Process(dstV); err != nil {
				return nil, err
			}
		} else if err := meter.Process(chans); err != nil {
			return nil, err
		}
		if det != nil {
			if err := det.Process(chans); err != nil {
				return nil, err
			}
		}
		if opts.Tap != nil {
			if err := opts.Tap(chans); err != nil {
				return nil, err
			}
		}
		done += int64(buf.N)
		if opts.Progress != nil {
			opts.Progress(done, track.Samples)
		}
	}
	meter.Flush()
	res := &AnalyzeResult{
		Format:         meterFmt,
		Samples:        done,
		IntegratedLUFS: meter.Integrated(),
		LoudnessRange:  meter.Range(),
		TruePeakDB:     meter.TruePeak(),
		SamplePeakDB:   meter.SamplePeak(),
	}
	if det != nil {
		det.Flush()
		spans := make([]SilenceSpan, len(det.Spans()))
		for i, s := range det.Spans() {
			spans[i] = SilenceSpan{From: s.From, To: s.To}
		}
		res.Silence = &SilenceResult{
			Version:        silence.Version,
			ThresholdDB:    silThreshold,
			MinDuration:    silMinDur,
			Spans:          spans,
			Dropped:        det.Dropped(),
			DroppedSamples: det.DroppedSamples(),
			TotalSamples:   det.TotalSamples(),
		}
	}
	return res, nil
}
