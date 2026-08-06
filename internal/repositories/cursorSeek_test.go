package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

func seekSeed(tb testing.TB, books int) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tb.TempDir(), "seek.db"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','lib-1')`); err != nil {
		tb.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	st, err := tx.Prepare(`INSERT INTO books (id,library_id,title,status,created_at) VALUES (?,?,?,'ready',datetime('now',?))`)
	if err != nil {
		tb.Fatal(err)
	}
	for i := 0; i < books; i++ {
		if _, err := st.Exec(fmt.Sprintf("b-%07d", i), "lib-1", fmt.Sprintf("t-%07d", i), fmt.Sprintf("-%d seconds", books-i)); err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	return db
}

// The cursor predicates compare the bare column against a text parameter. If either side
// changes format the comparison does not error, it silently matches the wrong rows: an
// RFC3339 cursor ('T' > ' ') matches everything and every page comes back as page 1, which
// looks like working pagination until the caller notices infinite scroll never ends.
func TestCursorPagesForwardAndDoesNotRepeat(t *testing.T) {
	db := seekSeed(t, 500)
	repo := NewBookDBRepository(db, nil).(*bookDBRepository)
	ctx := context.Background()

	seen := map[string]int{}
	var cursor *time.Time
	cursorID := ""
	pages := 0
	for {
		ids, err := repo.ListBookIDs(ctx, cursor, cursorID, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) == 0 {
			break
		}
		pages++
		if pages > 10 {
			t.Fatalf("pagination did not terminate after %d pages; cursor is not advancing", pages)
		}
		for _, id := range ids {
			seen[id]++
			if seen[id] > 1 {
				t.Fatalf("page %d returned %s again; the cursor comparison is matching rows it already served", pages, id)
			}
		}
		last := ids[len(ids)-1]
		book, err := repo.GetBook(ctx, last)
		if err != nil {
			t.Fatal(err)
		}
		created := book.CreatedAt
		cursor = &created
		cursorID = last
	}
	if len(seen) != 500 {
		t.Fatalf("paged %d of 500 books across %d pages; rows were skipped", len(seen), pages)
	}
	if pages != 5 {
		t.Fatalf("expected 5 pages of 100, got %d", pages)
	}
}

// A keyset cursor must seek the index, not scan to the offset and skip. Scanning costs the
// same per page at 4k and reveals itself only deeper in, so the guard compares a page near
// the end of a 4x larger table against the same page position in the smaller one.
func TestCursorDeepPageDoesNotScaleWithTableSize(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds books")
	}
	ctx := context.Background()
	var small time.Duration
	for _, books := range []int{10000, 40000} {
		db := seekSeed(t, books)
		repo := NewBookDBRepository(db, nil).(*bookDBRepository)

		var lastID string
		var lastRaw string
		if err := db.QueryRow(`SELECT id, created_at FROM books ORDER BY created_at DESC, id DESC LIMIT 1 OFFSET ?`, books-200).Scan(&lastID, &lastRaw); err != nil {
			t.Fatal(err)
		}
		cursor, err := time.Parse(time.RFC3339, lastRaw)
		if err != nil {
			t.Fatal(err)
		}

		ids, err := repo.ListBookIDs(ctx, &cursor, lastID, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 100 {
			t.Fatalf("deep page returned %d rows, want 100", len(ids))
		}

		d := facetTime(t, 20, func() error {
			_, err := repo.ListBookIDs(ctx, &cursor, lastID, 100)
			return err
		})
		t.Logf("books=%-6d deep page=%v", books, d)
		if small == 0 {
			small = d
			continue
		}
		if d > 2*small+200*time.Microsecond {
			t.Errorf("REGRESSION: deep page cost %v at %d books vs %v at 10000; the cursor is scanning to the offset instead of seeking the index", d, books, small)
		}
	}
}
