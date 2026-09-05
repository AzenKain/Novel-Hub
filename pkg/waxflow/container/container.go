// Package container defines the demuxer, muxer, and seeker interfaces and the track-routed packet model that wrap codec-level packets (ADR-0005).
package container

import (
	"bytes"
	"context"
	"io"
	"os"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/codec"
	"novelhub/pkg/waxflow/waxerr"
)

// Source is random-access input.
type Source interface {
	io.ReaderAt
	Size() int64
}

// Contextual is implemented by a Source whose reads can be bound to a context, as a network-backed source can.
type Contextual interface {
	WithContext(ctx context.Context) Source
}

// BindContext binds ctx to src when src is Contextual, and returns src unchanged otherwise.
func BindContext(ctx context.Context, src Source) Source {
	if c, ok := src.(Contextual); ok {
		return c.WithContext(ctx)
	}
	return src
}

// FileSource wraps an open file as a Source.
func FileSource(f *os.File) (Source, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, waxerr.Wrap(waxerr.CodeSourceUnreadable, "stat source", err)
	}
	return readerAtSource{f, fi.Size()}, nil
}

// BytesSource wraps an in-memory blob as a Source, mainly for tests and probes of spooled uploads.
func BytesSource(b []byte) Source {
	return bytes.NewReader(b)
}

type readerAtSource struct {
	io.ReaderAt
	size int64
}

func (s readerAtSource) Size() int64 { return s.size }

// ReadFull reads exactly len(p) bytes from src at off.
func ReadFull(src io.ReaderAt, p []byte, off int64) error {
	n, err := src.ReadAt(p, off)
	switch {
	case n == len(p):
		return nil
	case err == nil || err == io.EOF:
		return io.ErrUnexpectedEOF
	default:
		return err
	}
}

// Track describes one elementary stream in a container.
type Track struct {
	ID             int
	Codec          codec.ID
	CodecConfig    []byte
	Fmt            audio.Format
	Samples        int64
	Delay          int64
	Padding        int64
	SamplesExact   bool
	SourceBitDepth int
	Default        bool
}

// Packet is a codec packet routed to a track.
type Packet struct {
	Track int
	codec.Packet
}

// Demuxer yields a container's tracks and packets.
type Demuxer interface {
	Tracks() []Track
	ReadPacket(pkt *Packet) error
}

// Seeker is implemented by demuxers that can reposition.
type Seeker interface {
	SeekSample(track int, sample int64) (landed int64, err error)
}

// Indexer is implemented by demuxers whose seeking builds an expensive source index (exact frame tables, seek tables) worth persisting across sessions: the cacheDir/idx sidecar.
type Indexer interface {
	IndexSnapshot() []byte
	RestoreIndex(blob []byte) bool
}

// Warning is a structured note about input this decoder accepted but a caller should know about, surfaced through probe results.
type Warning struct {
	Offset int64
	Msg    string
}

// Warner is implemented by demuxers that record Warnings: tolerated damage, or a decoder limitation against a well-formed file.
type Warner interface {
	Warnings() []Warning
}

// Muxer writes one audio track to a container.
type Muxer interface {
	Begin(tracks []Track) error
	WritePacket(pkt Packet) error
	End(trailer codec.Trailer) error
	NeedsSeek() bool
}
