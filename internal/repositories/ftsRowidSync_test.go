package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"novelhub/pkg/database"
)

// The FTS tables address rows by rowid, aligned with the source table's rowid. Nothing enforces
// that alignment -- a trigger that writes the wrong rowid, or an INSERT whose source row does
// not exist yet, silently produces an index that is wrong rather than an error. These assert the
// alignment holds through the operations that move rows around.
func ftsTestDB(tb testing.TB) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", filepath.Join(tb.TempDir(), "fts.db")+"?_pragma=foreign_keys(ON)&_pragma=trusted_schema(OFF)")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		tb.Fatal(err)
	}
	return db
}

func assertFTSAligned(tb testing.TB, db *sql.DB, stage string) {
	tb.Helper()
	checks := []struct {
		name  string
		query string
	}{
		{"mismatch", `SELECT COUNT(*) FROM fts_metadata f JOIN books b ON b.rowid = f.rowid WHERE b.id <> f.book_id`},
		{"orphan", `SELECT COUNT(*) FROM fts_metadata f WHERE NOT EXISTS (SELECT 1 FROM books b WHERE b.rowid = f.rowid)`},
		{"missing", `SELECT COUNT(*) FROM books b WHERE NOT EXISTS (SELECT 1 FROM fts_metadata f WHERE f.rowid = b.rowid)`},
	}
	for _, c := range checks {
		var n int
		if err := db.QueryRow(c.query).Scan(&n); err != nil {
			tb.Fatalf("%s: %s check failed: %v", stage, c.name, err)
		}
		if n != 0 {
			tb.Fatalf("%s: %d rows %s between books and fts_metadata", stage, n, c.name)
		}
	}
}

func TestFTSMetadataStaysAlignedThroughLifecycle(t *testing.T) {
	db := ftsTestDB(t)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	mustExec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`)
	mustExec(`INSERT INTO authors (id,name) VALUES ('au-1','Akira Toriyama')`)
	mustExec(`INSERT INTO series (id,name) VALUES ('sr-1','Dragon Ball')`)
	mustExec(`INSERT INTO tags (id,name) VALUES ('tg-1','shounen')`)

	for i := 1; i <= 5; i++ {
		mustExec(`INSERT INTO books (id,library_id,title,author_id,status) VALUES (?,?,?,'au-1','active')`,
			fmt.Sprintf("bk-%d", i), "lib-1", fmt.Sprintf("Volume %d", i))
	}
	assertFTSAligned(t, db, "after inserts")

	matches := func(term string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM fts_metadata WHERE fts_metadata MATCH ?`, term).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	mustExec(`INSERT INTO book_series (book_id,series_id,series_index) VALUES ('bk-1','sr-1','1')`)
	mustExec(`INSERT INTO book_tags (book_id,tag_id) VALUES ('bk-1','tg-1')`)
	assertFTSAligned(t, db, "after linking series and tag")
	if n := matches("Dragon"); n != 1 {
		t.Fatalf("series link should be searchable, got %d matches", n)
	}
	if n := matches("shounen"); n != 1 {
		t.Fatalf("tag link should be searchable, got %d matches", n)
	}

	mustExec(`UPDATE authors SET name = 'Toriyama Akira' WHERE id = 'au-1'`)
	assertFTSAligned(t, db, "after author rename")
	if n := matches("Toriyama"); n != 5 {
		t.Fatalf("author rename should fan out to all 5 books, got %d", n)
	}

	mustExec(`UPDATE books SET title = 'Renamed Volume' WHERE id = 'bk-2'`)
	assertFTSAligned(t, db, "after book update")
	if n := matches("Renamed"); n != 1 {
		t.Fatalf("book rename should be searchable, got %d", n)
	}

	mustExec(`DELETE FROM book_series WHERE book_id = 'bk-1'`)
	assertFTSAligned(t, db, "after unlinking series")
	if n := matches("Dragon"); n != 0 {
		t.Fatalf("unlinked series should disappear, got %d", n)
	}

	mustExec(`DELETE FROM books WHERE id = 'bk-3'`)
	assertFTSAligned(t, db, "after deleting a book")

	mustExec(`INSERT INTO books (id,library_id,title,status) VALUES ('bk-6','lib-1','Sixth','active')`)
	assertFTSAligned(t, db, "after inserting into the gap")

	mustExec(`DELETE FROM books`)
	assertFTSAligned(t, db, "after deleting everything")
	var left int
	if err := db.QueryRow(`SELECT COUNT(*) FROM fts_metadata`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("%d fts rows survived deleting every book", left)
	}

	for i := 10; i <= 12; i++ {
		mustExec(`INSERT INTO books (id,library_id,title,status) VALUES (?,'lib-1','Refill','active')`, fmt.Sprintf("bk-%d", i))
	}
	assertFTSAligned(t, db, "after refilling from empty")
}

// InsertFTSChapter resolves the rowid from chapters. If it ever runs before the chapter row
// exists the subquery yields NULL, fts5 assigns an arbitrary rowid, and DeleteFTSBook can no
// longer find the row -- a leak that no error surfaces.
func TestFTSChaptersAlignToChapterRowid(t *testing.T) {
	db := ftsTestDB(t)
	ctx := context.Background()
	repo := NewBookDBRepository(db, nil)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	mustExec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`)
	mustExec(`INSERT INTO books (id,library_id,title,status) VALUES ('bk-1','lib-1','T','active')`)
	mustExec(`INSERT INTO books (id,library_id,title,status) VALUES ('bk-2','lib-1','T','active')`)

	// All chapters first, then a gap punched in the middle, and only then the FTS rows. If the
	// insert ever stopped resolving the real chapter rowid, fts5 would assign 1,2,3... which no
	// longer lines up with the surviving chapter rowids -- whereas indexing as we go would make
	// the two sequences match by accident and hide the bug.
	chapterIDs := make([]string, 0, 8)
	for i := 0; i < 4; i++ {
		for _, bookID := range []string{"bk-1", "bk-2"} {
			chapterID := fmt.Sprintf("%s-ch%d", bookID, i)
			mustExec(`INSERT INTO chapters (id,book_id,title,chapter_index) VALUES (?,?,?,?)`,
				chapterID, bookID, "Chapter", i)
			chapterIDs = append(chapterIDs, chapterID)
		}
	}
	mustExec(`DELETE FROM chapters WHERE id IN ('bk-1-ch0','bk-2-ch0','bk-1-ch1')`)

	indexed := 0
	for _, chapterID := range chapterIDs {
		var bookID string
		if err := db.QueryRow(`SELECT book_id FROM chapters WHERE id = ?`, chapterID).Scan(&bookID); err != nil {
			continue
		}
		if err := repo.InsertFTSChapter(ctx, bookID, chapterID, "Chapter", "the quick brown fox"); err != nil {
			t.Fatal(err)
		}
		indexed++
	}

	var misaligned int
	if err := db.QueryRow(`SELECT COUNT(*) FROM fts_chapters f
		WHERE NOT EXISTS (SELECT 1 FROM chapters c WHERE c.rowid = f.rowid AND c.id = f.chapter_id)`).Scan(&misaligned); err != nil {
		t.Fatal(err)
	}
	if misaligned != 0 {
		t.Fatalf("%d fts_chapters rows are not aligned to their chapter rowid", misaligned)
	}

	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM fts_chapters`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != indexed {
		t.Fatalf("indexed %d chapters but fts holds %d rows: a colliding rowid overwrote an earlier row", indexed, rows)
	}

	if err := repo.DeleteFTSBook(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}
	var left, other int
	db.QueryRow(`SELECT COUNT(*) FROM fts_chapters WHERE book_id = 'bk-1'`).Scan(&left)
	db.QueryRow(`SELECT COUNT(*) FROM fts_chapters WHERE book_id = 'bk-2'`).Scan(&other)
	if left != 0 {
		t.Fatalf("DeleteFTSBook left %d rows behind", left)
	}
	var want int
	db.QueryRow(`SELECT COUNT(*) FROM chapters WHERE book_id = 'bk-2'`).Scan(&want)
	if other != want {
		t.Fatalf("DeleteFTSBook removed rows of another book: bk-2 has %d fts rows, want %d", other, want)
	}
}

// The regression this whole change exists to prevent: addressing FTS rows by an UNINDEXED
// column makes every write a full table scan, so per-write cost grows with table size. Assert
// it stays flat when the table grows 4x.
func TestFTSWriteCostDoesNotGrowWithTableSize(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds 32k books")
	}
	perOp := make(map[int]time.Duration)
	for _, size := range []int{8000, 32000} {
		db := ftsTestDB(t)
		if _, err := db.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO series (id,name) VALUES ('sr-1','Dragon Ball')`); err != nil {
			t.Fatal(err)
		}
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		insert, err := tx.Prepare(`INSERT INTO books (id,library_id,title,status) VALUES (?,'lib-1','T','active')`)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < size; i++ {
			if _, err := insert.Exec(fmt.Sprintf("bk-%06d", i)); err != nil {
				t.Fatal(err)
			}
		}
		insert.Close()
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}

		const ops = 200
		tx2, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		link, err := tx2.Prepare(`INSERT INTO book_series (book_id,series_id,series_index) VALUES (?,'sr-1','1')`)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		for i := 0; i < ops; i++ {
			if _, err := link.Exec(fmt.Sprintf("bk-%06d", i)); err != nil {
				t.Fatal(err)
			}
		}
		elapsed := time.Since(start)
		link.Close()
		if err := tx2.Commit(); err != nil {
			t.Fatal(err)
		}

		perOp[size] = elapsed / ops
		t.Logf("books=%d: %d link-inserts in %s, per_op=%s", size, ops,
			elapsed.Round(time.Millisecond), (elapsed / ops).Round(time.Microsecond))
	}

	small, large := perOp[8000], perOp[32000]
	if large > small*2 {
		t.Fatalf("4x the rows made each write %.1fx slower (%s -> %s): FTS writes are scanning again",
			float64(large)/float64(small), small.Round(time.Microsecond), large.Round(time.Microsecond))
	}
}

// SQLite prints "SCAN <table> VIRTUAL TABLE INDEX 0:" for an fts5 write either way; the tell is
// the trailing "=", which means fts5 accepted a rowid constraint and will seek instead of walk.
// Without it the write is a full scan, which is the O(n) behaviour this change removed.
func TestFTSWritesAreRowidConstrained(t *testing.T) {
	db := ftsTestDB(t)
	cases := []struct{ name, query string }{
		{"metadata link update", `UPDATE fts_metadata SET series='x' WHERE rowid = (SELECT rowid FROM books WHERE id = 'b')`},
		{"metadata delete", `DELETE FROM fts_metadata WHERE rowid = 1`},
		{"chapters book delete", `DELETE FROM fts_chapters WHERE rowid IN (SELECT c.rowid FROM chapters c WHERE c.book_id = 'b')`},
		{"users update", `UPDATE fts_users SET haystack='x' WHERE rowid = 1`},
	}
	for _, c := range cases {
		rows, err := db.Query("EXPLAIN QUERY PLAN " + c.query)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		constrained := false
		for rows.Next() {
			var id, parent, notUsed int
			var detail string
			if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			if strings.HasPrefix(detail, "SCAN fts_") && strings.HasSuffix(detail, "=") {
				constrained = true
			}
		}
		rows.Close()
		if !constrained {
			t.Errorf("%s: fts5 got no rowid constraint, the write will scan the whole index", c.name)
		}
	}
}
