package localfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBookFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "novelhub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origEnv := os.Getenv("DATA_DIR")
	defer os.Setenv("DATA_DIR", origEnv)
	os.Setenv("DATA_DIR", tempDir)

	bookID := "test-book-uuid"
	filename := "book.epub"

	localBookDir := filepath.Join(tempDir, "books", bookID)
	if err := os.MkdirAll(localBookDir, 0755); err != nil {
		t.Fatalf("failed to create local book dir: %v", err)
	}
	localFile := filepath.Join(localBookDir, filename)
	if err := os.WriteFile(localFile, []byte("epub content"), 0644); err != nil {
		t.Fatalf("failed to write local file: %v", err)
	}

	tests := []struct {
		name    string
		rawPath string
		want    string
	}{
		{
			name:    "Windows absolute path on Linux (migrated)",
			rawPath: `C:\Users\Admin\NovelHub\data\books\test-book-uuid\book.epub`,
			want:    localFile,
		},
		{
			name:    "Linux absolute path (migrated from another directory)",
			rawPath: `/home/user/NovelHub/data/books/test-book-uuid/book.epub`,
			want:    localFile,
		},
		{
			name:    "Already correct local path",
			rawPath: localFile,
			want:    localFile,
		},
		{
			name:    "Empty path",
			rawPath: "",
			want:    "",
		},
		{
			name:    "Non-existent path with no file",
			rawPath: "invalid/path/file.epub",
			want:    filepath.Join(tempDir, "books", bookID, "file.epub"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveBookFilePath(bookID, tt.rawPath)
			gotClean := filepath.Clean(got)
			wantClean := filepath.Clean(tt.want)
			if gotClean != wantClean {
				t.Errorf("ResolveBookFilePath() = %v, want %v", gotClean, wantClean)
			}
		})
	}
}
