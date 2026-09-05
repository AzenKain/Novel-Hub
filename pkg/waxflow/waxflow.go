package waxflow

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/codec/alac"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/codec/mp3"
	"novelhub/pkg/waxflow/codec/opus"
	"novelhub/pkg/waxflow/codec/pcm"
	"novelhub/pkg/waxflow/codec/vorbis"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/container/adts"
	"novelhub/pkg/waxflow/container/aiff"
	"novelhub/pkg/waxflow/container/flacn"
	"novelhub/pkg/waxflow/container/mka"
	"novelhub/pkg/waxflow/container/mp4"
	"novelhub/pkg/waxflow/container/mpa"
	"novelhub/pkg/waxflow/container/ogg"
	"novelhub/pkg/waxflow/container/riff"
	"novelhub/pkg/waxflow/dsp"
	"novelhub/pkg/waxflow/dsp/dither"
	"novelhub/pkg/waxflow/dsp/gain"
	"novelhub/pkg/waxflow/dsp/resample"
	"novelhub/pkg/waxflow/format"
	"novelhub/pkg/waxflow/waxerr"
)

// Engine is the library-first entry point to the transcoding pipeline.
type Engine struct {
	log *slog.Logger
	idx IndexCache

	mu    sync.RWMutex
	plans map[planKey]*planCore
}

// New returns an Engine.
func New(opts ...Option) *Engine {
	e := &Engine{log: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Probe identifies src and returns its parsed headers.
func (e *Engine) Probe(src container.Source, hint string, opts *ProbeOptions) (*format.Info, error) {
	return format.Probe(src, hint, &format.Options{Strict: opts != nil && opts.Strict})
}

// OpenStream opens src for decoded, sample-exact PCM access.
func (e *Engine) OpenStream(src container.Source, hint string) (format.Media, error) {
	med, err := format.Open(src, hint, nil)
	if err != nil || e.idx == nil {
		return med, err
	}
	ix, ok := med.(container.Indexer)
	if !ok {
		return med, nil
	}
	if blob := e.idx.Load(src); blob != nil {
		if !ix.RestoreIndex(blob) {
			e.idx.Drop(src)
			e.log.Debug("index sidecar rejected and dropped", "hint", hint)
		}
	}
	return &indexSavingMedia{Media: med, ix: ix, cache: e.idx, src: src}, nil
}

type indexSavingMedia struct {
	format.Media
	ix    container.Indexer
	cache IndexCache
	src   container.Source
}

func (m *indexSavingMedia) Close() error {
	if m.ix != nil {
		if blob := m.ix.IndexSnapshot(); blob != nil {
			m.cache.Save(m.src, blob)
		}
		m.ix = nil
	}
	return m.Media.Close()
}

func (m *indexSavingMedia) IndexSnapshot() []byte {
	if m.ix == nil {
		return nil
	}
	return m.ix.IndexSnapshot()
}

func (m *indexSavingMedia) RestoreIndex(blob []byte) bool {
	return m.ix != nil && m.ix.RestoreIndex(blob)
}

// TranscodeResult reports what Transcode produced.
type TranscodeResult struct {
	Samples   int64
	Format    audio.Format
	Container string
}

// Transcode decodes src and writes it to dst in the requested output format: decode -> DSP -> encode -> mux, checking ctx between chunks.
func (e *Engine) Transcode(ctx context.Context, src container.Source, hint string, dst io.Writer, opts TranscodeOptions) (*TranscodeResult, error) {
	med, err := e.OpenStream(src, hint)
	if err != nil {
		return nil, err
	}
	defer med.Close()
	return e.TranscodeMedia(ctx, med, dst, opts)
}

// TranscodeMedia transcodes an already-opened Media to dst, the same decode -> DSP -> encode -> mux pipeline as Transcode without the source-open step.
func (e *Engine) TranscodeMedia(ctx context.Context, med format.Media, dst io.Writer, opts TranscodeOptions) (*TranscodeResult, error) {
	srcTrack := med.Info().Default()
	srcSamples := srcTrack.Samples
	if opts.FromSample < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: negative FromSample")
	}
	if opts.FromSample > 0 {
		landed, err := med.SeekSample(opts.FromSample)
		if err != nil {
			return nil, err
		}
		if srcSamples >= 0 {
			srcSamples = max(0, srcSamples-landed)
		}
	}
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	containerName, _, err := resolveContainer(row, opts.Container)
	if err != nil {
		return nil, err
	}
	spec := specFor(opts)
	if row.adjust != nil {
		row.adjust(&spec, srcTrack.Fmt, opts)
	}
	chain, err := dsp.NewChain(dsp.NewSource(med, srcTrack.Fmt), spec)
	if err != nil {
		return nil, err
	}
	defer chain.Release()

	f := chain.Format()
	e.logImplicitDownmix(opts, srcTrack.Fmt, f)
	enc, err := row.encode(f, opts)
	if err != nil {
		return nil, err
	}

	track := container.Track{
		Codec:       row.codecID,
		CodecConfig: enc.CodecConfig(),
		Fmt:         f,
		Samples:     chain.OutputSamples(srcSamples),
		Default:     true,
	}
	if d, ok := enc.(interface{ Delay() int }); ok {
		track.Delay = int64(d.Delay())
	}
	mux, err := row.mux(track, opts, enc, dst)
	if err != nil {
		return nil, err
	}
	if err := checkSeekable(mux, dst, opts.Format); err != nil {
		return nil, err
	}
	if err := mux.Begin([]container.Track{track}); err != nil {
		return nil, err
	}

	e.log.Debug("transcode started",
		"container", med.Info().Container, "source", srcTrack.Fmt.String(),
		"format", f.String(), "samples", track.Samples, "out", opts.Format,
		"dsp", strings.Join(chain.Versions(), ","))

	emit := func(p codec.Packet) error {
		return mux.WritePacket(container.Packet{Track: 0, Packet: p})
	}
	buf := audio.Get(f, max(audio.StandardChunk, spec.FrameSize))
	defer audio.Put(buf)
	var done int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, waxerr.Wrap(waxerr.CodeCanceled, "transcode canceled", err)
		}
		err := chain.ReadChunk(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := enc.Encode(buf, emit); err != nil {
			return nil, err
		}
		if opts.Progress != nil {
			done += int64(buf.N)
			opts.Progress(done, track.Samples)
		}
	}
	trailer, err := enc.Finish(emit)
	if err != nil {
		return nil, err
	}
	if err := mux.End(trailer); err != nil {
		return nil, err
	}
	e.log.Debug("transcode finished", "samples", trailer.Samples)
	return &TranscodeResult{Samples: trailer.Samples, Format: f, Container: containerName}, nil
}

func resolveContainer(row *output, name string) (containerName, mediaType string, err error) {
	if name == "" {
		return row.name, row.mediaType, nil
	}
	if row.container == nil {
		return "", "", waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: format %s has no alternate container (container=%s)", row.name, name))
	}
	mt, err := row.container(name)
	if err != nil {
		return "", "", err
	}
	return name, mt, nil
}

func checkSeekable(mux container.Muxer, dst io.Writer, format string) error {
	if !mux.NeedsSeek() {
		return nil
	}
	if _, ok := dst.(io.WriteSeeker); !ok {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: %s output requires a seekable destination", format))
	}
	return nil
}

func specFor(opts TranscodeOptions) dsp.ChainSpec {
	return dsp.ChainSpec{
		Rate:     opts.Rate,
		Channels: opts.Channels,
		BitDepth: opts.BitDepth,
		GainDB:   opts.GainDB,
		Dynamics: opts.Dynamics,
		Shaping:  opts.Shaping,
		Profile:  opts.ResampleProfile,
	}
}

// TranscodePlan describes what a transcode would produce, computed from headers alone: no decoding, no output.
type TranscodePlan struct {
	Format         audio.Format
	Container      string
	MediaType      string
	Live           bool
	Versions       []string
	Samples        int64
	BytesPerFrame  int
	FrameSize      int
	BitRate        int
	EstimatedBytes int64
}

func decodeVersion(id codec.ID) string {
	switch id {
	case codec.PCM:
		return pcm.Version
	case codec.FLAC:
		return flac.Version
	case codec.ALAC:
		return alac.Version
	case codec.MP3:
		return mp3.Version
	case codec.AACLC:
		return aac.Version
	case codec.Opus:
		return opus.Version
	case codec.Vorbis:
		return vorbis.Version
	default:
		return "dec:" + string(id)
	}
}

type eofReader struct{}

func (eofReader) ReadChunk(*audio.Buffer) error { return io.EOF }

type planKey struct {
	fmt  audio.Format
	opts planOpts
}

type planOpts struct {
	Format          string
	Container       string
	Rate            int
	Channels        int
	BitDepth        int
	GainDB          float64
	Dynamics        gain.Preset
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
}

func planOptsOf(opts TranscodeOptions) planOpts {
	return planOpts{
		Format:          opts.Format,
		Container:       opts.Container,
		Rate:            opts.Rate,
		Channels:        opts.Channels,
		BitDepth:        opts.BitDepth,
		GainDB:          opts.GainDB,
		Dynamics:        opts.Dynamics,
		FLACLevel:       opts.FLACLevel,
		MP3Bitrate:      opts.MP3Bitrate,
		MP3VBR:          opts.MP3VBR,
		OpusBitrate:     opts.OpusBitrate,
		AACBitrate:      opts.AACBitrate,
		OpusComplexity:  opts.OpusComplexity,
		OpusVBR:         opts.OpusVBR,
		OpusSignal:      opts.OpusSignal,
		VorbisQuality:   opts.VorbisQuality,
		VorbisBitrate:   opts.VorbisBitrate,
		Shaping:         opts.Shaping,
		ResampleProfile: opts.ResampleProfile,
	}
}

type planCore struct {
	format        audio.Format
	container     string
	mediaType     string
	live          bool
	versions      []string
	l, m          int
	bytesPerFrame int
	bitRate       int
	headerBytes   int
	frameSize     int
}

const maxPlanCache = 1024

// PlanTranscode plans a transcode of the given source track without opening a pipeline.
func (e *Engine) PlanTranscode(track container.Track, opts TranscodeOptions) (*TranscodePlan, error) {
	if opts.FromSample < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: negative FromSample")
	}
	key := planKey{fmt: track.Fmt, opts: planOptsOf(opts)}

	e.mu.RLock()
	core, ok := e.plans[key]
	e.mu.RUnlock()
	if !ok {
		norm := opts
		norm.FromSample = 0
		norm.Tags, norm.Chapters, norm.Art, norm.Progress = nil, nil, nil, nil
		var err error
		if core, err = buildPlanCore(track.Fmt, norm); err != nil {
			return nil, err
		}
		e.mu.Lock()
		if len(e.plans) < maxPlanCache {
			if e.plans == nil {
				e.plans = make(map[planKey]*planCore)
			}
			e.plans[key] = core
		}
		e.mu.Unlock()
	}

	remaining := track.Samples
	if remaining >= 0 {
		remaining = max(0, remaining-opts.FromSample)
	}
	samples := remaining
	if samples >= 0 {
		samples = (samples*int64(core.l) + int64(core.m) - 1) / int64(core.m)
	}
	bitRate := core.bitRate
	if bitRate == 0 {
		bitRate = core.bytesPerFrame * core.format.Rate * 8
	}
	estimated := int64(-1)
	switch {
	case samples >= 0 && core.bytesPerFrame > 0:
		estimated = int64(core.headerBytes) + samples*int64(core.bytesPerFrame)
	case samples >= 0 && bitRate > 0 && core.format.Rate > 0:
		estimated = int64(core.headerBytes) + samples*int64(bitRate)/(int64(core.format.Rate)*8)
	}
	return &TranscodePlan{
		Format:         core.format,
		Container:      core.container,
		MediaType:      core.mediaType,
		Live:           core.live,
		Versions:       append([]string{decodeVersion(track.Codec)}, core.versions...),
		Samples:        samples,
		BytesPerFrame:  core.bytesPerFrame,
		FrameSize:      core.frameSize,
		BitRate:        bitRate,
		EstimatedBytes: estimated,
	}, nil
}

func buildPlanCore(in audio.Format, opts TranscodeOptions) (*planCore, error) {
	row, err := outputRow(opts.Format)
	if err != nil {
		return nil, err
	}
	spec := specFor(opts)
	if row.adjust != nil {
		row.adjust(&spec, in, opts)
	}
	chain, err := dsp.NewChain(dsp.NewSource(eofReader{}, in), spec)
	if err != nil {
		return nil, err
	}
	defer chain.Release()
	f := chain.Format()
	version, bytesPerFrame, bitRate, err := row.plan(f, opts)
	if err != nil {
		if opts.Channels == 0 && in.Channels > 2 && planAcceptsStereo(row, f, opts) {
			return nil, fmt.Errorf("%w; set the output channel count to 2", err)
		}
		return nil, err
	}
	l, m := chain.Ratio()
	containerName, mediaType, err := resolveContainer(row, opts.Container)
	if err != nil {
		return nil, err
	}
	return &planCore{
		format:        f,
		container:     containerName,
		mediaType:     mediaType,
		live:          containerLive(row.live, opts.Container),
		versions:      append(chain.Versions(), version),
		l:             l,
		m:             m,
		bytesPerFrame: bytesPerFrame,
		bitRate:       bitRate,
		headerBytes:   row.headerBytes,
		frameSize:     spec.FrameSize,
	}, nil
}

func planAcceptsStereo(row *output, f audio.Format, opts TranscodeOptions) bool {
	f.Channels = 2
	f.Layout = audio.DefaultLayout(2)
	_, _, _, err := row.plan(f, opts)
	return err == nil
}

type output struct {
	name        string
	exts        []string
	writeExt    string
	live        bool
	lossy       bool
	mediaType   string
	headerBytes int
	codecID     codec.ID
	adjust      func(spec *dsp.ChainSpec, src audio.Format, opts TranscodeOptions)
	plan        func(f audio.Format, opts TranscodeOptions) (version string, bytesPerFrame, bitRate int, err error)
	encode      func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error)
	mux         func(t container.Track, opts TranscodeOptions, enc codec.Encoder, dst io.Writer) (container.Muxer, error)
	container   func(name string) (mediaType string, err error)
	hls         *hlsOutput
}

type hlsOutput struct {
	// codecs is the RFC 6381 CODECS attribute value for master playlists.
	codecs string
	delay  int64
	encode func(f audio.Format, opts TranscodeOptions, startSample int64) (codec.Encoder, error)
}

var outputs = []output{
	{
		name:        "wav",
		exts:        []string{"wav", "wave", "rf64", "bw64"},
		live:        true,
		mediaType:   "audio/wav",
		headerBytes: 44,
		codecID:     codec.PCM,
		plan: func(f audio.Format, _ TranscodeOptions) (string, int, int, error) {
			cfg, err := riff.DefaultConfig(f)
			if err != nil {
				return "", 0, 0, err
			}
			return pcm.Version, cfg.BytesPerFrame(f.Channels), 0, nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			if isMatroska(opts.Container) {
				return pcm.NewEncoder(mkaPCMConfig(f), f)
			}
			cfg, err := riff.DefaultConfig(f)
			if err != nil {
				return nil, err
			}
			return pcm.NewEncoder(cfg, f)
		},
		mux: func(_ container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if isMatroska(opts.Container) {
				return mkaMuxer(dst, opts), nil
			}
			return riff.NewMuxer(dst, nil), nil
		},
		container: func(name string) (string, error) {
			if mt, ok := matroskaContainer(name, false); ok {
				return mt, nil
			}
			return "", waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: wav container %q: want mka", name))
		},
	},
	{
		name:        "opus",
		exts:        []string{"opus"},
		live:        true,
		lossy:       true,
		mediaType:   "audio/ogg",
		headerBytes: 512,
		codecID:     codec.Opus,
		adjust: func(spec *dsp.ChainSpec, src audio.Format, _ TranscodeOptions) {
			spec.Rate = opus.SampleRate
			spec.Float = true
			spec.BitDepth = 0
			spec.FrameSize = 960
			foldWideToStereo(spec, src)
		},
		plan: func(f audio.Format, opts TranscodeOptions) (string, int, int, error) {
			eopts, err := opusEncoderOptions(opts)
			if err != nil {
				return "", 0, 0, err
			}
			enc, err := opus.NewEncoder(f, eopts)
			if err != nil {
				return "", 0, 0, err
			}
			return opus.EncoderVersion, 0, enc.Bitrate(), nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			eopts, err := opusEncoderOptions(opts)
			if err != nil {
				return nil, err
			}
			return opus.NewEncoder(f, eopts)
		},
		mux: func(_ container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if isMatroska(opts.Container) {
				return mkaMuxer(dst, opts), nil
			}
			return ogg.NewMuxer(dst, &ogg.MuxerOptions{Tags: opts.Tags}), nil
		},
		container: func(name string) (string, error) {
			if mt, ok := matroskaContainer(name, true); ok {
				return mt, nil
			}
			return "", waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: opus container %q: want mka or webm", name))
		},
		hls: &hlsOutput{
			codecs: "Opus",
			delay:  opus.EncoderDelay,
			encode: func(f audio.Format, opts TranscodeOptions, _ int64) (codec.Encoder, error) {
				eopts, err := opusEncoderOptions(opts)
				if err != nil {
					return nil, err
				}
				return opus.NewEncoder(f, eopts)
			},
		},
	},
	{
		name:        "vorbis",
		exts:        []string{"ogg", "oga"},
		live:        true,
		lossy:       true,
		mediaType:   "audio/ogg",
		headerBytes: 4096,
		codecID:     codec.Vorbis,
		adjust: func(spec *dsp.ChainSpec, _ audio.Format, _ TranscodeOptions) {
			spec.Float = true
			spec.BitDepth = 0
		},
		plan: func(f audio.Format, opts TranscodeOptions) (string, int, int, error) {
			eopts, err := vorbisEncoderOptions(opts)
			if err != nil {
				return "", 0, 0, err
			}
			enc, err := vorbis.NewEncoder(f, eopts)
			if err != nil {
				return "", 0, 0, err
			}
			return vorbis.EncoderVersion, 0, enc.Bitrate(), nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			eopts, err := vorbisEncoderOptions(opts)
			if err != nil {
				return nil, err
			}
			return vorbis.NewEncoder(f, eopts)
		},
		mux: func(_ container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if isMatroska(opts.Container) {
				return mkaMuxer(dst, opts), nil
			}
			return ogg.NewMuxer(dst, &ogg.MuxerOptions{Tags: opts.Tags}), nil
		},
		container: func(name string) (string, error) {
			if mt, ok := matroskaContainer(name, true); ok {
				return mt, nil
			}
			return "", waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: vorbis container %q: want mka or webm", name))
		},
	},
	{
		name:        "aiff",
		exts:        []string{"aif", "aiff", "aifc", "afc"},
		live:        false,
		mediaType:   "audio/aiff",
		headerBytes: 54,
		codecID:     codec.PCM,
		plan: func(f audio.Format, _ TranscodeOptions) (string, int, int, error) {
			cfg, err := aiff.DefaultConfig(f)
			if err != nil {
				return "", 0, 0, err
			}
			return pcm.Version, cfg.BytesPerFrame(f.Channels), 0, nil
		},
		encode: func(f audio.Format, _ TranscodeOptions) (codec.Encoder, error) {
			cfg, err := aiff.DefaultConfig(f)
			if err != nil {
				return nil, err
			}
			return pcm.NewEncoder(cfg, f)
		},
		mux: func(_ container.Track, _ TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			return aiff.NewMuxer(dst), nil
		},
	},
	{
		name:      "flac",
		exts:      []string{"flac"},
		live:      true,
		mediaType: "audio/flac",
		codecID:   codec.FLAC,
		adjust: func(spec *dsp.ChainSpec, src audio.Format, opts TranscodeOptions) {
			if level, err := flacLevel(opts); err == nil {
				spec.FrameSize = flac.EncoderBlockSize(level)
			}
			if src.Type == audio.Float && opts.BitDepth == 0 {
				spec.BitDepth = 24
			}
		},
		plan: func(f audio.Format, opts TranscodeOptions) (string, int, int, error) {
			level, err := flacLevel(opts)
			if err != nil {
				return "", 0, 0, err
			}
			if _, err := flac.NewEncoder(f, &flac.EncoderOptions{Level: level}); err != nil {
				return "", 0, 0, err
			}
			return flac.EncoderVersion(level), 0, 0, nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			level, err := flacLevel(opts)
			if err != nil {
				return nil, err
			}
			return flac.NewEncoder(f, &flac.EncoderOptions{Level: level})
		},
		mux: func(_ container.Track, opts TranscodeOptions, enc codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if isMatroska(opts.Container) {
				return mkaMuxer(dst, opts), nil
			}
			if opts.Container == "ogg" {
				return ogg.NewMuxer(dst, &ogg.MuxerOptions{Tags: opts.Tags}), nil
			}
			mo := flacn.MuxerOptions{Tags: opts.Tags}
			if fe, ok := enc.(*flac.Encoder); ok {
				mo.MD5 = fe.MD5
			}
			return flacn.NewMuxer(dst, &mo), nil
		},
		container: func(name string) (string, error) {
			if name == "ogg" {
				return "audio/ogg", nil
			}
			if mt, ok := matroskaContainer(name, false); ok {
				return mt, nil
			}
			return "", waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: flac container %q: want mka or ogg", name))
		},
		hls: &hlsOutput{
			codecs: "fLaC",
			encode: func(f audio.Format, opts TranscodeOptions, startSample int64) (codec.Encoder, error) {
				level, err := flacLevel(opts)
				if err != nil {
					return nil, err
				}
				return flac.NewEncoder(f, &flac.EncoderOptions{
					Level:      level,
					FirstFrame: startSample / int64(flac.EncoderBlockSize(level)),
				})
			},
		},
	},
	{
		name:        "mp3",
		exts:        []string{"mp3", "mpga"},
		live:        true,
		headerBytes: 1024,
		lossy:       true,
		mediaType:   "audio/mpeg",
		codecID:     codec.MP3,
		adjust: func(spec *dsp.ChainSpec, src audio.Format, _ TranscodeOptions) {
			spec.FrameSize = 1152
			spec.BitDepth = 0
			spec.Float = true
			foldWideToStereo(spec, src)
		},
		plan: func(f audio.Format, opts TranscodeOptions) (string, int, int, error) {
			eo, err := mp3EncoderOptions(opts)
			if err != nil {
				return "", 0, 0, err
			}
			enc, err := mp3.NewEncoder(f, eo)
			if err != nil {
				return "", 0, 0, err
			}
			return mp3.EncoderVersion, 0, enc.Bitrate(), nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			eo, err := mp3EncoderOptions(opts)
			if err != nil {
				return nil, err
			}
			return mp3.NewEncoder(f, eo)
		},
		mux: func(t container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			return mpa.NewMuxer(dst, &mpa.MuxerOptions{Delay: int(t.Delay), VBR: opts.MP3VBR, Tags: opts.Tags}), nil
		},
	},
	{
		name:        "aac",
		exts:        []string{"m4a", "aac", "m4b"},
		live:        true,
		lossy:       true,
		mediaType:   "audio/mp4",
		headerBytes: 700,
		codecID:     codec.AACLC,
		adjust: func(spec *dsp.ChainSpec, src audio.Format, _ TranscodeOptions) {
			spec.FrameSize = 1024
			spec.BitDepth = 0
			spec.Float = true
			foldWideToStereo(spec, src)
		},
		plan: func(f audio.Format, opts TranscodeOptions) (string, int, int, error) {
			if _, err := aacContainerMediaType(opts.Container); err != nil {
				return "", 0, 0, err
			}
			enc, err := aac.NewEncoder(f, &aac.EncoderOptions{Bitrate: opts.AACBitrate})
			if err != nil {
				return "", 0, 0, err
			}
			return aac.EncoderVersion, 0, enc.Bitrate(), nil
		},
		encode: func(f audio.Format, opts TranscodeOptions) (codec.Encoder, error) {
			if _, err := aacContainerMediaType(opts.Container); err != nil {
				return nil, err
			}
			return aac.NewEncoder(f, &aac.EncoderOptions{Bitrate: opts.AACBitrate})
		},
		mux: func(_ container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if isMatroska(opts.Container) {
				return mkaMuxer(dst, opts), nil
			}
			if opts.Container == "adts" {
				return adts.NewMuxer(dst), nil
			}
			if opts.Container == ContainerProgressive {
				return mp4.NewProgressiveMuxer(dst, mp4MuxerOptions(opts)), nil
			}
			return mp4.NewMuxer(dst, mp4MuxerOptions(opts)), nil
		},
		container: aacContainerMediaType,
		hls: &hlsOutput{
			codecs: "mp4a.40.2",
			delay:  aac.EncoderDelay,
			encode: func(f audio.Format, opts TranscodeOptions, _ int64) (codec.Encoder, error) {
				return aac.NewEncoder(f, &aac.EncoderOptions{Bitrate: opts.AACBitrate})
			},
		},
	},
	{
		name:      "alac",
		exts:      []string{},
		writeExt:  "m4a",
		live:      true,
		mediaType: "audio/mp4",
		codecID:   codec.ALAC,
		adjust: func(spec *dsp.ChainSpec, src audio.Format, opts TranscodeOptions) {
			spec.FrameSize = alac.FrameSize
			if opts.BitDepth != 0 {
				return
			}
			if src.Type == audio.Float {
				spec.BitDepth = 24
			} else if d := alacSnapDepth(src.BitDepth); d != src.BitDepth {
				spec.BitDepth = d
			}
		},
		plan: func(f audio.Format, _ TranscodeOptions) (string, int, int, error) {
			if _, err := alac.NewEncoder(f, nil); err != nil {
				return "", 0, 0, err
			}
			return alac.EncoderVersion, 0, 0, nil
		},
		encode: func(f audio.Format, _ TranscodeOptions) (codec.Encoder, error) {
			return alac.NewEncoder(f, nil)
		},
		mux: func(_ container.Track, opts TranscodeOptions, _ codec.Encoder, dst io.Writer) (container.Muxer, error) {
			if opts.Container == ContainerProgressive {
				return mp4.NewProgressiveMuxer(dst, mp4MuxerOptions(opts)), nil
			}
			return mp4.NewMuxer(dst, mp4MuxerOptions(opts)), nil
		},
		container: func(name string) (string, error) {
			if name == ContainerProgressive || name == ContainerFragmented {
				return mp4MediaType, nil
			}
			return "", waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("waxflow: alac container %q: want progressive or fragmented", name))
		},
		hls: &hlsOutput{
			codecs: "alac",
			encode: func(f audio.Format, _ TranscodeOptions, _ int64) (codec.Encoder, error) {
				return alac.NewEncoder(f, nil)
			},
		},
	},
}

func (e *Engine) logImplicitDownmix(opts TranscodeOptions, src, out audio.Format) {
	if opts.Channels == 0 && out.Channels < src.Channels {
		e.log.Warn("downmixed to fit the output format",
			"format", opts.Format, "source", src.Channels, "out", out.Channels)
	}
}

func foldWideToStereo(spec *dsp.ChainSpec, src audio.Format) {
	if spec.Channels == 0 && src.Channels > 2 {
		spec.Channels = 2
	}
}

func aacContainerMediaType(name string) (string, error) {
	switch name {
	case "":
		return mp4MediaType, nil
	case "adts":
		return "audio/aac", nil
	case ContainerProgressive, ContainerFragmented:
		return mp4MediaType, nil
	}
	if mt, ok := matroskaContainer(name, false); ok {
		return mt, nil
	}
	return "", waxerr.New(waxerr.CodeInvalidRequest,
		fmt.Sprintf("waxflow: aac container %q: want adts, progressive, fragmented, or mka", name))
}

const (
	ContainerProgressive = "progressive"
	ContainerFragmented  = "fragmented"
)

const mp4MediaType = "audio/mp4"

// FileOutputContainer resolves the container a file output should be written with: the caller's explicit choice, or the flat MP4 form when the plan says this is an mp4-family output and the caller expressed no preference.
func FileOutputContainer(requested string, plan *TranscodePlan) string {
	if requested == "" && plan != nil && plan.MediaType == mp4MediaType {
		return ContainerProgressive
	}
	return requested
}

func containerLive(rowLive bool, container string) bool {
	if container == ContainerProgressive {
		return false
	}
	return rowLive
}

func matroskaContainer(name string, webmOK bool) (mediaType string, ok bool) {
	switch name {
	case "mka":
		return "audio/x-matroska", true
	case "webm":
		if webmOK {
			return "audio/webm", true
		}
	}
	return "", false
}

func isMatroska(name string) bool { return name == "mka" || name == "webm" }

func mkaMuxer(dst io.Writer, opts TranscodeOptions) container.Muxer {
	return mka.NewMuxer(dst, &mka.MuxerOptions{WebM: opts.Container == "webm", Tags: opts.Tags})
}

func mkaPCMConfig(f audio.Format) pcm.Config {
	if f.Type == audio.Float {
		return pcm.Config{Encoding: pcm.Float, Bits: 32}
	}
	bits := pcm.ContainerBits(f.BitDepth)
	cfg := pcm.Config{Encoding: pcm.SignedInt, Bits: bits}
	if f.BitDepth != bits {
		cfg.ValidBits = f.BitDepth
	}
	return cfg
}

func mp4MuxerOptions(opts TranscodeOptions) *mp4.MuxerOptions {
	if len(opts.Tags) == 0 && len(opts.Chapters) == 0 && opts.Art == nil {
		return nil
	}
	return &mp4.MuxerOptions{Tags: opts.Tags, Chapters: opts.Chapters, Art: opts.Art}
}

func alacSnapDepth(d int) int {
	switch {
	case d <= 16:
		return 16
	case d <= 20:
		return 20
	case d <= 24:
		return 24
	default:
		return 32
	}
}

func opusEncoderOptions(opts TranscodeOptions) (*opus.EncoderOptions, error) {
	sig, err := opusSignal(opts)
	if err != nil {
		return nil, err
	}
	return &opus.EncoderOptions{
		Bitrate:    opusBitrate(opts),
		Complexity: opts.OpusComplexity,
		VBR:        opts.OpusVBR,
		Signal:     sig,
	}, nil
}

func vorbisEncoderOptions(opts TranscodeOptions) (*vorbis.EncoderOptions, error) {
	return &vorbis.EncoderOptions{
		Quality: opts.VorbisQuality,
		Bitrate: opts.VorbisBitrate,
	}, nil
}

func opusSignal(opts TranscodeOptions) (opus.Signal, error) {
	switch opts.OpusSignal {
	case "", "auto":
		return opus.SignalAuto, nil
	case "voice":
		return opus.SignalVoice, nil
	case "music":
		return opus.SignalMusic, nil
	}
	return opus.SignalAuto, waxerr.New(waxerr.CodeInvalidRequest,
		fmt.Sprintf("opus signal hint %q is not auto, voice, or music", opts.OpusSignal))
}

func opusBitrate(opts TranscodeOptions) int {
	if opts.OpusBitrate == 0 {
		return opus.DefaultBitrate
	}
	return opts.OpusBitrate
}

func mp3EncoderOptions(opts TranscodeOptions) (*mp3.EncoderOptions, error) {
	bitrate := opts.MP3Bitrate
	if bitrate == 0 {
		bitrate = mp3.DefaultBitrate
	}
	if bitrate < 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("waxflow: MP3 bit rate %d is negative", bitrate))
	}
	return &mp3.EncoderOptions{Bitrate: bitrate, VBR: opts.MP3VBR}, nil
}

func flacLevel(opts TranscodeOptions) (int, error) {
	switch {
	case opts.FLACLevel == FLACLevelDefault:
		return flac.DefaultEncoderLevel, nil
	case opts.FLACLevel == FLACLevelFastest:
		return 0, nil
	case opts.FLACLevel >= 1 && opts.FLACLevel <= 8:
		return opts.FLACLevel, nil
	}
	return 0, waxerr.New(waxerr.CodeInvalidRequest,
		fmt.Sprintf("waxflow: FLAC level %d outside -1..8", opts.FLACLevel))
}

// DefaultLiveFormat returns the output format that format=auto resolves to when a transcode is required: the first registered output with a streaming form.
func DefaultLiveFormat() string {
	for _, o := range outputs {
		if o.live {
			return o.name
		}
	}
	return ""
}

// OutputInfo describes one entry of the writer-side capability table.
type OutputInfo struct {
	Name string
	Exts []string
	Live bool
}

// Outputs lists the registered output formats, in table order.
func Outputs() []OutputInfo {
	infos := make([]OutputInfo, len(outputs))
	for i, o := range outputs {
		infos[i] = OutputInfo{Name: o.name, Exts: append([]string{}, o.exts...), Live: o.live}
	}
	return infos
}

// OutputFormats lists the registered output format names, in table order.
func OutputFormats() []string {
	names := make([]string, len(outputs))
	for i, o := range outputs {
		names[i] = o.name
	}
	return names
}

// LossyFormat reports whether the named output format is lossy (accepts bitrate/q), and whether it is a registered format at all.
func LossyFormat(name string) (lossy, known bool) {
	for _, o := range outputs {
		if o.name == name {
			return o.lossy, true
		}
	}
	return false, false
}

// OutputFormatForExt maps a file extension (with or without the leading dot, any case) to the output format name that writes it, or "" when no registered output claims the extension.
func OutputFormatForExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, o := range outputs {
		for _, e := range o.exts {
			if e == ext {
				return o.name
			}
		}
	}
	return ""
}

// OutputContainerForExt maps a container-selecting output extension (one that names a container form rather than a top-level format) to the format and container override it implies.
func OutputContainerForExt(ext string) (format, container string, ok bool) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "mka", "mkv":
		return "flac", "mka", true
	case "webm":
		return "opus", "webm", true
	}
	return "", "", false
}

// OutputExt is the file extension for an output written as format in container (empty for the format's default form), without the leading dot.
func OutputExt(format, container string) string {
	row, err := outputRow(format)
	if err != nil {
		return outputExtFallback
	}
	if ext, ok := containerExt(container); ok {
		return ext
	}
	if row.writeExt != "" {
		return row.writeExt
	}
	if len(row.exts) > 0 {
		return row.exts[0]
	}
	return outputExtFallback
}

const outputExtFallback = "bin"

func containerExt(name string) (ext string, ok bool) {
	switch name {
	case "adts":
		return "aac", true
	case "mka":
		return "mka", true
	case "webm":
		return "webm", true
	case "ogg":
		return "oga", true
	}
	return "", false
}

func outputRow(name string) (*output, error) {
	if name == "" {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "waxflow: no output format requested")
	}
	for i := range outputs {
		if outputs[i].name == name {
			return &outputs[i], nil
		}
	}
	return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
		fmt.Sprintf("waxflow: unsupported output format %q (available: %s)", name, strings.Join(OutputFormats(), ", ")))
}
