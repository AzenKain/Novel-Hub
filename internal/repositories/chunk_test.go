package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// StreamLibraryZip passes limit=1000000 to SearchBooks, which feeds every returned id
// into GetBooksByIDs -> "id IN (?,?,...)". SQLite caps bound parameters, so a large
// library makes the export fail outright rather than stream.
func TestExportManyBooksInClause(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	repo := NewBookDBRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib1','L')`); err != nil {
		t.Fatal(err)
	}
	const n = 40000
	tx, _ := db.Begin()
	st, err := tx.Prepare(`INSERT INTO books (id, library_id, title) VALUES (?,?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := st.Exec(fmt.Sprintf("b%06d", i), "lib1", fmt.Sprintf("T%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	st.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	lib := "lib1"
	books, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "", "", 1000000, "")
	if err != nil {
		t.Fatalf("export-style SearchBooks failed with %d books: %v", n, err)
	}
	if len(books) != n {
		t.Fatalf("got %d books, want %d", len(books), n)
	}
}

// ListChaptersByBook caches every chapter id of a book and feeds the whole list into
// GetChaptersByIDs -> "id IN (?,?,...)". A long web novel exceeds SQLite's bound-parameter
// cap, so opening its table of contents fails.
func TestListChaptersByBookLongNovel(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	repo := NewBookDBRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib1','L')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES ('b1','lib1','Long Novel')`); err != nil {
		t.Fatal(err)
	}
	const n = 2000
	tx, _ := db.Begin()
	st, err := tx.Prepare(`INSERT INTO chapters (id, book_id, title, chapter_index) VALUES (?,?,?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if _, err := st.Exec(fmt.Sprintf("c%05d", i), "b1", fmt.Sprintf("Ch %d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	st.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	chapters, err := repo.ListChaptersByBook(ctx, "b1")
	if err != nil {
		t.Fatalf("ListChaptersByBook failed for a %d-chapter novel: %v", n, err)
	}
	if len(chapters) != n {
		t.Fatalf("got %d chapters, want %d", len(chapters), n)
	}
	// Reading order must survive batching.
	for i, ch := range chapters {
		if int(ch.ChapterIndex) != i {
			t.Fatalf("chapter %d out of order: index %d", i, ch.ChapterIndex)
		}
	}
}

// Two books inserted in the same second share created_at exactly (SQLite
// CURRENT_TIMESTAMP is second-resolution). With the old cursor (created_at < cursor),
// page 2 dropped every book sharing the last row's created_at. The id tiebreaker
// (created_at = cursor AND id < cursor_id) must surface them on the next page.
func TestSearchBooksCursorTiebreaker(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	repo := NewBookDBRepository(db, cache.NewRamCache())
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib1','L')`); err != nil {
		t.Fatal(err)
	}
	// Insert 5 books sharing one created_at, stored as SQLite CURRENT_TIMESTAMP stores it
	// ("YYYY-MM-DD HH:MM:SS" space format). Inserting a Go time.Time instead would persist
	// "…T…Z", which datetime() cannot parse — so the test would not reproduce production.
	tx, _ := db.Begin()
	st, err := tx.Prepare(`INSERT INTO books (id, library_id, title, created_at) VALUES (?,?,?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	const ts = "2026-08-03 14:59:21"
	ids := []string{"b5", "b4", "b3", "b2", "b1"}
	for _, id := range ids {
		if _, err := st.Exec(id, "lib1", "T "+id, ts); err != nil {
			t.Fatal(err)
		}
	}
	st.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	lib := "lib1"
	// Page 1: limit 3 → newest 3 by (created_at DESC, id DESC): b5, b4, b3
	page1, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "", "", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 3 {
		t.Fatalf("page1 len = %d, want 3", len(page1))
	}
	last := page1[len(page1)-1]
	cursorTime := last.CreatedAt
	cursorID := last.ID

	// Page 2 from the tie point must return the remaining b2, b1 — not empty.
	cursorStr := cursorTime.Format(time.RFC3339Nano) + "|" + cursorID
	page2, err := repo.SearchBooks(ctx, &lib, nil, "", "", "", "", "", "", cursorStr, 3, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range page2 {
		t.Logf("page2: id=%s created_at=%v", b.ID, b.CreatedAt)
	}
	if len(page2) != 2 {
		t.Fatalf("page2 len = %d, want 2 (b2,b1 survived the tie) — cursor dropped them", len(page2))
	}
}
