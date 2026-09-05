// Package format identifies containers and opens them as decodable media: a bounded magic-byte sniff over an ordered driver table (extension hints only break ties), then demuxer plus decoder wired into a Media that reads planar PCM chunks and seeks sample-exact.
package format

import (
	"fmt"
	"io"
	"strings"

	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

// Options configures probing and opening.
type Options struct {
	Strict bool
}

// Info is a probe result.
type Info struct {
	Container string
	Tracks    []container.Track
	Chapters  []container.Chapter
	Tags      map[string][]string
	Warnings  []string
}

// Default returns the container's designated default track, or the first track.
func (i *Info) Default() container.Track {
	for _, t := range i.Tracks {
		if t.Default {
			return t
		}
	}
	return i.Tracks[0]
}

// Media is an opened source: probe info plus sample-exact PCM access.
type Media interface {
	Info() *Info
	ReadChunk(dst *audio.Buffer) error
	SeekSample(target int64) (landed int64, err error)
	Close() error
}

// Composite is implemented by a Media assembled from several sources rather than opened from one: a concatenated timeline.
type Composite interface {
	Members() []container.Track
}

const sniffLen = 64 * 1024

var maxSniffNeed = func() int64 {
	need := 0
	for i := range drivers {
		need = max(need, drivers[i].need)
	}
	return int64(min(need, sniffLen))
}()

// Probe identifies src and returns its parsed headers.
func Probe(src container.Source, hint string, opts *Options) (*Info, error) {
	src, d, err := resolve(src, hint)
	if err != nil {
		return nil, err
	}
	demux, err := d.open(src, opts)
	if err != nil {
		return nil, err
	}
	info := buildInfo(d.name, demux)
	if len(info.Tracks) == 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "format: no audio tracks")
	}
	return info, nil
}

// Open identifies src and wires demuxer and decoder into Media.
func Open(src container.Source, hint string, opts *Options) (Media, error) {
	src, d, err := resolve(src, hint)
	if err != nil {
		return nil, err
	}
	demux, err := d.open(src, opts)
	if err != nil {
		return nil, err
	}
	return newMedia(buildInfo(d.name, demux), demux)
}

// OpenDemuxer identifies src and opens its demuxer without wiring a decoder, for callers that move encoded packets rather than samples (the remux rung).
func OpenDemuxer(src container.Source, hint string, opts *Options) (container.Demuxer, *Info, error) {
	src, d, err := resolve(src, hint)
	if err != nil {
		return nil, nil, err
	}
	demux, err := d.open(src, opts)
	if err != nil {
		return nil, nil, err
	}
	info := buildInfo(d.name, demux)
	if len(info.Tracks) == 0 {
		return nil, nil, waxerr.New(waxerr.CodeUnsupportedFormat, "format: no audio tracks")
	}
	return demux, info, nil
}

// FromDemuxer wraps an already-opened demuxer into a Media, for sources that are assembled rather than sniffed from one byte stream.
func FromDemuxer(name string, demux container.Demuxer) (Media, error) {
	info := buildInfo(name, demux)
	if len(info.Tracks) == 0 {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat, "format: no audio tracks")
	}
	return newMedia(info, demux)
}

func buildInfo(name string, demux container.Demuxer) *Info {
	info := &Info{Container: name, Tracks: demux.Tracks()}
	if c, ok := demux.(container.Chapterer); ok {
		info.Chapters = c.Chapters()
	}
	if t, ok := demux.(container.Tagger); ok {
		info.Tags = t.Tags()
	}
	if w, ok := demux.(container.Warner); ok {
		for _, warn := range w.Warnings() {
			if warn.Offset >= 0 {
				info.Warnings = append(info.Warnings, fmt.Sprintf("%s (offset %d)", warn.Msg, warn.Offset))
			} else {
				info.Warnings = append(info.Warnings, warn.Msg)
			}
		}
	}
	return info
}

func resolve(src container.Source, hint string) (container.Source, *driver, error) {
	head, err := readHead(src, max(maxSniffNeed, 10))
	if err != nil {
		return nil, nil, err
	}
	if skip := id3v2Size(head); skip > 0 && skip < src.Size() {
		src = sectionSource{src, skip}
		head, err = readHead(src, maxSniffNeed)
		if err != nil {
			return nil, nil, err
		}
	}
	for i := range drivers {
		if drivers[i].match(head) {
			return src, &drivers[i], nil
		}
	}
	if ext := strings.ToLower(strings.TrimPrefix(hint, ".")); ext != "" {
		for i := range drivers {
			for _, e := range drivers[i].exts {
				if e == ext {
					return src, &drivers[i], nil
				}
			}
		}
	}
	return src, nil, waxerr.New(waxerr.CodeUnsupportedFormat, "format: unrecognized input (no magic bytes matched)")
}

func readHead(src container.Source, n int64) ([]byte, error) {
	if size := src.Size(); size < n {
		n = size
	}
	if n <= 0 {
		return nil, nil
	}
	head := make([]byte, n)
	got, err := src.ReadAt(head, 0)
	if got == len(head) || err == io.EOF {
		return head[:got], nil
	}
	return nil, waxerr.Wrap(waxerr.CodeSourceUnreadable, "format: reading file head", err)
}

func id3v2Size(head []byte) int64 {
	if len(head) < 10 || string(head[:3]) != "ID3" {
		return 0
	}
	for _, b := range head[6:10] {
		if b&0x80 != 0 {
			return 0
		}
	}
	n := int64(head[6])<<21 | int64(head[7])<<14 | int64(head[8])<<7 | int64(head[9])
	n += 10
	if head[5]&0x10 != 0 {
		n += 10
	}
	return n
}

type sectionSource struct {
	src container.Source
	off int64
}

func (s sectionSource) ReadAt(p []byte, off int64) (int, error) {
	return s.src.ReadAt(p, off+s.off)
}

func (s sectionSource) Size() int64 { return s.src.Size() - s.off }
