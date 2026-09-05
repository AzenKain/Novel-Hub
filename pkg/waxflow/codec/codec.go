// Package codec defines the compressed-domain types and the Decoder and Encoder interfaces every WaxFlow codec implements (ADR-0005).
package codec

import "novelhub/pkg/waxflow/audio"

// ID names a codec.
type ID string

const (
	PCM    ID = "pcm"
	FLAC   ID = "flac"
	ALAC   ID = "alac"
	MP3    ID = "mp3"
	AACLC  ID = "aac-lc"
	Opus   ID = "opus"
	Vorbis ID = "vorbis"
)

// Packet is one compressed unit as a codec defines it: a FLAC frame, an MP3 frame, an Opus packet, a run of PCM frames.
type Packet struct {
	Data []byte
	PTS  int64
	Dur  int64
	Sync bool
}

// Trailer carries gapless finalization from an encoder to a muxer: Samples is the true source length, Delay the encoder priming to trim from the front, Padding the trailing samples to trim.
type Trailer struct {
	Samples int64
	Delay   int64
	Padding int64
}

// Decoder turns packets into PCM buffers.
type Decoder interface {
	Decode(pkt []byte, emit func(*audio.Buffer) error) error
	Drain(emit func(*audio.Buffer) error) error
	Reset()
}

// Releaser is optionally implemented by decoders and encoders that hold pooled scratch buffers.
type Releaser interface {
	Release()
}

// Encoder turns PCM buffers into packets.
type Encoder interface {
	InputFormat() audio.Format
	FrameSize() int
	Encode(src *audio.Buffer, emit func(Packet) error) error
	Finish(emit func(Packet) error) (Trailer, error)
	CodecConfig() []byte
}
