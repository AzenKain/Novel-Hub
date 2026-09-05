package waxflow

import (
	"log/slog"

	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/dsp/dither"
	"novelhub/pkg/waxflow/dsp/gain"
	"novelhub/pkg/waxflow/dsp/resample"
)

// Option configures an Engine.
type Option func(*Engine)

// WithLogger sets the Engine's logger.
func WithLogger(l *slog.Logger) Option {
	return func(e *Engine) {
		if l != nil {
			e.log = l
		}
	}
}

// IndexCache persists demuxer-built source indexes across sessions (the cacheDir/idx sidecar): MP3 frame tables today, seek tables for later formats.
type IndexCache interface {
	Load(src container.Source) []byte
	Save(src container.Source, blob []byte)
	Drop(src container.Source)
}

// WithIndexCache wires an index sidecar cache into the Engine.
func WithIndexCache(c IndexCache) Option {
	return func(e *Engine) {
		e.idx = c
	}
}

// TranscodeOptions selects the Transcode output, with the DSP chain (resample, mix, gain, dither) between decode and encode.
type TranscodeOptions struct {
	Format          string
	Container       string
	Rate            int
	Channels        int
	BitDepth        int
	GainDB          float64
	Dynamics        gain.Preset
	FromSample      int64
	FLACLevel       int
	MP3Bitrate      int
	MP3VBR          bool
	OpusBitrate     int
	AACBitrate      int
	OpusComplexity  int
	OpusVBR         bool
	OpusSignal      string
	VorbisQuality   float64
	VorbisBitrate   int
	Shaping         dither.Shaping
	ResampleProfile resample.Profile
	Tags            []container.Tag
	Chapters        []container.Chapter
	Art             *container.Picture
	Progress        func(done, total int64)
}

const (
	FLACLevelDefault = 0
	FLACLevelFastest = -1
)

const (
	OpusComplexityDefault = 0
	OpusComplexityLowest  = -1
)

// ProbeOptions configures Engine.Probe.
type ProbeOptions struct {
	Strict bool
}
