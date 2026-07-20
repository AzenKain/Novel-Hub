package bookparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataSidecarRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.txt")
	if err := os.WriteFile(path, []byte("body"), 0600); err != nil {
		t.Fatalf("write book: %v", err)
	}
	if err := SaveMetadataSidecar(path, &BookMetadata{
		Title:       "Sidecar Title",
		Author:      "Sidecar Author",
		Description: "Sidecar Description",
	}); err != nil {
		t.Fatalf("SaveMetadataSidecar: %v", err)
	}
	meta := MergeMetadataSidecar(path, &BookMetadata{Title: "Original"})
	if meta.Title != "Sidecar Title" || meta.Author != "Sidecar Author" || meta.Description != "Sidecar Description" {
		t.Fatalf("unexpected merged metadata: %#v", meta)
	}
}
