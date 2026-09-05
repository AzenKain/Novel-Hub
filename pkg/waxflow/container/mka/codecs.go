package mka

import (
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/codec/aac"
	"novelhub/pkg/waxflow/codec/flac"
	"novelhub/pkg/waxflow/codec/opus"
	"novelhub/pkg/waxflow/codec/pcm"
	"novelhub/pkg/waxflow/codec/vorbis"
	"novelhub/pkg/waxflow/waxerr"
)

type trackEntry struct {
	number      uint64
	trackType   uint64
	codecID     string
	codecName   string
	codecPriv   []byte
	rate        int
	channels    int
	bitDepth    int
	codecDelay  int64
	seekPreRoll int64
	def         bool
}

type codecSetup struct {
	id     codec.ID
	config []byte
	fmt    audio.Format

	pcmBytesPerFrame int
	aacFrameLength   int
	vorbisCfg        vorbis.Config
	vorbisModeBits   int
	haveVorbis       bool

	warning string
}

func mkvCodecID(codecID string) codec.ID {
	switch {
	case codecID == "A_OPUS":
		return codec.Opus
	case codecID == "A_VORBIS":
		return codec.Vorbis
	case codecID == "A_FLAC":
		return codec.FLAC
	case codecID == "A_AAC" || hasPrefix(codecID, "A_AAC/"):
		return codec.AACLC
	case hasPrefix(codecID, "A_PCM/"):
		return codec.PCM
	default:
		return ""
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }

func resolveCodec(t *trackEntry) (codecSetup, error) {
	switch mkvCodecID(t.codecID) {
	case codec.Opus:
		return setupOpus(t)
	case codec.Vorbis:
		return setupVorbis(t)
	case codec.FLAC:
		return setupFLAC(t)
	case codec.AACLC:
		return setupAAC(t)
	case codec.PCM:
		return setupPCM(t)
	default:
		return codecSetup{}, malformed("codec %q is not one this build decodes", t.codecID)
	}
}

func setupOpus(t *trackEntry) (codecSetup, error) {
	cfg, err := opus.ParseOpusHead(t.codecPriv)
	if err != nil {
		return codecSetup{}, err
	}
	return codecSetup{id: codec.Opus, config: t.codecPriv, fmt: cfg.Format()}, nil
}

func setupVorbis(t *trackEntry) (codecSetup, error) {
	cfg, err := vorbis.ParseConfig(t.codecPriv)
	if err != nil {
		return codecSetup{}, err
	}
	return codecSetup{
		id:             codec.Vorbis,
		config:         t.codecPriv,
		fmt:            cfg.Format(),
		vorbisCfg:      cfg,
		vorbisModeBits: vorbis.ModeBits(cfg),
		haveVorbis:     true,
	}, nil
}

func setupFLAC(t *trackEntry) (codecSetup, error) {
	si, err := flacStreamInfo(t.codecPriv)
	if err != nil {
		return codecSetup{}, err
	}
	parsed, err := flac.ParseStreamInfo(si)
	if err != nil {
		return codecSetup{}, err
	}
	return codecSetup{id: codec.FLAC, config: si, fmt: parsed.PCMFormat()}, nil
}

func setupAAC(t *trackEntry) (codecSetup, error) {
	if len(t.codecPriv) == 0 {
		return codecSetup{}, malformed("AAC track has no CodecPrivate (AudioSpecificConfig)")
	}
	cfg, err := aac.ParseASC(t.codecPriv)
	if err != nil {
		return codecSetup{}, err
	}
	if cfg.Channels == 0 && t.channels >= 1 && t.channels <= 2 {
		cfg.Channels = t.channels
	}
	f, err := cfg.Format()
	if err != nil {
		return codecSetup{}, err
	}
	return codecSetup{
		id:             codec.AACLC,
		config:         t.codecPriv,
		fmt:            f,
		aacFrameLength: cfg.FrameLength,
		warning:        cfg.SBRWarning(),
	}, nil
}

func (s codecSetup) preRoll() int64 {
	switch s.id {
	case codec.Opus:
		return 3840 // 80 ms at 48 kHz (RFC 7845)
	case codec.Vorbis:
		if s.haveVorbis {
			return int64(s.vorbisCfg.LongBlock())
		}
	case codec.AACLC:
		return 1024
	}
	return 0
}

func setupPCM(t *trackEntry) (codecSetup, error) {
	var enc pcm.Encoding
	switch t.codecID {
	case "A_PCM/INT/LIT":
		enc = pcm.SignedInt
	case "A_PCM/INT/BIG":
		enc = pcm.SignedInt
	case "A_PCM/FLOAT/IEEE":
		enc = pcm.Float
	default:
		return codecSetup{}, malformed("unsupported PCM flavor %q", t.codecID)
	}
	if t.bitDepth == 0 {
		return codecSetup{}, malformed("PCM track has no BitDepth")
	}
	c := pcm.Config{Encoding: enc, Bits: t.bitDepth, BigEndian: t.codecID == "A_PCM/INT/BIG"}
	cfgBytes, err := c.MarshalBinary()
	if err != nil {
		return codecSetup{}, waxerr.Wrap(waxerr.CodeUnsupportedFormat, "mka: unusable PCM config", err)
	}
	if t.channels < 1 || t.channels > audio.MaxChannels {
		return codecSetup{}, malformed("PCM track with %d channels", t.channels)
	}
	f := c.PCMFormat(t.rate, t.channels, audio.DefaultLayout(t.channels))
	return codecSetup{
		id:               codec.PCM,
		config:           cfgBytes,
		fmt:              f,
		pcmBytesPerFrame: c.BytesPerFrame(t.channels),
	}, nil
}

func flacStreamInfo(priv []byte) ([]byte, error) {
	b := priv
	if len(b) >= 4 && string(b[:4]) == "fLaC" {
		b = b[4:]
	}
	if len(b) == flac.StreamInfoLen {
		return append([]byte(nil), b...), nil
	}
	if len(b) < 4 {
		return nil, malformed("FLAC CodecPrivate of %d bytes too short", len(priv))
	}
	if typ := b[0] & 0x7F; typ != 0 {
		return nil, malformed("FLAC CodecPrivate first block is type %d, want STREAMINFO", typ)
	}
	length := int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if length != flac.StreamInfoLen || 4+length > len(b) {
		return nil, malformed("FLAC CodecPrivate STREAMINFO length %d", length)
	}
	return append([]byte(nil), b[4:4+length]...), nil
}

func (d *Demuxer) frameSamples(data []byte) int64 {
	switch d.setup.id {
	case codec.Opus:
		n, err := opus.PacketSamples(data)
		if err != nil {
			return 0
		}
		return int64(n)
	case codec.AACLC:
		if d.setup.aacFrameLength > 0 {
			return int64(d.setup.aacFrameLength)
		}
		return 1024
	case codec.PCM:
		if d.setup.pcmBytesPerFrame <= 0 {
			return 0
		}
		return int64(len(data) / d.setup.pcmBytesPerFrame)
	case codec.FLAC:
		fi, err := flac.ParseFrameHeader(data)
		if err != nil {
			return 0
		}
		return int64(fi.BlockSize)
	case codec.Vorbis:
		if !d.setup.haveVorbis {
			return 0
		}
		block, ok := vorbis.PacketBlockSize(d.setup.vorbisCfg, d.setup.vorbisModeBits, data)
		if !ok {
			return 0
		}
		dur := int64(0)
		if d.vorbisPrevBlock != 0 {
			dur = int64(d.vorbisPrevBlock+block) / 4
		}
		d.vorbisPrevBlock = block
		return dur
	default:
		return 0
	}
}
