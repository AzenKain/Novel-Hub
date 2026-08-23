package services

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

func newVBookAudioTestSvc(t *testing.T) (*sql.DB, VBookService, cache.Cache) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "vbook-audio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib-1','Lib')`); err != nil {
		t.Fatal(err)
	}

	ramCache := cache.NewRamCache()
	bookRepo := repositories.NewBookDBRepository(db, ramCache)
	roleRepo := repositories.NewRoleRepository(db, ramCache)
	permissionCache := NewPermissionCache(roleRepo)
	if err := permissionCache.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	settingsService := NewSettingsService(repositories.NewSettingsRepository(db, ramCache), database.NewTxManager(db), permissionCache)
	bookService := NewBookService(bookRepo, nil, nil, nil, nil, database.NewTxManager(db), settingsService, permissionCache, nil, nil)
	audiobookRepo := repositories.NewAudiobookRepository(db, ramCache)
	svc := NewVBookService(bookRepo, bookRepo, audiobookRepo, bookService, nil, ramCache)
	return db, svc, ramCache
}

// seedAudioChapters inserts a book with the given (title, fileID) chapter pairs in order.
func seedAudioChapters(t *testing.T, db *sql.DB, bookID string, chapters [][2]string) {
	t.Helper()
	for i, ch := range chapters {
		title, fileID := ch[0], ch[1]
		var fileIDVal any
		if fileID != "" {
			// book_files must exist for the FK on file_id; insert ignores duplicates (same file reused across runs)
			if _, err := db.Exec(
				`INSERT OR IGNORE INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES (?, ?, ?, ?, 1, ?)`,
				fileID, bookID, "/tmp/"+bookID+"-"+string(rune('0'+i))+".mp3", "mp3", time.Now().UTC(),
			); err != nil {
				t.Fatal(err)
			}
			fileIDVal = fileID
		}
		if _, err := db.Exec(
			`INSERT INTO audiobook_chapters (id, book_id, file_id, chapter_index, title, start_sec) VALUES (?, ?, ?, ?, ?, 0)`,
			bookID+"-ch"+string(rune('0'+i)), bookID, fileIDVal, i, title,
		); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVBookAudioPlaylistGroupsRuns(t *testing.T) {
	db, svc, _ := newVBookAudioTestSvc(t)
	ctx := context.Background()

	if _, err := db.Exec(
		`INSERT INTO books (id, library_id, title, status, updated_at) VALUES ('audio-1', 'lib-1', 'Audio Book', 'active', datetime('now'))`,
	); err != nil {
		t.Fatal(err)
	}
	// Single-chapter run, multi-chapter run on the same file, then a gap run (nil file)
	seedAudioChapters(t, db, "audio-1", [][2]string{
		{"Chương 1", "f1"},
		{"Chương 2", "f1"},
		{"Chương 3", "f2"},
		{"Chương 4", ""}, // skipped: no file
		{"Chương 5", "f3"},
		{"Chương 6", "f1"}, // new run for the same file
	})

	tracks, err := svc.GetAudioPlaylist(ctx, "audio-1", nil)
	if err != nil {
		t.Fatalf("GetAudioPlaylist failed: %v", err)
	}
	if len(tracks) != 4 {
		t.Fatalf("got %d tracks, want 4", len(tracks))
	}

	want := []struct {
		name, desc string
		fileID     string
	}{
		{"Chương 1 (+1)", "2 chương", "f1"},
		{"Chương 3", "", "f2"},
		{"Chương 5", "", "f3"},
		{"Chương 6", "", "f1"},
	}
	for i, w := range want {
		tr := tracks[i]
		if tr.Name != w.name {
			t.Errorf("track %d name = %q, want %q", i, tr.Name, w.name)
		}
		if tr.Description != w.desc {
			t.Errorf("track %d description = %q, want %q", i, tr.Description, w.desc)
		}
		if !strings.Contains(tr.URL, "book_id=audio-1") || !strings.Contains(tr.URL, "file_id="+w.fileID) {
			t.Errorf("track %d url = %q, want book_id=audio-1 and file_id=%s", i, tr.URL, w.fileID)
		}
	}
}

func TestVBookAudioPlaylistEmptyIsNotFound(t *testing.T) {
	db, svc, _ := newVBookAudioTestSvc(t)
	ctx := context.Background()
	if _, err := db.Exec(
		`INSERT INTO books (id, library_id, title, status) VALUES ('audio-empty', 'lib-1', 'Empty', 'active')`,
	); err != nil {
		t.Fatal(err)
	}
	// chapters exist but all are nil-file (never grouped into a track)
	seedAudioChapters(t, db, "audio-empty", [][2]string{{"Chương 1", ""}, {"Chương 2", ""}})

	if _, err := svc.GetAudioPlaylist(ctx, "audio-empty", nil); err == nil {
		t.Fatal("GetAudioPlaylist: want ErrNotFound, got nil")
	}
}

func TestVBookAudioBooksCursorPagination(t *testing.T) {
	db, svc, _ := newVBookAudioTestSvc(t)
	ctx := context.Background()

	// 5 books with audio chapters, distinct updated_at so cursor ordering is stable
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		id := "audio-" + string(rune('A'+i))
		if _, err := db.Exec(
			`INSERT INTO books (id, library_id, title, status, updated_at) VALUES (?, 'lib-1', ?, 'active', ?)`,
			id, "Book "+id, now.Add(time.Duration(-i)*time.Hour).Format("2006-01-02 15:04:05"),
		); err != nil {
			t.Fatal(err)
		}
		seedAudioChapters(t, db, id, [][2]string{{"Chương 1", id + "-f1"}})
	}

	// Page 1: limit 2
	page1, err := svc.GetAudioBooks(ctx, "http://localhost", "1", 2, nil)
	if err != nil {
		t.Fatalf("GetAudioBooks page 1 failed: %v", err)
	}
	if len(page1.List) != 2 {
		t.Fatalf("page 1 len = %d, want 2", len(page1.List))
	}
	if page1.Next == nil || !strings.Contains(*page1.Next, "|") {
		t.Fatalf("page 1 Next = %v, want cursor", page1.Next)
	}

	// Page 2 via cursor
	page2, err := svc.GetAudioBooks(ctx, "http://localhost", *page1.Next, 2, nil)
	if err != nil {
		t.Fatalf("GetAudioBooks page 2 failed: %v", err)
	}
	if len(page2.List) != 2 {
		t.Fatalf("page 2 len = %d, want 2", len(page2.List))
	}
	if page2.Next == nil {
		t.Fatal("page 2 Next = nil, want cursor")
	}

	// Page 3 via cursor: 1 book left, no Next
	page3, err := svc.GetAudioBooks(ctx, "http://localhost", *page2.Next, 2, nil)
	if err != nil {
		t.Fatalf("GetAudioBooks page 3 failed: %v", err)
	}
	if len(page3.List) != 1 {
		t.Fatalf("page 3 len = %d, want 1", len(page3.List))
	}
	if page3.Next != nil {
		t.Fatalf("page 3 Next = %v, want nil", *page3.Next)
	}
}