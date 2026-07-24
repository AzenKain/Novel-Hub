package repositories

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"novelhub/pkg/bookparser"
	"novelhub/pkg/constants"
	"novelhub/pkg/jsonx"
)

type SavedBookFile struct {
	Path      string
	Format    string
	SizeBytes int64
	ModTime   time.Time
}

type BookFileRepository interface {
	SaveBook(ctx context.Context, bookID, originalFilename string, src io.Reader) (*SavedBookFile, error)
	WriteBookMeta(ctx context.Context, bookID string, meta map[string]string) error
	SaveCover(ctx context.Context, bookID, ext string, data []byte) (coverURL string, path string, err error)
	HashSHA256(ctx context.Context, path string) (string, error)
	Exists(ctx context.Context, path string) bool
	RemoveBookDir(ctx context.Context, bookID string) error
	RemoveEmptyBookDirs(ctx context.Context) (int, error)
}

type localBookFileRepository struct {
	baseDir string
}

func NewBookFileRepository(baseDir string) (BookFileRepository, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = filepath.Join(".", "data", "books")
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve books directory: %w", err)
	}
	if err := os.MkdirAll(absBase, 0750); err != nil {
		return nil, fmt.Errorf("create books directory: %w", err)
	}
	return &localBookFileRepository{baseDir: absBase}, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

var unsafeFilenameChars = regexp.MustCompile(`[^\pL\pN._ -]+`)

func (r contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}

func safeBookID(bookID string) (string, error) {
	if bookID == "" || !filepath.IsLocal(bookID) || filepath.Clean(bookID) != bookID {
		return "", fmt.Errorf("invalid book id")
	}
	if strings.ContainsAny(bookID, `/\`) {
		return "", fmt.Errorf("invalid book id")
	}
	return bookID, nil
}

func safeExtension(value string) string {
	ext := strings.ToLower(filepath.Ext(value))
	if strings.TrimSpace(value) != "" && strings.HasPrefix(value, ".") {
		ext = strings.ToLower(value)
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	if ext == "" || len(ext) > 16 {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return ext
}

func safeFilenameStem(value string) string {
	stem := strings.TrimSuffix(filepath.Base(value), filepath.Ext(value))
	stem = unsafeFilenameChars.ReplaceAllString(stem, "_")
	stem = strings.Trim(stem, " ._-")
	if len(stem) > 80 {
		stem = strings.TrimRight(stem[:80], " ._-")
	}
	return stem
}

func (r *localBookFileRepository) withRoot(fn func(*os.Root) error) error {
	root, err := os.OpenRoot(r.baseDir)
	if err != nil {
		return fmt.Errorf("open books root: %w", err)
	}
	defer root.Close()
	return fn(root)
}

func (r *localBookFileRepository) relFromBase(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	rel, err := filepath.Rel(r.baseDir, absPath)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	if rel == "." || !filepath.IsLocal(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes books directory")
	}
	return filepath.ToSlash(rel), nil
}

func (r *localBookFileRepository) SaveBook(ctx context.Context, bookID, originalFilename string, src io.Reader) (*SavedBookFile, error) {
	bookID, err := safeBookID(bookID)
	if err != nil {
		return nil, err
	}
	ext := safeExtension(originalFilename)
	stem := safeFilenameStem(originalFilename)
	if stem == "" {
		stem = bookID
	}
	filename := stem + ext
	relPath := filepath.ToSlash(filepath.Join(bookID, filename))
	absPath := filepath.Join(r.baseDir, bookID, filename)

	if err := r.withRoot(func(root *os.Root) error {
		if err := root.MkdirAll(bookID, 0750); err != nil {
			return err
		}
		for i := 2; ; i++ {
			if _, statErr := root.Stat(relPath); os.IsNotExist(statErr) {
				break
			} else if statErr != nil {
				return statErr
			}
			filename = fmt.Sprintf("%s-%d%s", stem, i, ext)
			relPath = filepath.ToSlash(filepath.Join(bookID, filename))
			absPath = filepath.Join(r.baseDir, bookID, filename)
		}
		file, err := root.OpenFile(relPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, contextReader{ctx: ctx, r: src})
		closeErr := file.Close()
		if copyErr != nil {
			_ = root.Remove(relPath)
			return copyErr
		}
		if closeErr != nil {
			_ = root.Remove(relPath)
			return closeErr
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("save book file: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat saved book file: %w", err)
	}
	return &SavedBookFile{
		Path:      absPath,
		Format:    strings.TrimPrefix(ext, "."),
		SizeBytes: info.Size(),
		ModTime:   info.ModTime(),
	}, nil
}

func (r *localBookFileRepository) WriteBookMeta(ctx context.Context, bookID string, meta map[string]string) error {
	bookID, err := safeBookID(bookID)
	if err != nil {
		return err
	}
	data, err := jsonx.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal book meta: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.withRoot(func(root *os.Root) error {
		if err := root.MkdirAll(bookID, 0750); err != nil {
			return err
		}
		return root.WriteFile(filepath.ToSlash(filepath.Join(bookID, "meta.json")), data, 0600)
	})
}

func (r *localBookFileRepository) SaveCover(ctx context.Context, bookID, ext string, data []byte) (string, string, error) {
	bookID, err := safeBookID(bookID)
	if err != nil {
		return "", "", err
	}
	if len(data) == 0 || int64(len(data)) > constants.HardMaxCoverBytes {
		return "", "", fmt.Errorf("image exceeds size limit")
	}
	trimmed := bytes.TrimSpace(data)
	if (strings.EqualFold(ext, ".svg") || strings.EqualFold(ext, "image/svg+xml")) && (bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml"))) {
		ext = ".svg"
	} else {
		ext, err = bookparser.ValidateImage(data, constants.HardMaxCoverBytes)
		if err != nil {
			return "", "", err
		}
	}
	filename := bookID + ext
	relPath := filepath.ToSlash(filepath.Join(bookID, filename))
	absPath := filepath.Join(r.baseDir, bookID, filename)
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if err := r.withRoot(func(root *os.Root) error {
		if err := root.MkdirAll(bookID, 0750); err != nil {
			return err
		}
		return root.WriteFile(relPath, data, 0600)
	}); err != nil {
		return "", "", fmt.Errorf("save cover: %w", err)
	}
	return "/storage/books/" + bookID + "/" + filename, absPath, nil
}

func (r *localBookFileRepository) HashSHA256(ctx context.Context, path string) (string, error) {
	rel, err := r.relFromBase(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if err := r.withRoot(func(root *os.Root) error {
		file, err := root.Open(rel)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(h, contextReader{ctx: ctx, r: file})
		return err
	}); err != nil {
		return "", fmt.Errorf("hash book file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (r *localBookFileRepository) Exists(ctx context.Context, path string) bool {
	if err := ctx.Err(); err != nil {
		return false
	}
	rel, err := r.relFromBase(path)
	if err != nil {
		return false
	}
	err = r.withRoot(func(root *os.Root) error {
		info, err := root.Stat(rel)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("path is directory")
		}
		return nil
	})
	return err == nil
}

func isRetryableRemoveError(err error) bool {
	return errors.Is(err, syscall.EBUSY) || errors.Is(err, syscall.ENOTEMPTY)
}

func removeAllWithRetry(ctx context.Context, remove func() error) error {
	const (
		attempts = 10
		delay    = 50 * time.Millisecond
	)

	for attempt := 0; ; attempt++ {
		err := remove()
		if err == nil || !isRetryableRemoveError(err) || attempt == attempts {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (r *localBookFileRepository) RemoveBookDir(ctx context.Context, bookID string) error {
	bookID, err := safeBookID(bookID)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	err = r.withRoot(func(root *os.Root) error {
		return removeAllWithRetry(ctx, func() error {
			return root.RemoveAll(bookID)
		})
	})
	if !isRetryableRemoveError(err) {
		return err
	}

	// ponytail: FUSE may retain an open file briefly; remove only an empty directory
	// so a newly imported book with the same ID can never be deleted by this callback.
	time.AfterFunc(500*time.Millisecond, func() {
		_ = r.withRoot(func(root *os.Root) error {
			dir, err := root.Open(bookID)
			if err != nil {
				return err
			}
			_, readErr := dir.Readdirnames(1)
			_ = dir.Close()
			if errors.Is(readErr, io.EOF) {
				return root.Remove(bookID)
			}
			return nil
		})
	})
	return nil
}

func (r *localBookFileRepository) RemoveEmptyBookDirs(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		if !entry.IsDir() {
			continue
		}
		bookID, err := safeBookID(entry.Name())
		if err != nil {
			continue
		}
		dir, err := os.Open(filepath.Join(r.baseDir, bookID))
		if err != nil {
			continue
		}
		_, readErr := dir.Readdirnames(1)
		_ = dir.Close()
		if readErr != io.EOF {
			continue
		}
		if err := r.withRoot(func(root *os.Root) error {
			return removeAllWithRetry(ctx, func() error { return root.Remove(bookID) })
		}); err != nil {
			if errors.Is(err, syscall.ENOTEMPTY) {
				continue
			}
			return removed, err
		}
		removed++
	}
	return removed, nil
}
