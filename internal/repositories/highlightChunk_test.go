package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// GetByChapter caches the whole id list for a chapter and feeds it straight into GetHighlightsByIDs -> "id IN (?,?,...)".
func TestGetHighlightsByIDsBeyondBindParameterLimit(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "highlights.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id, name) VALUES ('lib','L')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, library_id, title) VALUES ('book','lib','B')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, email) VALUES ('user','u@n.h')`); err != nil {
		t.Fatal(err)
	}

	const n = 33000
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare(`INSERT INTO highlights (id, user_id, book_id, chapter_id, text_content, start_index, end_index) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		if _, err := stmt.Exec(fmt.Sprintf("h%06d", i), "user", "book", "chapter-1", "text", i, i+5); err != nil {
			t.Fatal(err)
		}
	}
	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	repo := NewHighlightRepository(db, cache.NewRamCache())
	ctx := context.Background()

	ids := make([]string, n)
	for i := range n {
		ids[i] = fmt.Sprintf("h%06d", i)
	}
	got, err := repo.GetHighlightsByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetHighlightsByIDs failed with %d ids: %v", n, err)
	}
	if len(got) != n {
		t.Fatalf("got %d highlights, want %d", len(got), n)
	}
}
