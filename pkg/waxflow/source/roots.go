package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"sync"

	"novelhub/pkg/waxflow/waxerr"
)

// DefaultMaxBytes is the per-source open cap when none is configured.
const DefaultMaxBytes = 4 << 30

// Root names a library directory.
type Root struct {
	Name string
	Path string
}

type mounted struct {
	path string
	root *os.Root
}

// Roots resolves root-relative references.
type Roots struct {
	mu       sync.RWMutex
	reloadMu sync.Mutex
	maxBytes int64
	order    []string
	roots    map[string]mounted
}

func validateRootName(name string) error {
	if name == "" || strings.ContainsAny(name, "/:") {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("source: root name %q must be non-empty without '/' or ':'", name))
	}
	return nil
}

func openMount(root Root) (mounted, error) {
	if err := validateRootName(root.Name); err != nil {
		return mounted{}, err
	}
	or, err := os.OpenRoot(root.Path)
	if err != nil {
		return mounted{}, waxerr.Wrap(waxerr.CodeInvalidRequest,
			fmt.Sprintf("source: opening root %q at %s", root.Name, root.Path), err)
	}
	return mounted{path: root.Path, root: or}, nil
}

// OpenRoots opens the named roots.
func OpenRoots(roots []Root, maxBytes int64) (*Roots, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	r := &Roots{maxBytes: maxBytes, roots: make(map[string]mounted, len(roots))}
	for _, root := range roots {
		if _, dup := r.roots[root.Name]; dup {
			r.Close()
			return nil, waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("source: duplicate root name %q", root.Name))
		}
		m, err := openMount(root)
		if err != nil {
			r.Close()
			return nil, err
		}
		r.roots[root.Name] = m
		r.order = append(r.order, root.Name)
	}
	return r, nil
}

// ReloadResult reports what a Reload changed.
type ReloadResult struct {
	Added, Removed, Changed, Roots []string
}

var errReloadAfterClose = waxerr.Wrap(waxerr.CodeInternal, "source: reload after close", fs.ErrClosed)

// Reload reconciles the live root set to desired and replaces the size cap with maxBytes (0 means DefaultMaxBytes, as OpenRoots).
func (r *Roots) Reload(desired []Root, maxBytes int64) (ReloadResult, error) {
	r.reloadMu.Lock()
	defer r.reloadMu.Unlock()

	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}

	r.mu.RLock()
	if r.roots == nil {
		r.mu.RUnlock()
		return ReloadResult{}, errReloadAfterClose
	}
	current := make(map[string]string, len(r.roots))
	for name, m := range r.roots {
		current[name] = m.path
	}
	r.mu.RUnlock()

	seen := make(map[string]bool, len(desired))
	var toOpen []Root
	var result ReloadResult
	for _, root := range desired {
		if err := validateRootName(root.Name); err != nil {
			return ReloadResult{}, err
		}
		if seen[root.Name] {
			return ReloadResult{}, waxerr.New(waxerr.CodeInvalidRequest,
				fmt.Sprintf("source: duplicate root name %q", root.Name))
		}
		seen[root.Name] = true
		result.Roots = append(result.Roots, root.Name)
		if oldPath, ok := current[root.Name]; ok {
			if oldPath == root.Path {
				continue
			}
			result.Changed = append(result.Changed, root.Name)
		} else {
			result.Added = append(result.Added, root.Name)
		}
		toOpen = append(toOpen, root)
	}
	for name := range current {
		if !seen[name] {
			result.Removed = append(result.Removed, name)
		}
	}
	slices.Sort(result.Removed)

	opened := make(map[string]mounted, len(toOpen))
	for _, root := range toOpen {
		m, err := openMount(root)
		if err != nil {
			for _, o := range opened {
				o.root.Close()
			}
			return ReloadResult{}, err
		}
		opened[root.Name] = m
	}

	r.mu.Lock()
	if r.roots == nil {
		r.mu.Unlock()
		for _, m := range opened {
			m.root.Close()
		}
		return ReloadResult{}, errReloadAfterClose
	}
	old := r.roots
	newRoots := make(map[string]mounted, len(desired))
	newOrder := make([]string, 0, len(desired))
	for _, root := range desired {
		if m, ok := opened[root.Name]; ok {
			newRoots[root.Name] = m
		} else {
			newRoots[root.Name] = old[root.Name]
		}
		newOrder = append(newOrder, root.Name)
	}
	var toClose []*os.Root
	for name, m := range old {
		if _, replaced := opened[name]; replaced {
			toClose = append(toClose, m.root)
		} else if _, kept := newRoots[name]; !kept {
			toClose = append(toClose, m.root)
		}
	}
	r.roots = newRoots
	r.order = newOrder
	r.maxBytes = maxBytes
	r.mu.Unlock()

	for _, or := range toClose {
		or.Close()
	}
	return result, nil
}

// Names lists the configured root names in configuration order.
func (r *Roots) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.order...)
}

// Close releases the held root directories and clears the set, so a double-close is a no-op and any post-close Resolve is an ordinary unknown-root error rather than a use of a closed handle.
func (r *Roots) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var first error
	for _, m := range r.roots {
		if err := m.root.Close(); err != nil && first == nil {
			first = err
		}
	}
	r.roots = nil
	r.order = nil
	return first
}

// Resolve implements Resolver for "<root>/<relative/path>" references.
func (r *Roots) Resolve(_ context.Context, ref string) (*File, error) {
	if ref == "" {
		return nil, waxerr.New(waxerr.CodeInvalidRequest, "source: empty source reference")
	}
	if s, ok := scheme(ref); ok {
		return nil, unsupportedScheme(s)
	}
	name, rel, ok := strings.Cut(ref, "/")
	if !ok || rel == "" {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("source: reference %q is not <root>/<path>", ref))
	}

	r.mu.RLock()
	m, exists := r.roots[name]
	if !exists {
		msg := fmt.Sprintf("source: unknown root %q (configured: %s)", name, strings.Join(r.order, ", "))
		r.mu.RUnlock()
		return nil, waxerr.New(waxerr.CodeNotFound, msg)
	}
	maxBytes := r.maxBytes
	r.mu.RUnlock()

	f, err := m.root.OpenFile(rel, os.O_RDONLY|openNonblock, 0)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return nil, waxerr.Wrap(waxerr.CodeNotFound, "source: no such file", err)
		case errors.Is(err, fs.ErrClosed):
			return nil, waxerr.Wrap(waxerr.CodeNotFound, "source: root reloaded away", err)
		case errors.Is(err, fs.ErrPermission):
			return nil, waxerr.Wrap(waxerr.CodeSourceUnreadable, "source: permission denied", err)
		default:
			return nil, waxerr.Wrap(waxerr.CodeInvalidRequest, "source: unresolvable path", err)
		}
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
	if fi.Size() > maxBytes {
		f.Close()
		return nil, waxerr.New(waxerr.CodePayloadTooLarge,
			fmt.Sprintf("source: %d bytes exceeds the %d-byte source cap", fi.Size(), maxBytes))
	}
	return &File{
		Ref: ref,
		Ext: extHint(rel),
		ID:  Identity{Size: fi.Size(), MtimeNS: fi.ModTime().UnixNano()},
		f:   f,
	}, nil
}

func modeWord(m fs.FileMode) string {
	switch {
	case m.IsDir():
		return "directory"
	case m&fs.ModeNamedPipe != 0:
		return "named pipe"
	case m&fs.ModeDevice != 0:
		return "device"
	case m&fs.ModeSocket != 0:
		return "socket"
	default:
		return "special file"
	}
}
