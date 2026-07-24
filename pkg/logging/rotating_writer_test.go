package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingWriterKeepsBoundedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novelhub.log")
	writer, err := NewRotatingWriter(path, 8, 2)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if _, err := writer.Write([]byte("12345678")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"novelhub.log", "novelhub.log.1", "novelhub.log.2"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatalf("unexpected extra rotated file: %v", err)
	}
}
