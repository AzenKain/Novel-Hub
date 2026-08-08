package services

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/database"
)

func zipSeed(t *testing.T, books, filesPer int) (*libraryService, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "zip.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','lib-1')`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	insB, err := tx.Prepare(`INSERT INTO books (id,library_id,title,status,created_at) VALUES (?,?,?,'ready',datetime('now',?))`)
	if err != nil {
		t.Fatal(err)
	}
	insF, err := tx.Prepare(`INSERT INTO book_files (id,book_id,path,format,size_bytes,mod_time,state) VALUES (?,?,?,?,?,CURRENT_TIMESTAMP,?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < books; i++ {
		bid := fmt.Sprintf("b-%05d", i)
		if _, err := insB.Exec(bid, "lib-1", fmt.Sprintf("Title/%05d", i), fmt.Sprintf("-%d seconds", books-i)); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < filesPer; j++ {
			p := filepath.Join(dir, fmt.Sprintf("%s-%d.epub", bid, j))
			body := fmt.Appendf(nil, "body %s %d", bid, j)
			if err := os.WriteFile(p, body, 0o640); err != nil {
				t.Fatal(err)
			}
			state := "managed"
			if j == filesPer-1 && i%7 == 0 {
				state = "deleted"
			}
			if _, err := insF.Exec(fmt.Sprintf("f-%s-%d", bid, j), bid, p, "epub", len(body), state); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return &libraryService{
		libraryRepo: repositories.NewLibraryRepository(db, nil),
		bookRepo:    repositories.NewBookDBRepository(db, nil),
	}, dir
}

func zipEntries(t *testing.T, buf *bytes.Buffer) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, f.Name+"|"+string(b))
	}
	return out
}

// StreamLibraryZip batches file lookups per 100-book page. The batch must return the same
// entries the per-book loop did: every non-deleted file, the "/"-escaped title as the name,
// and no book dropped because its files landed in a different map bucket.
func TestStreamLibraryZipCoversEveryFile(t *testing.T) {
	svc, _ := zipSeed(t, 250, 3)
	var buf bytes.Buffer
	if err := svc.StreamLibraryZip(context.Background(), "lib-1", &buf); err != nil {
		t.Fatal(err)
	}
	got := zipEntries(t, &buf)

	wantCount := 0
	for i := 0; i < 250; i++ {
		wantCount += 3
		if i%7 == 0 {
			wantCount--
		}
	}
	if len(got) != wantCount {
		t.Fatalf("zip holds %d entries, want %d; batched lookup dropped files", len(got), wantCount)
	}
	for _, e := range got {
		if len(e) == 0 || e[0] == '/' {
			t.Fatalf("entry name is not escaped: %q", e)
		}
	}
	seen := map[string]int{}
	for _, e := range got {
		seen[e]++
	}
	if len(seen) != len(got) {
		t.Fatalf("%d duplicate entries; a book was zipped twice", len(got)-len(seen))
	}
}

func TestStreamLibraryZipMissingLibrary(t *testing.T) {
	svc, _ := zipSeed(t, 1, 1)
	var buf bytes.Buffer
	if err := svc.StreamLibraryZip(context.Background(), "nope", &buf); err == nil {
		t.Fatal("expected an error for an unknown library")
	}
}
