// Package source resolves source references onto opened, validated files.
package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/waxerr"
)

// Identity pins the exact bytes a reference resolved to: file size plus modification time in nanoseconds.
type Identity struct {
	Size    int64
	MtimeNS int64
}

// String renders the identity in the canonical "size-mtimeNS" form used in signed URLs and cache keys.
func (id Identity) String() string {
	return strconv.FormatInt(id.Size, 10) + "-" + strconv.FormatInt(id.MtimeNS, 10)
}

// ParseIdentity parses the canonical String form.
func ParseIdentity(s string) (Identity, error) {
	sizeStr, mtimeStr, ok := strings.Cut(s, "-")
	if !ok {
		return Identity{}, waxerr.New(waxerr.CodeInvalidRequest, "source: identity is not size-mtimeNS")
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size < 0 {
		return Identity{}, waxerr.New(waxerr.CodeInvalidRequest, "source: malformed identity size")
	}
	mtime, err := strconv.ParseInt(mtimeStr, 10, 64)
	if err != nil {
		return Identity{}, waxerr.New(waxerr.CodeInvalidRequest, "source: malformed identity mtime")
	}
	return Identity{Size: size, MtimeNS: mtime}, nil
}

// File is an opened, validated source.
type File struct {
	Ref string
	Ext string
	ID  Identity

	f *os.File
}

// ReadAt implements container.Source.
func (f *File) ReadAt(p []byte, off int64) (int, error) { return f.f.ReadAt(p, off) }

// Size implements container.Source, from the identity captured at open.
func (f *File) Size() int64 { return f.ID.Size }

// ReadSeeker exposes the open file for http.ServeContent (direct play).
func (f *File) ReadSeeker() io.ReadSeeker { return f.f }

// ModTime returns the identity mtime as a time, for HTTP validators.
func (f *File) ModTime() time.Time { return time.Unix(0, f.ID.MtimeNS) }

// Close releases the underlying file.
func (f *File) Close() error { return f.f.Close() }

var _ container.Source = (*File)(nil)

// Resolver opens the file behind a source reference.
type Resolver interface {
	Resolve(ctx context.Context, ref string) (*File, error)
}

func scheme(ref string) (string, bool) {
	head := ref
	if i := strings.IndexByte(ref, '/'); i >= 0 {
		head = ref[:i]
	}
	s, _, ok := strings.Cut(head, ":")
	return s, ok
}

func unsupportedScheme(s string) error {
	switch s {
	case "upload":
		return waxerr.New(waxerr.CodeUnsupportedSource, "source: upload references need the daemon's upload spool (uploads are disabled here)")
	case "pid":
		return waxerr.New(waxerr.CodeUnsupportedSource, "source: pid references require a build with a catalog resolver")
	default:
		return waxerr.New(waxerr.CodeUnsupportedSource, fmt.Sprintf("source: unknown source scheme %q", s))
	}
}

func extHint(rel string) string {
	return strings.TrimPrefix(strings.ToLower(path.Ext(rel)), ".")
}

// OpenLocal opens a trusted local path (the upload spool) as a resolved File with the same regular-file validation as root resolution.
func OpenLocal(ref, path, name string) (*File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|openNonblock, 0)
	if err != nil {
		return nil, waxerr.Wrap(waxerr.CodeNotFound, "source: no such file", err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, waxerr.Wrap(waxerr.CodeSourceUnreadable, "source: stat", err)
	}
	if !fi.Mode().IsRegular() {
		f.Close()
		return nil, waxerr.New(waxerr.CodeUnsupportedSource,
			fmt.Sprintf("source: %q is a %s, not a regular file", ref, modeWord(fi.Mode())))
	}
	return &File{
		Ref: ref,
		Ext: extHint(name),
		ID:  Identity{Size: fi.Size(), MtimeNS: fi.ModTime().UnixNano()},
		f:   f,
	}, nil
}
