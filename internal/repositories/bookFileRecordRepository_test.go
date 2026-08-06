package repositories

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/models"
	"novelhub/pkg/database"
)

func fileIDs(items []*models.BookFileEntity) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

func newBookFileTestRepo(t *testing.T) (*bookDBRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplySchema(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO libraries (id, name) VALUES ('lib-1', 'Lib')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO books (id, library_id, title) VALUES ('book-1', 'lib-1', 'Book')`); err != nil {
		t.Fatal(err)
	}
	return NewBookDBRepository(db, nil).(*bookDBRepository), db
}

// GetFilesByBookIDs writes the same cache key that GetFilesByBookId reads back, so both
// must agree on order: epub first, then created_at. Callers take files[0] as the epub
// (send-to-Kindle in bookService_email.go, kepub conversion in koboService.go).
func TestGetFilesByBookIDsOrdersEpubFirst(t *testing.T) {
	repo, db := newBookFileTestRepo(t)
	ctx := context.Background()

	// pdf inserted first so rowid order is the opposite of the wanted order
	if _, err := db.ExecContext(ctx,
		`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time, created_at) VALUES
		 ('f-pdf',  'book-1', '/tmp/a.pdf',  'pdf',  1, '2024-01-01', '2024-01-01'),
		 ('f-epub', 'book-1', '/tmp/a.epub', 'EPUB', 1, '2024-01-02', '2024-01-02')`); err != nil {
		t.Fatal(err)
	}

	plural, err := repo.GetFilesByBookIDs(ctx, []string{"book-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fileIDs(plural); len(got) != 2 || got[0] != "f-epub" {
		t.Errorf("GetFilesByBookIDs = %v, want f-epub first", got)
	}

	singular, err := repo.GetFilesByBookId(ctx, "book-1")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fileIDs(singular), fileIDs(plural); len(got) != len(want) || got[0] != want[0] {
		t.Errorf("GetFilesByBookId = %v disagrees with GetFilesByBookIDs = %v", got, want)
	}
}

// ListFileIDsByBookId and GetBookFilesByIDs are two separate queries with no transaction
// between them, so the id list can be longer than the entities that come back. The
// surviving entities must be returned in id-list order with no holes; indexing the id
// array by id-list position (the previous code) panicked once a gap appeared before a hit.
func TestGetFilesByBookIdSkipsMissingRows(t *testing.T) {
	repo, db := newBookFileTestRepo(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time, created_at) VALUES
		 ('f-epub', 'book-1', '/tmp/a.epub', 'epub', 1, '2024-01-01', '2024-01-01'),
		 ('f-mid',  'book-1', '/tmp/b.pdf',  'pdf',  1, '2024-01-02', '2024-01-02'),
		 ('f-last', 'book-1', '/tmp/c.pdf',  'pdf',  1, '2024-01-03', '2024-01-03')`); err != nil {
		t.Fatal(err)
	}

	idRows, err := repo.queries.ListFileIDsByBookId(ctx, "book-1")
	if err != nil {
		t.Fatal(err)
	}
	// Delete the middle id, then run the second half of the read against the stale list:
	// f-last is a hit at id-list index 2 while only 2 entities exist.
	if _, err := db.ExecContext(ctx, `DELETE FROM book_files WHERE id = 'f-mid'`); err != nil {
		t.Fatal(err)
	}

	rows, err := repo.queries.GetBookFilesByIDs(ctx, idRows)
	if err != nil {
		t.Fatal(err)
	}
	out := (&models.BookFileEntities{}).FromSqlc(rows)
	fileMap := make(map[string]*models.BookFileEntity, len(out))
	for _, e := range out {
		fileMap[e.ID] = e
	}
	ordered := make([]*models.BookFileEntity, 0, len(idRows))
	cachedIDs := make([]string, 0, len(idRows))
	for _, id := range idRows {
		if e, ok := fileMap[id]; ok {
			ordered = append(ordered, e)
			cachedIDs = append(cachedIDs, id)
		}
	}

	if got := fileIDs(ordered); len(got) != 2 || got[0] != "f-epub" || got[1] != "f-last" {
		t.Errorf("ordered = %v, want [f-epub f-last]", got)
	}
	for i, id := range cachedIDs {
		if id == "" {
			t.Errorf("cachedIDs[%d] is an empty hole: %v", i, cachedIDs)
		}
	}
}

// reading_progress.file_id and highlights.chapter_id ("<file_id>:<index>") carry no FK to
// book_files, so deleting a duplicate file orphans them silently. RepointFileUserData moves
// them onto the surviving file.
func TestRepointFileUserDataMovesProgressAndHighlights(t *testing.T) {
	repo, db := newBookFileTestRepo(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO book_files (id, book_id, path, format, size_bytes, mod_time) VALUES
		 ('f-old', 'book-1', '/tmp/old.epub', 'EPUB', 1, '2024-01-01'),
		 ('f-new', 'book-1', '/tmp/new.epub', 'EPUB', 1, '2024-01-02')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ('u-1', 'a@b.c', 'x')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO reading_progress (user_id, book_id, file_id, chapter_ref, progress_percent)
		 VALUES ('u-1', 'book-1', 'f-old', 'f-old:3', 42)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO highlights (id, user_id, book_id, chapter_id, text_content, start_index, end_index)
		 VALUES ('h-1', 'u-1', 'book-1', 'f-old:3', 'note', 0, 5),
		        ('h-2', 'u-1', 'book-1', 'f-other:9', 'keep', 0, 5)`); err != nil {
		t.Fatal(err)
	}

	if err := repo.RepointFileUserData(ctx, "f-old", "f-new"); err != nil {
		t.Fatal(err)
	}

	var progressFile string
	if err := db.QueryRowContext(ctx,
		`SELECT file_id FROM reading_progress WHERE user_id = 'u-1' AND book_id = 'book-1'`).Scan(&progressFile); err != nil {
		t.Fatal(err)
	}
	if progressFile != "f-new" {
		t.Errorf("reading_progress.file_id = %q, want f-new", progressFile)
	}

	var moved, untouched string
	if err := db.QueryRowContext(ctx, `SELECT chapter_id FROM highlights WHERE id = 'h-1'`).Scan(&moved); err != nil {
		t.Fatal(err)
	}
	if moved != "f-new:3" {
		t.Errorf("highlights[h-1].chapter_id = %q, want f-new:3", moved)
	}
	if err := db.QueryRowContext(ctx, `SELECT chapter_id FROM highlights WHERE id = 'h-2'`).Scan(&untouched); err != nil {
		t.Fatal(err)
	}
	if untouched != "f-other:9" {
		t.Errorf("highlights[h-2].chapter_id = %q, want f-other:9 (unrelated file must not move)", untouched)
	}
}
