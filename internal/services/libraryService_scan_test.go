package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSettledFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * inboxSettleDelay)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
}

func isEpub(name string) bool { return strings.HasSuffix(name, ".epub") }

func TestCollectInboxFiles(t *testing.T) {
	root := t.TempDir()

	writeSettledFile(t, filepath.Join(root, "flat.epub"))
	writeSettledFile(t, filepath.Join(root, "Series", "vol1.epub"))
	writeSettledFile(t, filepath.Join(root, "Series", "Sub", "vol2.epub"))
	writeSettledFile(t, filepath.Join(root, "notes.txt"))

	fresh := filepath.Join(root, "copying.epub")
	if err := os.WriteFile(fresh, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}

	deep := root
	for i := 0; i < inboxMaxDepth+1; i++ {
		deep = filepath.Join(deep, "d")
	}
	writeSettledFile(t, filepath.Join(deep, "toodeep.epub"))

	got := map[string]bool{}
	for _, p := range collectInboxFiles(root, 0, isEpub) {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatal(err)
		}
		got[filepath.ToSlash(rel)] = true
	}

	for _, want := range []string{"flat.epub", "Series/vol1.epub", "Series/Sub/vol2.epub"} {
		if !got[want] {
			t.Errorf("expected %q to be collected, got %v", want, got)
		}
	}
	for _, unwanted := range []string{"notes.txt", "copying.epub"} {
		if got[unwanted] {
			t.Errorf("%q should not have been collected", unwanted)
		}
	}
	if len(got) != 3 {
		t.Errorf("expected exactly 3 files (deep one excluded), got %v", got)
	}
}

func TestCollectInboxFilesSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeSettledFile(t, filepath.Join(outside, "escaped.epub"))

	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "escaped.epub"), filepath.Join(root, "escaped.epub")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if files := collectInboxFiles(root, 0, isEpub); len(files) != 0 {
		t.Errorf("symlinks must not be followed out of the inbox, got %v", files)
	}
}

func TestPruneEmptyInboxDirs(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "empty", "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	writeSettledFile(t, filepath.Join(root, "kept", "leftover.txt"))

	pruneEmptyInboxDirs(root, 0)

	if _, err := os.Stat(root); err != nil {
		t.Errorf("library root must never be removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "empty")); !os.IsNotExist(err) {
		t.Errorf("empty subtree should have been pruned, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "kept", "leftover.txt")); err != nil {
		t.Errorf("non-empty folder must be kept: %v", err)
	}
}
