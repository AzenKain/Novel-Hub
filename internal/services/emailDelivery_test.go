package services

import (
	"os"
	"path/filepath"
	"testing"

	"novelhub/internal/models"
)

func tempFile(t *testing.T, name string, size int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveEmailAttachmentSkipsUndeliverableFormats(t *testing.T) {
	files := []*models.BookFileEntity{
		{ID: "f-cbz", Format: "cbz", Path: tempFile(t, "a.cbz", 10)},
		{ID: "f-pdf", Format: "pdf", Path: tempFile(t, "a.pdf", 10)},
		{ID: "f-epub", Format: "EPUB", Path: tempFile(t, "a.epub", 10)},
	}
	got, err := resolveEmailAttachment(files, 50)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "f-epub" {
		t.Errorf("picked %q, want f-epub (epub outranks pdf, cbz is not deliverable)", got.ID)
	}
}

// GetFilesByBookId sorts epub first then created_at, so files[0] on a comic-only book is a cbz — which Amazon rejects on arrival.
func TestResolveEmailAttachmentRejectsComicOnlyBook(t *testing.T) {
	files := []*models.BookFileEntity{
		{ID: "f-cbz", Format: "cbz", Path: tempFile(t, "a.cbz", 10)},
		{ID: "f-m4b", Format: "m4b", Path: tempFile(t, "a.m4b", 10)},
	}
	if _, err := resolveEmailAttachment(files, 50); err == nil {
		t.Fatal("want error for a book with no e-reader compatible file, got nil")
	}
}

func TestResolveEmailAttachmentEnforcesSizeLimit(t *testing.T) {
	files := []*models.BookFileEntity{
		{ID: "f-epub", Format: "epub", Path: tempFile(t, "a.epub", 2*1024*1024)},
	}
	if _, err := resolveEmailAttachment(files, 1); err == nil {
		t.Fatal("want error when file exceeds the limit, got nil")
	}
	if _, err := resolveEmailAttachment(files, 50); err != nil {
		t.Fatalf("want no error under the limit, got %v", err)
	}
}

func TestResolveEmailAttachmentDefaultsZeroLimit(t *testing.T) {
	files := []*models.BookFileEntity{
		{ID: "f-epub", Format: "epub", Path: tempFile(t, "a.epub", 1024)},
	}
	if _, err := resolveEmailAttachment(files, 0); err != nil {
		t.Fatalf("unset limit must fall back to 50MB, got %v", err)
	}
}
